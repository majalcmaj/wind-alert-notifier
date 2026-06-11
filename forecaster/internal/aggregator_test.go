package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/majalcmaj/wind-alert/shared/model"
)

type stubForecaster struct {
	reading *WeatherReading
	err     error
}

func (s *stubForecaster) GetForecast(_ context.Context, _ model.Location) (*WeatherReading, error) {
	return s.reading, s.err
}

func TestFetchAllReturnsAllReadings(t *testing.T) {
	r1 := &WeatherReading{Readings: map[string][]WindDataPoint{"hourly": {}}}
	r2 := &WeatherReading{Readings: map[string][]WindDataPoint{"hourly": {}}}

	providers := []NamedForecaster{
		{Name: "p1", Forecaster: &stubForecaster{reading: r1}},
		{Name: "p2", Forecaster: &stubForecaster{reading: r2}},
	}

	results := FetchAll(context.Background(), model.Location{}, providers)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Name != "p1" || results[0].Reading != r1 || results[0].Err != nil {
		t.Errorf("unexpected p1 result: %+v", results[0])
	}
	if results[1].Name != "p2" || results[1].Reading != r2 || results[1].Err != nil {
		t.Errorf("unexpected p2 result: %+v", results[1])
	}
}

func TestFetchAllPreservesOrder(t *testing.T) {
	providers := make([]NamedForecaster, 5)
	readings := make([]*WeatherReading, 5)
	for i := range providers {
		readings[i] = &WeatherReading{}
		providers[i] = NamedForecaster{
			Name:       string(rune('a' + i)),
			Forecaster: &stubForecaster{reading: readings[i]},
		}
	}

	results := FetchAll(context.Background(), model.Location{}, providers)

	for i, r := range results {
		if r.Reading != readings[i] {
			t.Errorf("index %d: wrong reading (order not preserved)", i)
		}
	}
}

func TestFetchAllRecordsErrors(t *testing.T) {
	sentinelErr := errors.New("provider down")
	r1 := &WeatherReading{}

	providers := []NamedForecaster{
		{Name: "ok", Forecaster: &stubForecaster{reading: r1}},
		{Name: "bad", Forecaster: &stubForecaster{err: sentinelErr}},
	}

	results := FetchAll(context.Background(), model.Location{}, providers)

	if results[0].Err != nil {
		t.Errorf("expected no error for ok provider, got %v", results[0].Err)
	}
	if results[1].Err != sentinelErr {
		t.Errorf("expected sentinel error for bad provider, got %v", results[1].Err)
	}
	if results[1].Reading != nil {
		t.Errorf("expected nil reading for bad provider")
	}
}
