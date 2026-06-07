package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"
)

const icmTestToken = "test-icm-token"

func TestIcmMeteoGetForecast(t *testing.T) {
	const lat = 54.646034
	const lon = 18.512407

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Token "+icmTestToken {
			t.Errorf("missing/incorrect Authorization header: got %q", got)
		}

		var fixture string
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/model/coamps/grid/2a/coordinates/130,111/field/uuwind_zht_fcstfld/level/000010_000000/date/":
			fixture = "testdata/icmmeteo_dates.json"
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/model/coamps/grid/2a/coordinates/130,111/field/uuwind_zht_fcstfld/level/000010_000000/date/2024-05-24T12/forecast/":
			fixture = "testdata/icmmeteo_u.json"
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/model/coamps/grid/2a/coordinates/130,111/field/vvwind_zht_fcstfld/level/000010_000000/date/2024-05-24T12/forecast/":
			fixture = "testdata/icmmeteo_v.json"
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		content, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatalf("cannot read fixture %s: %v", fixture, err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	icm, err := NewIcmMeteo(server.URL, icmTestToken)
	if err != nil {
		t.Fatalf("NewIcmMeteo error: %v", err)
	}

	loc := Location{Lat: lat, Lon: lon}
	reading, err := icm.GetForecast(context.Background(), loc)
	if err != nil {
		t.Fatalf("GetForecast error: %v", err)
	}

	hourly, ok := reading.Readings["hourly"]
	if !ok {
		t.Fatal("expected 'hourly' key in readings")
	}

	expected := []WindDataPoint{
		{Time: time.Date(2024, 5, 24, 12, 0, 0, 0, time.UTC), WindSpeed: 5.0, WindAngle: 0.0},
		{Time: time.Date(2024, 5, 24, 13, 0, 0, 0, time.UTC), WindSpeed: 5.0, WindAngle: 90.0},
		{Time: time.Date(2024, 5, 24, 14, 0, 0, 0, time.UTC), WindSpeed: 5.0, WindAngle: 180.0},
		{Time: time.Date(2024, 5, 24, 15, 0, 0, 0, time.UTC), WindSpeed: 5.0, WindAngle: 270.0},
	}
	if !reflect.DeepEqual(hourly, expected) {
		t.Errorf("hourly mismatch:\ngot  %+v\nwant %+v", hourly, expected)
	}

	if reading.Location != loc {
		t.Errorf("location mismatch: got %+v want %+v", reading.Location, loc)
	}
}

func TestIcmMeteoWindVectorToSpeedAngle(t *testing.T) {
	tests := []struct {
		name          string
		u, v          float64
		expectedSpeed float64
		expectedAngle float64
	}{
		{"from north", 0, -5, 5, 0},
		{"from east", -5, 0, 5, 90},
		{"from south", 0, 5, 5, 180},
		{"from west", 5, 0, 5, 270},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			speed, angle := windVectorToSpeedAngle(tc.u, tc.v)
			if speed != tc.expectedSpeed {
				t.Errorf("speed: got %v want %v", speed, tc.expectedSpeed)
			}
			if angle != tc.expectedAngle {
				t.Errorf("angle: got %v want %v", angle, tc.expectedAngle)
			}
		})
	}
}
