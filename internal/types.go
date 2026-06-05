package internal

import "time"

type WeatherReading struct {
	Lat      float64
	Lon      float64
	Readings map[string]*[]WindDataPoint
}

type WindDataPoint struct {
	Time      time.Time
	WindSpeed float64
	WindAngle float64
	Matched   bool
}
