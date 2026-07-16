package internal

import (
	"wind-alert/internal/model"
)

type ConfidentRule struct {
	model.Rule
	Confidence     float64
	MatchedBy      []string
	TotalProviders int
}

func matches(r model.Rule, dp WindDataPoint) bool {
	return r.AngleRange.WithinCyclicRange(dp.WindAngle) &&
		r.SpeedRange.WithinRange(dp.WindSpeed) &&
		r.HourRange.WithinCyclicRange(float64(dp.Time.Hour())+float64(dp.Time.Minute())/60.0)
}

func RunRuleEngine(dataPoint WindDataPoint, rules []model.Rule) bool {
	for _, rule := range rules {
		if matches(rule, dataPoint) {
			return true
		}
	}
	return false
}

func EvaluateWithConfidence(readings []ProviderReading, rules []model.Rule) []ConfidentRule {
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
			if len(EvaluateForecast(pr.Reading, []model.Rule{rule})) > 0 {
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

func EvaluateForecast(reading *WeatherReading, rules []model.Rule) []model.Rule {
	var triggered []model.Rule
	for _, rule := range rules {
		matched := false
		for _, dps := range reading.Readings {
			for i := range dps {
				if matches(rule, dps[i]) {
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
