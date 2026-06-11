package model

import (
	"strings"
	"testing"
)

func TestDescribeNamedRule(t *testing.T) {
	rule := Rule{Name: "Strong NW wind", SpeedRange: Range{5, 15}, AngleRange: Range{270, 360}, HourRange: Range{6, 20}}
	if rule.Describe() != "Strong NW wind" {
		t.Errorf("Describe() = %q, want %q", rule.Describe(), "Strong NW wind")
	}
}

func TestDescribeUnnamedRule(t *testing.T) {
	rule := Rule{SpeedRange: Range{5, 15}, AngleRange: Range{45, 135}, HourRange: Range{6, 20}}
	desc := rule.Describe()
	if desc == "" {
		t.Error("Describe() returned empty string for unnamed rule")
	}
	for _, want := range []string{"5.0", "15.0", "06:00", "20:00"} {
		if !strings.Contains(desc, want) {
			t.Errorf("Describe() = %q, missing %q", desc, want)
		}
	}
}
