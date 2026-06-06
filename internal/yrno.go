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
)

type YrNo struct{ baseURL string }

func NewYrNo(baseURL string) *YrNo {
	return &YrNo{baseURL: baseURL}
}

func (y *YrNo) GetForecast(ctx context.Context, loc Location) (*WeatherReading, error) {
	url := fmt.Sprintf("%s/weatherapi/locationforecast/2.0/compact?lat=%f&lon=%f", y.baseURL, loc.Lat, loc.Lon)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "yrno: cannot create request")
	}
	req.Header.Set("User-Agent", "wind-alert-go/1.0 github.com/majalcmaj/wind-alert-go")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "yrno: request failed")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprint(os.Stderr, errors.Errorf("yrno: warning closing body: %+v", err))
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "yrno: cannot read body")
	}

	reading, err := parseYrNoResponse(body)
	if err != nil {
		return nil, err
	}
	reading.Location = loc
	return reading, nil
}

type yrnoResponse struct {
	Properties struct {
		Timeseries []struct {
			Time time.Time `json:"time"`
			Data struct {
				Instant struct {
					Details struct {
						WindSpeed         float64 `json:"wind_speed"`
						WindFromDirection float64 `json:"wind_from_direction"`
					} `json:"details"`
				} `json:"instant"`
			} `json:"data"`
		} `json:"timeseries"`
	} `json:"properties"`
}

func parseYrNoResponse(content []byte) (*WeatherReading, error) {
	var resp yrnoResponse
	if err := json.Unmarshal(content, &resp); err != nil {
		return nil, errors.Wrap(err, "yrno: cannot parse response")
	}

	hourly := make([]WindDataPoint, len(resp.Properties.Timeseries))
	for i, ts := range resp.Properties.Timeseries {
		hourly[i] = WindDataPoint{
			Time:      ts.Time,
			WindSpeed: ts.Data.Instant.Details.WindSpeed,
			WindAngle: ts.Data.Instant.Details.WindFromDirection,
		}
	}

	return &WeatherReading{
		Readings: map[string][]WindDataPoint{"hourly": hourly},
	}, nil
}
