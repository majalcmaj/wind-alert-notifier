package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"wind-alert/internal/model"
)

func TestYrNoRequest(t *testing.T) {
	const lat = 54.646034
	const lon = 18.512407

	var capturedUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/weatherapi/locationforecast/2.0/compact" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("lat") == "" || q.Get("lon") == "" {
			t.Errorf("missing lat/lon query params in %s", r.URL.RawQuery)
		}
		capturedUserAgent = r.Header.Get("User-Agent")
		content, err := os.ReadFile("testdata/yrno.json")
		if err != nil {
			t.Fatalf("cannot read yrno fixture: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	yrno := NewYrNo(server.URL)
	loc := model.Location{Lat: lat, Lon: lon}
	reading, err := yrno.GetForecast(context.Background(), loc)
	if err != nil {
		t.Fatalf("GetForecast error: %v", err)
	}

	if capturedUserAgent == "" {
		t.Error("User-Agent header not set")
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
