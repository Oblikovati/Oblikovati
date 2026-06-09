// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/brep"
	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// revVolume validates the body as a watertight solid and returns its tessellated volume.
func revVolume(t *testing.T, b *topo.Body) float64 {
	t.Helper()
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("revolution not a valid solid: %+v", r.Issues)
	}
	if open := ops.BoundaryEdges(b); len(open) != 0 {
		t.Fatalf("revolution has %d boundary edges, want 0 (watertight)", len(open))
	}
	return ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
}

// TestRevolutionTubeIsAnalyticAnnulus revolves a rectangular annular meridian into a tube and
// asserts it is a valid watertight solid with the analytic annulus volume AND two true cylindrical
// walls (the bore + outer) — the surfaces thread/chamfer/fillet attach to (#129).
func TestRevolutionTubeIsAnalyticAnnulus(t *testing.T) {
	const rIn, rOut, h = 2.5, 6.0, 20.0
	mer := []math.Point2{math.P2(rIn, 0), math.P2(rOut, 0), math.P2(rOut, h), math.P2(rIn, h)}

	body, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), mer, "tube")
	if err != nil || body == nil {
		t.Fatalf("SolidOfRevolution(tube) = %v, %v; want a body", body, err)
	}
	got := revVolume(t, body)
	want := stdmath.Pi * (rOut*rOut - rIn*rIn) * h
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("tube volume = %.4f, want analytic %.4f (rel %.4f > 3%% faceting band)", got, want, rel)
	}

	cyls := 0
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			cyls++
		}
	}
	if cyls != 2 {
		t.Errorf("tube has %d analytic cylinder faces, want 2 (bore + outer wall)", cyls)
	}
}

// TestRevolutionDiscIsSolidCylinder revolves a rectangle touching the axis into a SOLID cylinder
// (the inner edge is on the axis ⇒ disk caps, one cylindrical wall).
func TestRevolutionDiscIsSolidCylinder(t *testing.T) {
	const r, h = 5.0, 8.0
	mer := []math.Point2{math.P2(0, 0), math.P2(r, 0), math.P2(r, h), math.P2(0, h)}

	body, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), mer, "disc")
	if err != nil || body == nil {
		t.Fatalf("SolidOfRevolution(disc) = %v, %v; want a body", body, err)
	}
	got := revVolume(t, body)
	want := stdmath.Pi * r * r * h
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("disc volume = %.4f, want analytic %.4f (rel %.4f > 3%% band)", got, want, rel)
	}
	if n := len(body.Faces()); n != 3 {
		t.Errorf("solid cylinder has %d faces, want 3 (wall + 2 disc caps)", n)
	}
}

// TestRevolutionObliqueEdgeFallsBack pins the documented limit: an oblique (cone) meridian edge
// returns (nil,nil) so the caller uses the faceted path until analytic cones land.
func TestRevolutionObliqueEdgeFallsBack(t *testing.T) {
	mer := []math.Point2{math.P2(2, 0), math.P2(6, 0), math.P2(2, 10)} // a cone: oblique hypotenuse
	body, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), mer, "cone")
	if err != nil || body != nil {
		t.Fatalf("oblique meridian: got (%v,%v), want (nil,nil) fallback", body, err)
	}
}
