package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/majalcmaj/wind-alert/shared/model"
)

func TestOpenMeteoRequest(t *testing.T) {
	const lat = 54.646034
	const lon = 18.512407

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/forecast" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		for _, param := range []string{"latitude", "longitude", "hourly", "wind_speed_unit", "forecast_days"} {
			if q.Get(param) == "" {
				t.Errorf("missing query param: %s", param)
			}
		}
		if q.Get("wind_speed_unit") != "ms" {
			t.Errorf("expected wind_speed_unit=ms, got %s", q.Get("wind_speed_unit"))
		}
		content, err := os.ReadFile("testdata/openmeteo.json")
		if err != nil {
			t.Fatalf("cannot read openmeteo fixture: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	om := NewOpenMeteo(server.URL)
	loc := model.Location{Lat: lat, Lon: lon}
	reading, err := om.GetForecast(context.Background(), loc)
	if err != nil {
		t.Fatalf("GetForecast error: %v", err)
	}

	hourly, ok := reading.Readings["hourly"]
	if !ok {
		t.Fatal("expected 'hourly' key in readings")
	}
	if len(hourly) != 2 {
		t.Fatalf("expected 2 hourly readings, got %d", len(hourly))
	}

	expected := []WindDataPoint{
		{Time: time.Date(2024, 5, 24, 12, 0, 0, 0, time.UTC), WindSpeed: 5.2, WindAngle: 270.0},
		{Time: time.Date(2024, 5, 24, 13, 0, 0, 0, time.UTC), WindSpeed: 6.1, WindAngle: 280.0},
	}
	if !reflect.DeepEqual(hourly, expected) {
		t.Errorf("hourly mismatch:\ngot  %+v\nwant %+v", hourly, expected)
	}

	if reading.Location != loc {
		t.Errorf("location mismatch: got %+v want %+v", reading.Location, loc)
	}
}
