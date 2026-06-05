package internal

import "time"

type Location struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

type WeatherReading struct {
	Location Location
	Readings map[string][]WindDataPoint
}

type WindDataPoint struct {
	Time      time.Time
	WindSpeed float64
	WindAngle float64
	Matched   bool
}

type LocationResult struct {
	Location       Location
	Reading        *WeatherReading
	TriggeredRules []string
}
