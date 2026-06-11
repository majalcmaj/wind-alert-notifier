package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/majalcmaj/wind-alert/shared/model"
)

type OpenMeteo struct{ baseURL string }

func NewOpenMeteo(baseURL string) *OpenMeteo {
	return &OpenMeteo{baseURL: baseURL}
}

func (o *OpenMeteo) GetForecast(ctx context.Context, loc model.Location) (*WeatherReading, error) {
	url := fmt.Sprintf(
		"%s/v1/forecast?latitude=%f&longitude=%f&hourly=wind_speed_10m,wind_direction_10m&wind_speed_unit=ms&forecast_days=7",
		o.baseURL, loc.Lat, loc.Lon,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "openmeteo: cannot create request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "openmeteo: request failed")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprint(os.Stderr, errors.Errorf("openmeteo: warning closing body: %+v", err))
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "openmeteo: cannot read body")
	}

	reading, err := parseOpenMeteoResponse(body)
	if err != nil {
		return nil, err
	}
	reading.Location = loc
	return reading, nil
}

type openMeteoResponse struct {
	Hourly struct {
		Time             []string  `json:"time"`
		WindSpeed10m     []float64 `json:"wind_speed_10m"`
		WindDirection10m []float64 `json:"wind_direction_10m"`
	} `json:"hourly"`
}

const openMeteoTimeLayout = "2006-01-02T15:04"

func parseOpenMeteoResponse(content []byte) (*WeatherReading, error) {
	var resp openMeteoResponse
	if err := json.Unmarshal(content, &resp); err != nil {
		return nil, errors.Wrap(err, "openmeteo: cannot parse response")
	}

	n := len(resp.Hourly.Time)
	hourly := make([]WindDataPoint, n)
	for i := range n {
		t, err := time.Parse(openMeteoTimeLayout, resp.Hourly.Time[i])
		if err != nil {
			return nil, errors.Wrapf(err, "openmeteo: cannot parse time %q", resp.Hourly.Time[i])
		}
		hourly[i] = WindDataPoint{
			Time:      t.UTC(),
			WindSpeed: resp.Hourly.WindSpeed10m[i],
			WindAngle: resp.Hourly.WindDirection10m[i],
		}
	}

	return &WeatherReading{
		Readings: map[string][]WindDataPoint{"hourly": hourly},
	}, nil
}
