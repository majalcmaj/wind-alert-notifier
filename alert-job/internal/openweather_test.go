package internal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/majalcmaj/wind-alert/shared/model"
)

func TestMakingOpenweatherRequest(t *testing.T) {
	const lat = 33.44
	const lon = -94.04
	const token = "abcdef12345"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := fmt.Sprintf("/data/3.0/onecall?lat=%f&lon=%f&exclude=current,minutely,alerts&appid=%s", lat, lon, token)
		if r.URL.Path != "/data/3.0/onecall" {
			t.Errorf("Expected the context path %s but got %s", expectedPath, r.URL.Path)
		}

		for k, v := range map[string]string{
			"lat":     fmt.Sprintf("%f", lat),
			"lon":     fmt.Sprintf("%f", lon),
			"token":   token,
			"exclude": "current,minutely,alerts",
		} {
			if !r.URL.Query().Has(k) && r.URL.Query().Get(k) == v {
				t.Errorf("Expected query parameter %s=%s but it was not found in %s", k, v, r.URL.RawQuery)
			}
		}

		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept: application/json header, got: %s", r.Header.Get("Accept"))
		}

		w.WriteHeader(http.StatusOK)
		content, err := readTestData()
		if err != nil {
			t.Errorf("Could not read test data, err: %s", err)
		}
		_, _ = w.Write(content)
	}))
	defer server.Close()

	openWeather, err := NewOpenWeather(server.URL, token)

	if err != nil {
		t.Errorf("Could not construct the OpenWeather instance: %s", err)
	}

	loc := model.Location{Lat: lat, Lon: lon}
	response, err := openWeather.GetForecast(context.Background(), loc)
	if err != nil {
		t.Errorf("Error while calling the OpenWeather mock: %s", err)
	}

	expected := WeatherReading{
		Location: loc,
		Readings: map[string][]WindDataPoint{
			"hourly": {
				{
					Time:      time.Unix(1684926000, 0),
					WindSpeed: 2.58,
					WindAngle: 86,
				},
			},
			"daily": {
				{
					Time:      time.Unix(1684951200, 0),
					WindSpeed: 3.98,
					WindAngle: 76,
				},
			},
		},
	}
	if !reflect.DeepEqual(response, &expected) {
		t.Errorf("Expected parsed response was %+v but got %+v", &expected, response)
	}
}

func readTestData() ([]byte, error) {
	return os.ReadFile("testdata/openWeatherMap.json")
}
