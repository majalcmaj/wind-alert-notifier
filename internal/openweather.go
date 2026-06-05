package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type reading struct {
	Dt        int     `json:"dt"`
	WindSpeed float64 `json:"wind_speed"`
	WindDeg   int     `json:"wind_deg"`
}

type openWeatherResponse struct {
	Lat    float64   `json:"lat"`
	Lon    float64   `json:"lon"`
	Hourly []reading `json:"hourly"`
	Daily  []reading `json:"daily"`
}

type OpenWeather struct {
	baseUrl     string
	accessToken string
}

func NewOpenWeather(baseUrl, accessToken string) (*OpenWeather, error) {
	if strings.TrimSpace(baseUrl) == "" || strings.TrimSpace(accessToken) == "" {
		return nil, errors.Errorf("Both the baseUrl and accessToken are required")
	}
	return &OpenWeather{baseUrl, accessToken}, nil
}

func (o *OpenWeather) GetForecast(latitude, longitude float64) (*WeatherReading, error) {
	url := fmt.Sprintf("%s/data/3.0/onecall?lat=%f&lon=%f&exclude=current,minutely,alerts&appid=%s", o.baseUrl, latitude, longitude, o.accessToken)

	client := http.DefaultClient

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot create a request")
	}
	req.Header.Add("Accept", "application/json")

	response, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Could not make a request")
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			fmt.Fprint(os.Stderr, errors.Errorf("warning: error when closing the response body: %+v", err))
		}
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Could not parse response body")
	}

	return parseOpenweatherResponse(body)
}

func parseOpenweatherResponse(content []byte) (*WeatherReading, error) {
	var resp openWeatherResponse
	err := json.Unmarshal(content, &resp)
	if err != nil {
		return nil, err
	}

	return &WeatherReading{
		Lat: resp.Lat,
		Lon: resp.Lon,
		Readings: map[string][]WindDataPoint{
			"hourly": getWindDatapoints(resp.Hourly),
			"daily":  getWindDatapoints(resp.Daily),
		},
	}, nil
}

func getWindDatapoints(readings []reading) []WindDataPoint {
	result := make([]WindDataPoint, len(readings))
	for idx, r := range readings {
		result[idx] = WindDataPoint{
			Time:      time.Unix(int64(r.Dt), 0),
			WindSpeed: r.WindSpeed,
			WindAngle: float64(r.WindDeg),
		}
	}
	return result
}
