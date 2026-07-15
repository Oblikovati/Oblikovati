// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// nearly reports whether two scalars agree to a generous absolute tolerance (the fixtures are exact;
// the slack only absorbs the OCCT oracle's 2-decimal BREP rounding and floating-point round-off).
func nearly(got, want float64) bool { return stdmath.Abs(got-want) < 1e-2 }

// nearlyPt reports whether two points agree component-wise under nearly.
func nearlyPt(got, want math.Point3) bool {
	return nearly(float64(got.X), float64(want.X)) &&
		nearly(float64(got.Y), float64(want.Y)) &&
		nearly(float64(got.Z), float64(want.Z))
}

// TestSolveBlend_B3CurvedCorner drives the curved-host corner solve on the REAL corpus fixture: B3's
// three picked edges (top-rim arc, vertical wall line, top radial segment, all r=10) meet at the
// [Cylinder,Plane,Plane] vertex (0,−50,100). computeCorners must now return ONE cornerBlend carrying an
// analytic geom.Sphere of radius 10 tangent to the R=50 wall (centre 40 = R−r from the z-axis) and 10
// inside both planes — centre (10, −38.73, 90) — instead of the old "corner face must be planar" reject.
//
// The asserted centre is the committed fixture's frame: the derivation's DRAWEXE run reported
// (38.73, 10, 90), which is THIS point rotated −90° about z (a trotate-frame difference). Both are the
// same OCCT corner KPart (BREP surface code 4) — radius r, centre 40 from the axis, z=90.
func TestSolveBlend_B3CurvedCorner(t *testing.T) {
	body := importCorpusSolid(t, "simple/B3")
	picks := cornerEdgePicks(t, body, math.P3(0, -50, 100), 10)
	blends, _, err := computeCorners(picks)
	if err != nil {
		t.Fatalf("computeCorners on B3 curved corner errored (still rejecting curved host): %v", err)
	}
	cb := singleBlend(t, blends)
	if !nearly(cb.sphere.Radius, 10) || !nearlyPt(cb.center, math.P3(10, -stdmath.Sqrt(1500), 90)) {
		t.Fatalf("B3 corner sphere = {center %v r %.6f}, want center (10,-38.7298,90) r10 (OCCT KPart code 4)",
			cb.center, cb.sphere.Radius)
	}
	assertOnCylinderWall(t, cb.center, body)
}

// assertOnCylinderWall confirms the corner centre sits at distance R−r (=40) from B3's boss axis — the
// convex/concave discrimination: an R+r=60 (concave) or mis-signed centre fails here, not just the
// nominal coordinate check.
func assertOnCylinderWall(t *testing.T, c math.Point3, body *topo.Body) {
	t.Helper()
	cyl := b3BossWall(t, body)
	w := cyl.Origin.VectorTo(c)
	perp := w.Sub(cyl.AxisDir.AsVector().Scale(w.Dot(cyl.AxisDir.AsVector())))
	if d := float64(perp.Length()); !nearly(d, 40) {
		t.Fatalf("corner centre is %.4f from the boss axis, want R−r = 40 (convex)", d)
	}
}

// b3BossWall returns B3's R=50 boss-wall cylinder geometry.
func b3BossWall(t *testing.T, body *topo.Body) geom.Cylinder {
	t.Helper()
	for _, f := range body.Faces() {
		if c, ok := f.Geometry().(geom.Cylinder); ok {
			return c
		}
	}
	t.Fatalf("B3 boss-wall cylinder face not found")
	return geom.Cylinder{}
}
