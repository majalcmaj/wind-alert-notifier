package internal

import (
	"fmt"
	"math"
)

type Range struct {
	From float64 `json:"from" dynamodbav:"from"`
	To   float64 `json:"to"   dynamodbav:"to"`
}

type Rule struct {
	Name          string  `json:"name,omitempty"          dynamodbav:"name,omitempty"`
	LocationID    string  `json:"location_id"             dynamodbav:"location_id"`
	AngleRange    Range   `json:"angle"                   dynamodbav:"angle"`
	SpeedRange    Range   `json:"speed"                   dynamodbav:"speed"`
	HourRange     Range   `json:"hour"                    dynamodbav:"hour"`
	MinConfidence float64 `json:"min_confidence,omitempty" dynamodbav:"min_confidence,omitempty"`
}

type ConfidentRule struct {
	Rule
	Confidence     float64
	MatchedBy      []string
	TotalProviders int
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

func RunRuleEngine(dataPoint WindDataPoint, rules []Rule) bool {
	for _, rule := range rules {
		if rule.matches(dataPoint) {
			return true
		}
	}
	return false
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

func EvaluateWithConfidence(readings []ProviderReading, rules []Rule) []ConfidentRule {
	var successful []ProviderReading
	for _, pr := range readings {
		if pr.Err == nil {
			successful = append(successful, pr)
		}
	}
	if len(successful) == 0 {
		return nil
	}

	var result []ConfidentRule
	for _, rule := range rules {
		var matchedBy []string
		for _, pr := range successful {
			if len(EvaluateForecast(pr.Reading, []Rule{rule})) > 0 {
				matchedBy = append(matchedBy, pr.Name)
			}
		}
		if len(matchedBy) == 0 {
			continue
		}
		confidence := float64(len(matchedBy)) / float64(len(successful))
		if rule.MinConfidence > 0 && confidence < rule.MinConfidence {
			continue
		}
		result = append(result, ConfidentRule{
			Rule:           rule,
			Confidence:     confidence,
			MatchedBy:      matchedBy,
			TotalProviders: len(successful),
		})
	}
	return result
}

func EvaluateForecast(reading *WeatherReading, rules []Rule) []Rule {
	var triggered []Rule
	for _, rule := range rules {
		matched := false
		for _, dps := range reading.Readings {
			for i := range dps {
				if rule.matches(dps[i]) {
					dps[i].Matched = true
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
