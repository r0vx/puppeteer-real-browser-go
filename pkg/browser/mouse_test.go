package browser

import (
	"math"
	"testing"
)

// TestWheelSteps verifies the scroll delta is split into multiple ticks whose
// sum equals the original delta exactly (drift folded into the last tick).
func TestWheelSteps(t *testing.T) {
	cases := [][2]float64{{0, 800}, {0, 50}, {-300, 0}, {0, 0}, {120, 640}}
	for _, c := range cases {
		steps := wheelSteps(c[0], c[1])
		if len(steps) < 1 {
			t.Fatalf("wheelSteps(%v) returned no steps", c)
		}
		// large scrolls must be broken into multiple ticks (not one atomic wheel)
		if math.Max(math.Abs(c[0]), math.Abs(c[1])) > 150 && len(steps) < 2 {
			t.Errorf("wheelSteps(%v): expected multiple ticks, got %d", c, len(steps))
		}
		var sumX, sumY float64
		for _, s := range steps {
			sumX += s[0]
			sumY += s[1]
		}
		if math.Abs(sumX-c[0]) > 1e-6 || math.Abs(sumY-c[1]) > 1e-6 {
			t.Errorf("wheelSteps(%v): sum (%v,%v) != delta", c, sumX, sumY)
		}
	}
}
