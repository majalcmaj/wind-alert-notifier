package internal

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/majalcmaj/wind-alert/shared/model"
)

func TestPassingEmptyRule(t *testing.T) {
	reading := WindDataPoint{Time: time.Now(), WindSpeed: 10.0, WindAngle: 30.0}
	rules := []model.Rule{}

	if RunRuleEngine(reading, rules) {
		t.Errorf("Rule engine should return false when no rules are passed")
	}
}

func TestPassingNoMatchingRules(t *testing.T) {
	reading := WindDataPoint{Time: getTimeForHour(12.37), WindSpeed: 8.0, WindAngle: 20.1}
	rules := []model.Rule{
		{ // Not matching anything
			SpeedRange: model.Range{From: 0.0, To: 7.0},
			AngleRange: model.Range{From: 95.0, To: 130.37},
			HourRange:  model.Range{From: 20.0, To: 7.0},
		},
		{ // Not matching speed
			SpeedRange: model.Range{From: 20.1, To: 30.4},
			AngleRange: model.Range{From: 15.0, To: 21.37},
			HourRange:  model.Range{From: 10.0, To: 20.0},
		},
		{ // Not matching angle
			SpeedRange: model.Range{From: 7.0, To: 13.5},
			AngleRange: model.Range{From: 0.2, To: 2.137},
			HourRange:  model.Range{From: 10.0, To: 20.0},
		},
		{ // Not matching hour
			SpeedRange: model.Range{From: 7.0, To: 13.5},
			AngleRange: model.Range{From: 0.2, To: 21.37},
			HourRange:  model.Range{From: 20.0, To: 4.0},
		},
	}

	if RunRuleEngine(reading, rules) {
		t.Errorf("Rule engine should return false for no matching rules")
	}
}

func TestPassingSingleMatchingRule(t *testing.T) {
	reading := WindDataPoint{Time: getTimeForHour(1.0), WindSpeed: 8.0, WindAngle: 20.1}
	rules := []model.Rule{
		{ // Not matching rule
			SpeedRange: model.Range{From: 0.0, To: 7.0},
			AngleRange: model.Range{From: 95.0, To: 130.37},
			HourRange:  model.Range{From: 8.23, To: 21.37},
		},
		{ // Finally - a matching rule
			SpeedRange: model.Range{From: 0.0, To: 10.0},
			AngleRange: model.Range{From: 15.0, To: 21.37},
			HourRange:  model.Range{From: 21.37, To: 8.10},
		},
		{ // Another not matching angle
			SpeedRange: model.Range{From: 7.0, To: 13.5},
			AngleRange: model.Range{From: 0.2, To: 2.137},
			HourRange:  model.Range{From: 20.0, To: 7.0},
		},
	}

	if !RunRuleEngine(reading, rules) {
		t.Errorf("Rule engine should return true for a data matching the rule")
	}
}

func TestRuleWithAngleRangeFromHigherThanToAngleLessThanTo(t *testing.T) {
	reading := WindDataPoint{Time: getTimeForHour(10.0), WindSpeed: 5.0, WindAngle: 83}
	rules := []model.Rule{
		{
			SpeedRange: model.Range{From: 0.0, To: 7.0},
			AngleRange: model.Range{From: 270.0, To: 90.0},
			HourRange:  model.Range{From: 0.0, To: 23.99},
		},
	}

	if !RunRuleEngine(reading, rules) {
		t.Errorf("Rule engine should return true when rule's from is higher than to and angle is less than to")
	}
}

func TestRuleWithAngleRangeFromHigherThanToAngleBiggerThanFrom(t *testing.T) {
	reading := WindDataPoint{Time: getTimeForHour(12.01), WindSpeed: 5.0, WindAngle: 300}
	rules := []model.Rule{
		{
			SpeedRange: model.Range{From: 0.0, To: 7.0},
			AngleRange: model.Range{From: 270.0, To: 90.0},
			HourRange:  model.Range{From: 8.02, To: 13.12},
		},
	}

	if !RunRuleEngine(reading, rules) {
		t.Errorf("Rule engine should return true when rule's from is higher than to and angle is bigger than from")
	}
}

func TestRuleWithHourRangeFromHigherThanToAngleBiggerThanFrom(t *testing.T) {
	reading := WindDataPoint{Time: getTimeForHour(8.01), WindSpeed: 5.0, WindAngle: 300}
	rules := []model.Rule{
		{
			SpeedRange: model.Range{From: 0.0, To: 7.0},
			AngleRange: model.Range{From: 290.0, To: 315.0},
			HourRange:  model.Range{From: 21.37, To: 10.15},
		},
	}

	if !RunRuleEngine(reading, rules) {
		t.Errorf("Rule engine should return true when rule's from is higher than to and angle is bigger than from")
	}
}

func getTimeForHour(hour float64) time.Time {
	return time.Date(2025, time.November, 1, int(hour), int((hour-math.Floor(hour))*60.0), 0, 0, time.UTC)
}

func TestEvaluateForecastNoMatch(t *testing.T) {
	reading := &WeatherReading{
		Readings: map[string][]WindDataPoint{
			"hourly": {
				{Time: getTimeForHour(10), WindSpeed: 2, WindAngle: 90},
			},
		},
	}
	rules := []model.Rule{
		{SpeedRange: model.Range{From: 10, To: 20}, AngleRange: model.Range{From: 0, To: 360}, HourRange: model.Range{From: 0, To: 23.99}},
	}
	triggered := EvaluateForecast(reading, rules)
	if len(triggered) != 0 {
		t.Errorf("expected no triggered rules, got %d", len(triggered))
	}
	for _, dp := range reading.Readings["hourly"] {
		if dp.Matched {
			t.Error("expected no datapoint to be matched")
		}
	}
}

func TestEvaluateForecastOneMatch(t *testing.T) {
	dps := []WindDataPoint{
		{Time: getTimeForHour(10), WindSpeed: 8, WindAngle: 300},
		{Time: getTimeForHour(15), WindSpeed: 3, WindAngle: 90},
	}
	reading := &WeatherReading{
		Readings: map[string][]WindDataPoint{"hourly": dps},
	}
	rules := []model.Rule{
		{Name: "NW", SpeedRange: model.Range{From: 6, To: 20}, AngleRange: model.Range{From: 270, To: 360}, HourRange: model.Range{From: 8, To: 12}},
	}
	triggered := EvaluateForecast(reading, rules)
	if len(triggered) != 1 {
		t.Fatalf("expected 1 triggered rule, got %d", len(triggered))
	}
	if triggered[0].Name != "NW" {
		t.Errorf("wrong rule triggered: %s", triggered[0].Name)
	}
	if !dps[0].Matched {
		t.Error("expected first datapoint to be matched")
	}
	if dps[1].Matched {
		t.Error("expected second datapoint not to be matched")
	}
}

func TestEvaluateForecastDedupe(t *testing.T) {
	dps := []WindDataPoint{
		{Time: getTimeForHour(10), WindSpeed: 8, WindAngle: 300},
		{Time: getTimeForHour(11), WindSpeed: 9, WindAngle: 310},
	}
	reading := &WeatherReading{
		Readings: map[string][]WindDataPoint{"hourly": dps},
	}
	rules := []model.Rule{
		{Name: "NW", SpeedRange: model.Range{From: 6, To: 20}, AngleRange: model.Range{From: 270, To: 360}, HourRange: model.Range{From: 8, To: 12}},
	}
	triggered := EvaluateForecast(reading, rules)
	if len(triggered) != 1 {
		t.Errorf("expected deduplicated 1 rule, got %d", len(triggered))
	}
}

func TestEvaluateForecastHourlyAndDaily(t *testing.T) {
	hourly := []WindDataPoint{{Time: getTimeForHour(10), WindSpeed: 8, WindAngle: 300}}
	daily := []WindDataPoint{{Time: getTimeForHour(12), WindSpeed: 3, WindAngle: 90}}
	reading := &WeatherReading{
		Readings: map[string][]WindDataPoint{
			"hourly": hourly,
			"daily":  daily,
		},
	}
	rules := []model.Rule{
		{Name: "NW", SpeedRange: model.Range{From: 6, To: 20}, AngleRange: model.Range{From: 270, To: 360}, HourRange: model.Range{From: 8, To: 12}},
		{Name: "E", SpeedRange: model.Range{From: 0, To: 5}, AngleRange: model.Range{From: 60, To: 120}, HourRange: model.Range{From: 10, To: 14}},
	}
	triggered := EvaluateForecast(reading, rules)
	if len(triggered) != 2 {
		t.Errorf("expected 2 triggered rules (one from each bucket), got %d", len(triggered))
	}
}

func makeHourlyReading(speed, angle, hour float64) *WeatherReading {
	return &WeatherReading{
		Readings: map[string][]WindDataPoint{
			"hourly": {{Time: getTimeForHour(hour), WindSpeed: speed, WindAngle: angle}},
		},
	}
}

func TestEvaluateWithConfidenceAllAgree(t *testing.T) {
	rule := model.Rule{Name: "NW", SpeedRange: model.Range{From: 6, To: 20}, AngleRange: model.Range{From: 270, To: 360}, HourRange: model.Range{From: 8, To: 12}}
	readings := []ProviderReading{
		{Name: "p1", Reading: makeHourlyReading(8, 300, 10)},
		{Name: "p2", Reading: makeHourlyReading(9, 310, 11)},
		{Name: "p3", Reading: makeHourlyReading(7, 290, 10)},
	}

	result := EvaluateWithConfidence(readings, []model.Rule{rule})

	if len(result) != 1 {
		t.Fatalf("expected 1 confident rule, got %d", len(result))
	}
	if result[0].Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", result[0].Confidence)
	}
	if len(result[0].MatchedBy) != 3 {
		t.Errorf("expected 3 in MatchedBy, got %v", result[0].MatchedBy)
	}
	if result[0].TotalProviders != 3 {
		t.Errorf("expected TotalProviders=3, got %d", result[0].TotalProviders)
	}
}

func TestEvaluateWithConfidenceTwoOfThree(t *testing.T) {
	rule := model.Rule{Name: "NW", SpeedRange: model.Range{From: 6, To: 20}, AngleRange: model.Range{From: 270, To: 360}, HourRange: model.Range{From: 8, To: 12}}
	readings := []ProviderReading{
		{Name: "p1", Reading: makeHourlyReading(8, 300, 10)},
		{Name: "p2", Reading: makeHourlyReading(9, 310, 11)},
		{Name: "p3", Reading: makeHourlyReading(2, 90, 10)}, // no match
	}

	result := EvaluateWithConfidence(readings, []model.Rule{rule})

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Confidence != 2.0/3.0 {
		t.Errorf("expected confidence 2/3, got %f", result[0].Confidence)
	}
}

func TestEvaluateWithConfidenceNoneMatch(t *testing.T) {
	rule := model.Rule{Name: "NW", SpeedRange: model.Range{From: 6, To: 20}, AngleRange: model.Range{From: 270, To: 360}, HourRange: model.Range{From: 8, To: 12}}
	readings := []ProviderReading{
		{Name: "p1", Reading: makeHourlyReading(1, 90, 10)},
		{Name: "p2", Reading: makeHourlyReading(1, 90, 10)},
	}

	result := EvaluateWithConfidence(readings, []model.Rule{rule})

	if len(result) != 0 {
		t.Errorf("expected no results when nothing matches, got %d", len(result))
	}
}

func TestEvaluateWithConfidenceMinConfidenceFilters(t *testing.T) {
	rule := model.Rule{Name: "NW", SpeedRange: model.Range{From: 6, To: 20}, AngleRange: model.Range{From: 270, To: 360}, HourRange: model.Range{From: 8, To: 12}, MinConfidence: 0.9}
	readings := []ProviderReading{
		{Name: "p1", Reading: makeHourlyReading(8, 300, 10)},
		{Name: "p2", Reading: makeHourlyReading(9, 310, 11)},
		{Name: "p3", Reading: makeHourlyReading(2, 90, 10)}, // no match → confidence 0.66 < 0.9
	}

	result := EvaluateWithConfidence(readings, []model.Rule{rule})

	if len(result) != 0 {
		t.Errorf("expected rule filtered by MinConfidence, got %d results", len(result))
	}
}

func TestEvaluateWithConfidenceSkipsFailedProviders(t *testing.T) {
	rule := model.Rule{Name: "NW", SpeedRange: model.Range{From: 6, To: 20}, AngleRange: model.Range{From: 270, To: 360}, HourRange: model.Range{From: 8, To: 12}}
	readings := []ProviderReading{
		{Name: "p1", Reading: makeHourlyReading(8, 300, 10)},
		{Name: "p2", Err: errors.New("provider down")},
	}

	result := EvaluateWithConfidence(readings, []model.Rule{rule})

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Confidence != 1.0 {
		t.Errorf("expected confidence 1.0 (1/1 successful), got %f", result[0].Confidence)
	}
	if result[0].TotalProviders != 1 {
		t.Errorf("expected TotalProviders=1, got %d", result[0].TotalProviders)
	}
}
