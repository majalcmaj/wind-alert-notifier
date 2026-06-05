package internal

import (
	"fmt"
	"math"
)

type Range struct {
	From float64 `json:"from"`
	To   float64 `json:"to"`
}

type Rule struct {
	Name       string `json:"name,omitempty"`
	AngleRange Range  `json:"angle"`
	SpeedRange Range  `json:"speed"`
	HourRange  Range  `json:"hour"`
}

func (rng Range) withinCyclicRange(value float64) bool {
	if rng.From > rng.To {
		return value >= rng.From || value <= rng.To
	}
	return rng.withinRange(value)
}

func (rng Range) withinRange(number float64) bool {
	return number >= rng.From && number <= rng.To
}

func (r Rule) matches(dp WindDataPoint) bool {
	return r.AngleRange.withinCyclicRange(dp.WindAngle) &&
		r.SpeedRange.withinRange(dp.WindSpeed) &&
		r.HourRange.withinCyclicRange(float64(dp.Time.Hour())+float64(dp.Time.Minute())/60.0)
}

func RunRuleEngine(dataPoint WindDataPoint, rules *[]Rule) (bool, error) {
	for _, rule := range *rules {
		if rule.matches(dataPoint) {
			return true, nil
		}
	}
	return false, nil
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

func EvaluateForecast(reading *WeatherReading, rules []Rule) []Rule {
	var triggered []Rule
	for _, rule := range rules {
		matched := false
		for _, dps := range reading.Readings {
			for i := range *dps {
				dp := &(*dps)[i]
				if rule.matches(*dp) {
					dp.Matched = true
					matched = true
				}
			}
		}
		if matched {
			triggered = append(triggered, rule)
		}
	}
	return triggered
}
