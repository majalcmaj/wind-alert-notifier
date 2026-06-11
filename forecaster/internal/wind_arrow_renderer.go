package internal

import "math"

func renderWindArrow(angle float64) string {
	angle = math.Mod(angle, 360)
	switch {
	case angle >= 337.5 || angle < 22.5:
		return "↑\uFE0E" // North
	case angle < 67.5:
		return "↗\uFE0E" // Northeast
	case angle < 112.5:
		return "→\uFE0E" // East
	case angle < 157.5:
		return "↘\uFE0E" // Southeast
	case angle < 202.5:
		return "↓\uFE0E" // South
	case angle < 247.5:
		return "↙\uFE0E" // Southwest
	case angle < 292.5:
		return "←\uFE0E" // West
	case angle >= 292.5:
		return "↖\uFE0E" // Northwest
	default:
		return "?" // Should never happen
	}
}
