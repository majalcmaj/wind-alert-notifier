package internal

import (
	"testing"
)

func TestRendersWindArrowsCorrectly(t *testing.T) {
	input := map[float64]string{
		338: "↑\uFE0E",
		372: "↑\uFE0E",
		0:   "↑\uFE0E",
		21:  "↑\uFE0E",
		23:  "↗\uFE0E",
		45:  "↗\uFE0E",
		67:  "↗\uFE0E",
		68:  "→\uFE0E",
		90:  "→\uFE0E",
		112: "→\uFE0E",
		113: "↘\uFE0E",
		135: "↘\uFE0E",
		157: "↘\uFE0E",
		180: "↓\uFE0E",
		202: "↓\uFE0E",
		203: "↙\uFE0E",
		225: "↙\uFE0E",
		247: "↙\uFE0E",
		270: "←\uFE0E",
		292: "←\uFE0E",
		293: "↖\uFE0E",
		315: "↖\uFE0E",
		337: "↖\uFE0E",
	}

	for angle, expectedRune := range input {
		result := renderWindArrow(angle)
		if result != expectedRune {
			t.Errorf("For angle %f expected %s but got %s", angle, expectedRune, result)
		}
	}
}
