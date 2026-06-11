package model

import (
	"fmt"
	"math"
)

type Location struct {
	ID   string  `json:"id"   dynamodbav:"id"`
	Name string  `json:"name" dynamodbav:"name"`
	Lat  float64 `json:"lat"  dynamodbav:"lat"`
	Lon  float64 `json:"lon"  dynamodbav:"lon"`
}

type Range struct {
	From float64 `json:"from" dynamodbav:"from"`
	To   float64 `json:"to"   dynamodbav:"to"`
}

type Rule struct {
	Name          string  `json:"name,omitempty"           dynamodbav:"name,omitempty"`
	LocationID    string  `json:"location_id"              dynamodbav:"location_id"`
	AngleRange    Range   `json:"angle"                    dynamodbav:"angle"`
	SpeedRange    Range   `json:"speed"                    dynamodbav:"speed"`
	HourRange     Range   `json:"hour"                     dynamodbav:"hour"`
	MinConfidence float64 `json:"min_confidence,omitempty" dynamodbav:"min_confidence,omitempty"`
}

func (rng Range) WithinCyclicRange(value float64) bool {
	if rng.From > rng.To {
		return value >= rng.From || value <= rng.To
	}
	return rng.WithinRange(value)
}

func (rng Range) WithinRange(number float64) bool {
	return number >= rng.From && number <= rng.To
}

func angleToCompass(angle float64) string {
	angle = math.Mod(angle, 360)
	switch {
	case angle >= 337.5 || angle < 22.5:
		return "N"
	case angle < 67.5:
		return "NE"
	case angle < 112.5:
		return "E"
	case angle < 157.5:
		return "SE"
	case angle < 202.5:
		return "S"
	case angle < 247.5:
		return "SW"
	case angle < 292.5:
		return "W"
	default:
		return "NW"
	}
}

func formatHour(h float64) string {
	hour := int(h)
	minute := int(math.Round((h - float64(hour)) * 60))
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

func (r Rule) Describe() string {
	if r.Name != "" {
		return r.Name
	}
	return fmt.Sprintf("wind %.1f–%.1f m/s from %s–%s (%.0f°–%.0f°), %s–%s",
		r.SpeedRange.From, r.SpeedRange.To,
		angleToCompass(r.AngleRange.From), angleToCompass(r.AngleRange.To),
		r.AngleRange.From, r.AngleRange.To,
		formatHour(r.HourRange.From), formatHour(r.HourRange.To),
	)
}
