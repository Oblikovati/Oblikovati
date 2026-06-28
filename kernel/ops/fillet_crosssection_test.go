// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
)

// filletCross fillets one vertical edge of a 2×2×2 box at r=0.5 with the given cross-section, and
// returns the validated result body's volume (fine tessellation).
func filletCross(t *testing.T, cross ops.FilletCrossSection, rho float64) float64 {
	t.Helper()
	box := shellBox(2, 2, 2)
	res, err := ops.FilletEdgesVarying(box, []ops.EdgeFilletRadii{
		{Key: verticalEdgeKey(t, box), R0: 0.5, R1: 0.5, Cross: cross, Rho: rho},
	})
	if err != nil {
		t.Fatalf("fillet (cross=%s rho=%g): %v", cross, rho, err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("cross=%s fillet not a valid solid: %+v", cross, r)
	}
	return ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-3}).Volume
}

// TestFilletG2EdgeIsValidSolid: a G2 cross-section fillet of a box edge builds a watertight solid
// (the swept G2 band retrims the walls to the same tangent lines as the arc).
func TestFilletG2EdgeIsValidSolid(t *testing.T) {
	v := filletCross(t, ops.FilletG2, 0)
	if v <= 7 || v >= 8 {
		t.Errorf("G2 fillet volume = %g, want a rounded box (7..8)", v)
	}
}

// TestFilletConicEdgeIsValidSolid: a conic (parabola) cross-section fillet builds a valid solid.
func TestFilletConicEdgeIsValidSolid(t *testing.T) {
	v := filletCross(t, ops.FilletConic, 0.5)
	if v <= 7 || v >= 8 {
		t.Errorf("conic fillet volume = %g, want a rounded box (7..8)", v)
	}
}

// TestFilletConicRhoSweepsFullness: a fuller conic (higher rho) leaves MORE material in the corner
// (the profile bulges toward the sharp corner), so the body volume increases monotonically with rho.
func TestFilletConicRhoSweepsFullness(t *testing.T) {
	flat := filletCross(t, ops.FilletConic, 0.25)
	mid := filletCross(t, ops.FilletConic, 0.5)
	full := filletCross(t, ops.FilletConic, 0.75)
	if !(flat < mid && mid < full) {
		t.Errorf("conic volume not monotonic in rho: 0.25→%g, 0.5→%g, 0.75→%g (want increasing)", flat, mid, full)
	}
}

// TestFilletG2DiffersFromArc: the G2 cross-section removes more material near the tangency lines than
// the circular arc (it flattens toward the walls), so its volume differs measurably from the arc.
func TestFilletG2DiffersFromArc(t *testing.T) {
	arc := filletCross(t, ops.FilletArc, 0)
	g2 := filletCross(t, ops.FilletG2, 0)
	if stdmath.Abs(arc-g2) < 1e-3 {
		t.Errorf("G2 (%g) should differ from arc (%g) — it is a genuinely different cross-section", g2, arc)
	}
}
