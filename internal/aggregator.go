package internal

import (
	"context"
	"sync"
)

type Forecaster interface {
	GetForecast(ctx context.Context, loc Location) (*WeatherReading, error)
}

type NamedForecaster struct {
	Name       string
	Forecaster Forecaster
}

type ProviderReading struct {
	Name    string
	Reading *WeatherReading
	Err     error
}

func FetchAll(ctx context.Context, loc Location, providers []NamedForecaster) []ProviderReading {
	results := make([]ProviderReading, len(providers))
	var wg sync.WaitGroup
	for i, p := range providers {
		wg.Add(1)
		go func(i int, p NamedForecaster) {
			defer wg.Done()
			reading, err := p.Forecaster.GetForecast(ctx, loc)
			results[i] = ProviderReading{Name: p.Name, Reading: reading, Err: err}
		}(i, p)
	}
	wg.Wait()
	return results
}
