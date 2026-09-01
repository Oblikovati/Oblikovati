// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// TestNopNutTrapCSG pins the vertical nut_trap construction shape: a tall screw
// clearance cylinder joined with a shorter hexagonal nut pocket.
func TestNopNutTrapCSG(t *testing.T) {
	t.Parallel()
	screw := prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.17, 32, 0), -1, 1, "screw-clearance")
	hex := hexPrismBody(0.32, -0.25, 0.25)
	body, err := ops.Boolean(ops.Join, screw, hex)
	if err != nil {
		t.Fatalf("Boolean(Join screw+hex): %v", err)
	}

	requireValidNopSolid(t, "nut_trap", body)
	screwArea := nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.17, 32, 0))
	hexArea := 3 * stdmath.Sqrt(3) * 0.32 * 0.32 / 2
	want := screwArea*2.0 + (hexArea-screwArea)*0.5
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("nut_trap volume = %.6f, want ~%.6f", got, want)
	}
}
