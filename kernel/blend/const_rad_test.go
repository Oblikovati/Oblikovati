// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestSectionArcBetweenPlanes checks the section geometry on the simplest case: a radius-1 ball
// between the xy-plane and the yz-plane. The centre sits at (1,y,1); the contacts must land on each
// plane at distance 1, the sweep is 90°, and every section point is exactly radius 1 from the centre
// with PointAt(0)/PointAt(1) hitting the two contacts.
func TestSectionArcBetweenPlanes(t *testing.T) {
	xy, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1)) // z=0
	yz, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(1, 0, 0)) // x=0
	c := math.P3(1, 3, 1)                                      // equidistant 1 from both planes

	arc, ok := sectionAt(c, xy, yz, 1, 1e-9, func(math.Point3) bool { return false })
	if !ok {
		t.Fatal("section failed on a valid centre point")
	}
	if d := arc.FootA.DistanceTo(math.P3(1, 3, 0)); float64(d) > 1e-9 {
		t.Errorf("footA = %v, want (1,3,0)", arc.FootA)
	}
	if d := arc.FootB.DistanceTo(math.P3(0, 3, 1)); float64(d) > 1e-9 {
		t.Errorf("footB = %v, want (0,3,1)", arc.FootB)
	}
	if stdmath.Abs(arc.Sweep-stdmath.Pi/2) > 1e-9 {
		t.Errorf("sweep = %g, want π/2", arc.Sweep)
	}
	if d := arc.PointAt(0).DistanceTo(arc.FootA); float64(d) > 1e-9 {
		t.Errorf("PointAt(0) = %v, want footA %v", arc.PointAt(0), arc.FootA)
	}
	if d := arc.PointAt(1).DistanceTo(arc.FootB); float64(d) > 1e-9 {
		t.Errorf("PointAt(1) = %v, want footB %v", arc.PointAt(1), arc.FootB)
	}
	for _, v := range []float64{0, 0.25, 0.5, 0.75, 1} {
		if d := float64(arc.PointAt(v).DistanceTo(c)); stdmath.Abs(d-1) > 1e-9 {
			t.Errorf("section point at v=%g is %g from centre, want radius 1", v, d)
		}
	}
}

// TestSectionArcSelectsExposedSide checks that when the minor arc dips into the material, sectionAt
// flips to the major arc — the exposed blend surface always bulges out of the solid.
func TestSectionArcSelectsExposedSide(t *testing.T) {
	xy, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	yz, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(1, 0, 0))
	c := math.P3(1, 0, 1)

	// "material" = the quarter wedge x<1 ∧ z<1 near the centre: the minor-arc midpoint (0.29,0,0.29)
	// is inside it, so the section must flip to the major (>π/2) arc.
	inside := func(p math.Point3) bool { return float64(p.X) < 1 && float64(p.Z) < 1 }
	arc, ok := sectionAt(c, xy, yz, 1, 1e-9, inside)
	if !ok {
		t.Fatal("section failed")
	}
	if arc.Sweep <= stdmath.Pi/2 {
		t.Errorf("sweep = %g, want the major arc (>π/2) when the minor dips inside", arc.Sweep)
	}
	if !insidePointOutsideMaterial(arc, inside) {
		t.Error("major-arc midpoint should now be outside the material")
	}
}

func insidePointOutsideMaterial(arc SectionArc, inside func(math.Point3) bool) bool {
	return !inside(arc.PointAt(0.5))
}
