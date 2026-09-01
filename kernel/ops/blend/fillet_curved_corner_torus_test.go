// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// TestSolveCurvedMixedCornerM8 pins the M8 mixed-sense curved-host corner-patch geometry against the
// OCCT DRAWEXE oracle (blend/simple/M8, tscale ×10): the 2r-torus corner centre (55,30.635,105), major
// 10, minor 5, and the two derived spine feet F_A=(52.5,20.955,105) / F_B=(65,30.635,105). Inputs are
// the exact DRAWEXE arm surfaces (fillet_curved_corner_torus.go's derivation, slice1a-report.md).
func TestSolveCurvedMixedCornerM8(t *testing.T) {
	t.Parallel()
	const r = 5
	arms := curvedMixedArms{
		convex: mustCylinder(t, math.P3(55, 30.6350832689629, 100), math.V3(0, 0, 1), r),
		cove:   mustTorus(t, math.P3(60, 50, 105), math.V3(0, 0, 1), 30, r),
		boss:   mustCylinder(t, math.P3(60, 50, 100), math.V3(0, 0, 1), 25),
		planar: mustCylinder(t, math.P3(65, 50, 105), math.V3(0, -1, 0), r),
		top:    mustPlane(t, math.P3(0, 0, 100), math.V3(0, 0, 1)),
		topOut: math.V3(0, 0, 1),
	}
	c, ok := solveCurvedMixedCorner(arms, r, tol.ForPoints([]math.Point3{math.P3(60, 50, 100), math.P3(0, 0, 0)}))
	if !ok {
		t.Fatal("solveCurvedMixedCorner declined the DRAWEXE-verified M8 corner")
	}
	assertPoint(t, "center", c.center, math.P3(55, 30.6350832689629, 105))
	if stdmath.Abs(c.major-10) > 1e-6 {
		t.Errorf("major = %.9f, want 10 (=2r)", c.major)
	}
	// F_A is the shared point with the cove torus arm — arcCove's inner-equator endpoint sits on the boss
	// wall (radius 25 from the boss axis) and its bottom endpoint sits on the box top (z=100).
	assertPoint(t, "arcCove.start (Q_inner_A, boss wall)", c.arcCove.PointAt(0), math.P3(53.75, 25.79385408620363, 105))
	assertPoint(t, "arcCove.end   (Q_bot_A, box top)", c.arcCove.PointAt(1), math.P3(52.5, 20.95262490344436, 100))
	// arcInner (b) is shared with the convex cylinder arm: both endpoints at radius r from the convex axis.
	assertPoint(t, "arcInner.end (Q_inner_B)", c.arcInner.PointAt(1), math.P3(60, 30.6350832689629, 105))
	// arcTop (d) lies on the box top plane z=100 at radius 2r=10 from the axis.
	if z := c.arcTop.PointAt(0).Z; stdmath.Abs(z-100) > 1e-6 {
		t.Errorf("arcTop.start.z = %.9f, want 100 (on the box top plane)", z)
	}
	assertPoint(t, "arcTop.end (Q_bot_B)", c.arcTop.PointAt(1), math.P3(65, 30.6350832689629, 100))
}

// TestSolveCurvedMixedCornerDeclinesNon2r checks the R=2r gate: a cove circle whose foot is NOT at 2r
// from C (a BSpline mixed corner, N4/O1/H7 class) is declined, never mis-modelled as an analytic torus.
func TestSolveCurvedMixedCornerDeclinesNon2r(t *testing.T) {
	t.Parallel()
	const r = 5
	arms := curvedMixedArms{
		convex: mustCylinder(t, math.P3(55, 30.6350832689629, 100), math.V3(0, 0, 1), r),
		cove:   mustTorus(t, math.P3(60, 50, 105), math.V3(0, 0, 1), 15, r), // R−r cove (major 15) — not R+r
		boss:   mustCylinder(t, math.P3(60, 50, 100), math.V3(0, 0, 1), 25),
		planar: mustCylinder(t, math.P3(65, 50, 105), math.V3(0, -1, 0), r),
		top:    mustPlane(t, math.P3(0, 0, 100), math.V3(0, 0, 1)),
		topOut: math.V3(0, 0, 1),
	}
	if _, ok := solveCurvedMixedCorner(arms, r, tol.ForPoints([]math.Point3{math.P3(60, 50, 100), math.P3(0, 0, 0)})); !ok {
		return // declined as expected (F_A off the major-15 circle)
	}
	t.Error("solveCurvedMixedCorner accepted a corner whose cove foot is not at the 2r pivot")
}

func mustTorus(t *testing.T, c math.Point3, axis math.Vector3, major, minor float64) geom.Torus {
	t.Helper()
	tor, err := geom.NewTorus(c, axis, major, minor)
	if err != nil {
		t.Fatalf("NewTorus: %v", err)
	}
	return tor
}
