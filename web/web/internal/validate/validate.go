package validate

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/majalcmaj/wind-alert/shared/model"
)

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func ValidateLocation(l model.Location) error {
	var errs []error

	if l.ID == "" || !slugRe.MatchString(l.ID) {
		errs = append(errs, fmt.Errorf("id: must be a non-empty slug (lowercase letters, digits, hyphens)"))
	}
	if l.Name == "" {
		errs = append(errs, fmt.Errorf("name: must not be empty"))
	}
	if l.Lat < -90 || l.Lat > 90 {
		errs = append(errs, fmt.Errorf("lat: must be in [-90, 90]"))
	}
	if l.Lon < -180 || l.Lon > 180 {
		errs = append(errs, fmt.Errorf("lon: must be in [-180, 180]"))
	}

	return errors.Join(errs...)
}

func ValidateRule(r model.Rule) error {
	var errs []error

	if r.LocationID == "" {
		errs = append(errs, fmt.Errorf("location_id: must not be empty"))
	}
	if r.Name == "" {
		errs = append(errs, fmt.Errorf("name: must not be empty"))
	}
	if r.SpeedRange.From < 0 {
		errs = append(errs, fmt.Errorf("speed.from: must be >= 0"))
	}
	if r.SpeedRange.To < 0 {
		errs = append(errs, fmt.Errorf("speed.to: must be >= 0"))
	}
	if r.SpeedRange.From > r.SpeedRange.To {
		errs = append(errs, fmt.Errorf("speed: from must be <= to"))
	}
	if r.AngleRange.From < 0 || r.AngleRange.From > 360 {
		errs = append(errs, fmt.Errorf("angle.from: must be in [0, 360]"))
	}
	if r.AngleRange.To < 0 || r.AngleRange.To > 360 {
		errs = append(errs, fmt.Errorf("angle.to: must be in [0, 360]"))
	}
	if r.HourRange.From < 0 || r.HourRange.From > 24 {
		errs = append(errs, fmt.Errorf("hour.from: must be in [0, 24]"))
	}
	if r.HourRange.To < 0 || r.HourRange.To > 24 {
		errs = append(errs, fmt.Errorf("hour.to: must be in [0, 24]"))
	}
	if r.MinConfidence < 0 || r.MinConfidence > 1 {
		errs = append(errs, fmt.Errorf("min_confidence: must be in [0, 1]"))
	}

	return errors.Join(errs...)
}
