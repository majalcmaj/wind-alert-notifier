package internal

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

func makeModel(reading *WeatherReading) MailModel {
	return MailModel{Results: []LocationResult{{Location: reading.Location, Reading: reading}}}
}

func TestRenderingMailDisplaysTitle(t *testing.T) {
	result, err := RenderMail(makeModel(&WeatherReading{
		Location: Location{Name: "Test"},
		Readings: map[string][]WindDataPoint{},
	}))

	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	if !strings.Contains(result, "Wind Alert!") {
		t.Errorf("Expected 'Wind alert!' title to be present but got '%s'", result)
	}
}

func TestRenderingLocationName(t *testing.T) {
	result, err := RenderMail(makeModel(&WeatherReading{
		Location: Location{Name: "Sopot"},
		Readings: map[string][]WindDataPoint{},
	}))

	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	if !strings.Contains(result, "Sopot") {
		t.Errorf("Expected location name to be present but got '%s'", result)
	}
}

func TestRenderingDailyAndHourlyTables(t *testing.T) {
	reading := WeatherReading{
		Location: Location{Name: "Test"},
		Readings: map[string][]WindDataPoint{
			"daily": {
				{Time: parseTime("2025-01-01T10:00"), WindSpeed: 10, WindAngle: 180},
				{Time: parseTime("2025-01-02T10:00"), WindSpeed: 20, WindAngle: 80},
			},
			"hourly": {
				{Time: parseTime("2025-01-01T10:00"), WindSpeed: 10, WindAngle: 180},
				{Time: parseTime("2025-01-01T11:00"), WindSpeed: 12, WindAngle: 190},
			},
		},
	}

	renderedMail, err := RenderMail(makeModel(&reading))

	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	for _, row := range reading.Readings["daily"] {
		matchRow(t, renderedMail, row)
	}

	for _, row := range reading.Readings["hourly"] {
		matchRow(t, renderedMail, row)
	}
}

func TestRenderingTriggeredRules(t *testing.T) {
	model := MailModel{
		Results: []LocationResult{
			{
				Reading:        &WeatherReading{Readings: map[string][]WindDataPoint{}},
				TriggeredRules: []string{"Strong NW afternoon wind", "Any strong wind"},
			},
		},
	}

	result, err := RenderMail(model)
	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	for _, rule := range model.Results[0].TriggeredRules {
		if !strings.Contains(result, rule) {
			t.Errorf("Expected triggered rule %q in output", rule)
		}
	}
}

func TestRenderingMatchedRowBolded(t *testing.T) {
	reading := WeatherReading{
		Readings: map[string][]WindDataPoint{
			"hourly": {
				{Time: parseTime("2025-01-01T10:00"), WindSpeed: 10, WindAngle: 180, Matched: true},
				{Time: parseTime("2025-01-01T11:00"), WindSpeed: 5, WindAngle: 90, Matched: false},
			},
			"daily": {},
		},
	}

	result, err := RenderMail(makeModel(&reading))
	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	boldPattern := regexp.MustCompile(`<strong>.*10\.0.*</strong>`)
	if !boldPattern.MatchString(result) {
		t.Errorf("Expected matched row (10.0) to be bolded in:\n%s", result)
	}

	nonBoldTime := "2025-01-01 11:00"
	boldPatternUnmatched := regexp.MustCompile(`<strong>.*` + nonBoldTime + `.*</strong>`)
	if boldPatternUnmatched.MatchString(result) {
		t.Errorf("Expected unmatched row (%s) NOT to be bolded", nonBoldTime)
	}
}

func parseTime(tStr string) time.Time {
	tm, err := time.Parse(time.RFC3339, tStr+":00Z")
	if err != nil {
		panic(err)
	}
	return tm
}

func matchRow(t *testing.T, mailHtml string, row WindDataPoint) {
	tableRegEx := regexp.MustCompile(
		fmt.Sprintf(`(?s)<tr.*>.*<td.*>.*%s.*</td>.*<td.*>.*%.1f.*</td>.*<td.*>.*%s.*</td>.*</tr>`,
			row.Time.Format("2006-01-02 15:04"), row.WindSpeed, renderWindArrow(row.WindAngle)))

	match := tableRegEx.MatchString(mailHtml)
	if !match {
		t.Errorf("Expected WindDataPoint %+v not found in:\n%s", row, mailHtml)
	}
}
