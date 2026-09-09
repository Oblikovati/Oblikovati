// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A plane that wedges a corner off a cylinder sections its side in an ELLIPSE that leaves through the
// top rim. The section is shared: the wall is trimmed by it and so is the tool's own face, and both
// must receive the SAME bounded arc or they meet the rim at two different corners and the stitch cannot
// weld them. The planar half of that co-refinement existed (clipSectionToFace); the wall half did not,
// and a straddling section was declined outright — which is what every notched cylinder in the corpus
// was built on (ADR-0062).
func TestClipSectionToWallBoundsAStraddlingEllipseAtTheRim(t *testing.T) {
	t.Parallel()
	body, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	wall := cylindricalFaceOfBody(t, body)
	// x + z = 9.5: the section ellipse is centred at z=9.5 and spans z ∈ [6.5, 12.5], so it leaves
	// through the rim at z=10.
	plane, err := geom.NewPlane(math.P3(1.5, 0, 8), math.V3(1, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	curves, handled := geom.IntersectSurfacesAnalytic(plane, wall.surface, geom.ResolutionForSize(10))
	if !handled || len(curves) != 1 {
		t.Fatalf("plane × cylinder: handled=%v n=%d, want one ellipse", handled, len(curves))
	}
	arcs, ok := clipSectionToWall(curves[0], wall)
	if !ok {
		t.Fatal("a section leaving through the rim was not bounded")
	}
	if len(arcs) != 1 {
		t.Fatalf("the clip gave %d arcs, want the one run inside the wall", len(arcs))
	}
	// Every point of the kept arc is on the wall, between its rims, and its ENDS sit on the top rim —
	// which is where the tool's own face will meet it too.
	arc := arcs[0]
	lo, hi := arc.Domain()
	for k := 0; k <= 32; k++ {
		z := float64(arc.PointAt(lo + (hi-lo)*float64(k)/32).Z)
		if z < -1e-9 || z > 10+1e-9 {
			t.Fatalf("the kept arc reaches z=%g, outside the wall's [0, 10]", z)
		}
	}
	for _, p := range []math.Point3{arc.PointAt(lo), arc.PointAt(hi)} {
		if d := stdmath.Abs(float64(p.Z) - 10); d > 1e-9 {
			t.Errorf("an arc end is %g off the top rim (z=%v), want it ON the rim", d, p.Z)
		}
	}
}

// A section that stays wholly inside the wall is not clipped: it has no crossing to be bounded at, so
// the clip declines and the caller keeps the whole island.
func TestClipSectionToWallDeclinesASectionInsideTheWall(t *testing.T) {
	t.Parallel()
	body, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	circle, err := geom.NewCircle(math.P3(0, 0, 5), math.V3(0, 0, 1), 3)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	if arcs, ok := clipSectionToWall(circle, cylindricalFaceOfBody(t, body)); ok || len(arcs) > 0 {
		t.Errorf("a section inside the wall gave %d arcs, ok=%v; want nothing to clip", len(arcs), ok)
	}
}

// cylinderSideFace returns the body's cylindrical face.
func cylindricalFaceOfBody(t *testing.T, body *topo.Body) curvedFace {
	t.Helper()
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return curvedFaceOf(f)
		}
	}
	t.Fatal("the body has no cylindrical face")
	return curvedFace{}
}
