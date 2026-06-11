package validate

import (
	"testing"

	"github.com/majalcmaj/wind-alert/shared/model"
)

func TestValidateLocation(t *testing.T) {
	tests := []struct {
		name    string
		loc     model.Location
		wantErr bool
	}{
		{"valid", model.Location{ID: "lake-balaton", Name: "Lake Balaton", Lat: 46.8, Lon: 17.7}, false},
		{"empty id", model.Location{ID: "", Name: "X", Lat: 0, Lon: 0}, true},
		{"non-slug id", model.Location{ID: "Lake_Balaton", Name: "X", Lat: 0, Lon: 0}, true},
		{"empty name", model.Location{ID: "x", Name: "", Lat: 0, Lon: 0}, true},
		{"lat too high", model.Location{ID: "x", Name: "X", Lat: 91, Lon: 0}, true},
		{"lat too low", model.Location{ID: "x", Name: "X", Lat: -91, Lon: 0}, true},
		{"lon too high", model.Location{ID: "x", Name: "X", Lat: 0, Lon: 181}, true},
		{"lon too low", model.Location{ID: "x", Name: "X", Lat: 0, Lon: -181}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLocation(tt.loc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLocation(%+v) error = %v, wantErr %v", tt.loc, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRule(t *testing.T) {
	valid := model.Rule{
		Name:          "strong-wind",
		LocationID:    "lake-balaton",
		AngleRange:    model.Range{From: 90, To: 180},
		SpeedRange:    model.Range{From: 5, To: 15},
		HourRange:     model.Range{From: 8, To: 18},
		MinConfidence: 0.5,
	}

	tests := []struct {
		name    string
		mutate  func(r model.Rule) model.Rule
		wantErr bool
	}{
		{"valid", func(r model.Rule) model.Rule { return r }, false},
		{"cyclic angle wrap", func(r model.Rule) model.Rule { r.AngleRange = model.Range{From: 350, To: 10}; return r }, false},
		{"cyclic hour overnight", func(r model.Rule) model.Rule { r.HourRange = model.Range{From: 22, To: 6}; return r }, false},
		{"empty location id", func(r model.Rule) model.Rule { r.LocationID = ""; return r }, true},
		{"empty name", func(r model.Rule) model.Rule { r.Name = ""; return r }, true},
		{"speed from negative", func(r model.Rule) model.Rule { r.SpeedRange = model.Range{From: -1, To: 10}; return r }, true},
		{"speed to negative", func(r model.Rule) model.Rule { r.SpeedRange = model.Range{From: 0, To: -1}; return r }, true},
		{"speed from > to", func(r model.Rule) model.Rule { r.SpeedRange = model.Range{From: 20, To: 10}; return r }, true},
		{"angle from out of range", func(r model.Rule) model.Rule { r.AngleRange = model.Range{From: -1, To: 10}; return r }, true},
		{"angle to out of range", func(r model.Rule) model.Rule { r.AngleRange = model.Range{From: 0, To: 361}; return r }, true},
		{"hour from out of range", func(r model.Rule) model.Rule { r.HourRange = model.Range{From: -1, To: 6}; return r }, true},
		{"hour to out of range", func(r model.Rule) model.Rule { r.HourRange = model.Range{From: 0, To: 25}; return r }, true},
		{"min confidence too high", func(r model.Rule) model.Rule { r.MinConfidence = 1.5; return r }, true},
		{"min confidence negative", func(r model.Rule) model.Rule { r.MinConfidence = -0.1; return r }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.mutate(valid)
			err := ValidateRule(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRule(%+v) error = %v, wantErr %v", r, err, tt.wantErr)
			}
		})
	}
}
