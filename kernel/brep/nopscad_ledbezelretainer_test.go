// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

func TestNopLedBezelRetainerCSG(t *testing.T) {
	t.Parallel()
	body := annularPrism(t, 0.45, 0.32, 0.4, "led-bezel-retainer")
	requireValidNopSolid(t, "led_bezel_retainer", body)
	want := (nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.45, 64, 0)) - nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.32, 12, stdmath.Pi/12))) * 0.4
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("led_bezel_retainer volume = %.6f, want %.6f", got, want)
	}
}
