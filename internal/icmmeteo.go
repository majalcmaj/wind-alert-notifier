package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
)

// Hardcoded for the Gdańsk/Sopot location until lat/lon → row/col resolution is implemented.
const (
	icmModel = "coamps"
	icmGrid  = "2a"
	icmRow   = 130
	icmCol   = 111
	icmLevel = "000010_000000"

	icmFieldU = "uuwind_zht_fcstfld"
	icmFieldV = "vvwind_zht_fcstfld"

	icmRunDateLayout = "2006-01-02T15"
)

type IcmMeteo struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewIcmMeteo(baseURL, token string) (*IcmMeteo, error) {
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(token) == "" {
		return nil, errors.Errorf("Both the baseURL and token are required")
	}

	retryClient := retryablehttp.NewClient()
	retryClient.Logger = nil

	return &IcmMeteo{baseURL: baseURL, token: token, httpClient: retryClient.StandardClient()}, nil
}

func (i *IcmMeteo) GetForecast(ctx context.Context, loc Location) (*WeatherReading, error) {
	runDate, err := i.latestRunDate(ctx, icmRow, icmCol)
	if err != nil {
		return nil, err
	}

	uTimes, uData, err := i.fetchComponent(ctx, icmRow, icmCol, icmFieldU, runDate)
	if err != nil {
		return nil, err
	}
	vTimes, vData, err := i.fetchComponent(ctx, icmRow, icmCol, icmFieldV, runDate)
	if err != nil {
		return nil, err
	}

	if len(uTimes) != len(vTimes) {
		return nil, errors.Errorf("icm: U/V forecast length mismatch (u=%d, v=%d)", len(uTimes), len(vTimes))
	}

	hourly := make([]WindDataPoint, len(uTimes))
	for idx, t := range uTimes {
		if !t.Equal(vTimes[idx]) {
			return nil, errors.Errorf("icm: U/V forecast times differ at index %d (%s vs %s)", idx, t, vTimes[idx])
		}
		speed, angle := windVectorToSpeedAngle(uData[idx], vData[idx])
		hourly[idx] = WindDataPoint{
			Time:      t,
			WindSpeed: speed,
			WindAngle: angle,
		}
	}

	return &WeatherReading{
		Location: loc,
		Readings: map[string][]WindDataPoint{"hourly": hourly},
	}, nil
}

// windVectorToSpeedAngle converts U/V wind vector components into speed and the
// meteorological "from" direction (matching wind_deg / wind_from_direction).
func windVectorToSpeedAngle(u, v float64) (speed, angle float64) {
	speed = math.Hypot(u, v)
	angle = math.Mod(math.Atan2(-u, -v)*180/math.Pi+360, 360)
	return speed, angle
}

type icmDateRange struct {
	StartingDate string `json:"starting-date"`
	Count        int    `json:"count"`
	Interval     int    `json:"interval"`
}

type icmDatesResponse struct {
	Dates []icmDateRange `json:"dates"`
}

func (i *IcmMeteo) latestRunDate(ctx context.Context, row, col int) (string, error) {
	path := fmt.Sprintf("/api/v1/model/%s/grid/%s/coordinates/%d,%d/field/%s/level/%s/date/",
		icmModel, icmGrid, row, col, icmFieldU, icmLevel)

	var resp icmDatesResponse
	if err := i.get(ctx, path, &resp); err != nil {
		return "", err
	}
	if len(resp.Dates) == 0 {
		return "", errors.New("icm: no model runs available")
	}

	last := resp.Dates[len(resp.Dates)-1]
	start, err := time.Parse(icmRunDateLayout, last.StartingDate)
	if err != nil {
		return "", errors.Wrapf(err, "icm: cannot parse run starting date %q", last.StartingDate)
	}
	latest := start.Add(time.Duration(last.Count-1) * time.Duration(last.Interval) * time.Hour)
	return latest.Format(icmRunDateLayout), nil
}

type icmForecastResponse struct {
	Times []string  `json:"times"`
	Data  []float64 `json:"data"`
}

func (i *IcmMeteo) fetchComponent(ctx context.Context, row, col int, field, date string) ([]time.Time, []float64, error) {
	path := fmt.Sprintf("/api/v1/model/%s/grid/%s/coordinates/%d,%d/field/%s/level/%s/date/%s/forecast/",
		icmModel, icmGrid, row, col, field, icmLevel, date)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, i.baseURL+path, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "icm: cannot create request")
	}
	req.Header.Set("Authorization", "Token "+i.token)

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "icm: request failed")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprint(os.Stderr, errors.Errorf("icm: warning closing body: %+v", err))
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, errors.Wrap(err, "icm: cannot read body")
	}

	if resp.StatusCode == http.StatusPaymentRequired {
		return nil, nil, errors.Errorf("icm: forecast for %q unavailable — account budget exhausted (402)", field)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, errors.Errorf("icm: forecast request for %q failed with status %d: %s", field, resp.StatusCode, string(body))
	}

	var parsed icmForecastResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, nil, errors.Wrap(err, "icm: cannot parse forecast response")
	}
	if len(parsed.Times) != len(parsed.Data) {
		return nil, nil, errors.Errorf("icm: forecast for %q has mismatched times/data lengths (%d/%d)", field, len(parsed.Times), len(parsed.Data))
	}

	times := make([]time.Time, len(parsed.Times))
	for idx, ts := range parsed.Times {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "icm: cannot parse time %q", ts)
		}
		times[idx] = t.UTC()
	}
	return times, parsed.Data, nil
}

func (i *IcmMeteo) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, i.baseURL+path, nil)
	if err != nil {
		return errors.Wrap(err, "icm: cannot create request")
	}
	req.Header.Set("Authorization", "Token "+i.token)

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "icm: request failed")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprint(os.Stderr, errors.Errorf("icm: warning closing body: %+v", err))
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "icm: cannot read body")
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("icm: request to %q failed with status %d: %s", path, resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, out); err != nil {
		return errors.Wrap(err, "icm: cannot parse response")
	}
	return nil
}
