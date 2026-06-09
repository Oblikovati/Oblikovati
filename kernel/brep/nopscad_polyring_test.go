// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

func TestNopPolyRingCSG(t *testing.T) {
	body := annularPrism(t, 0.7, 0.35, 0.12, "poly-ring")
	requireValidNopSolid(t, "poly_ring", body)
	want := (nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.7, 64, 0)) - nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.35, 12, stdmath.Pi/12))) * 0.12
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("poly_ring volume = %.6f, want %.6f", got, want)
	}
}
