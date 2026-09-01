// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// TestEntityDistancePointCases locks the point→curve and point→face recursion on a 4×3×5 block: a
// point off a face projects to its foot when the foot is in the trim, else to the nearest boundary
// edge; a point on the solid touches (0).
func TestEntityDistancePointCases(t *testing.T) {
	t.Parallel()
	block, err := SolidBlock(gmath.P3(0, 0, 0), gmath.P3(4, 3, 5), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	top := faceWithNormal(t, block, gmath.V3(0, 0, 1)) // z=5 face

	// A point 2 above the centre of the top face projects straight down onto it → gap 2.
	if got := EntityDistance(PointSupport(gmath.P3(2, 1.5, 7)), FaceSupport(top)); math.Abs(got-2) > 1e-9 {
		t.Errorf("point over face = %.9f, want 2 (perpendicular foot in trim)", got)
	}
	// A point beyond the +x,+y corner above the face: foot leaves the trim, so the nearest boundary
	// edge/corner governs — the corner (4,3,5) is √(1+1+2²)=√6 away from (5,4,7).
	if got := EntityDistance(PointSupport(gmath.P3(5, 4, 7)), FaceSupport(top)); math.Abs(got-math.Sqrt(6)) > 1e-9 {
		t.Errorf("point beyond corner = %.9f, want √6 (boundary governs)", got)
	}
	// A point lying on the face itself touches it.
	if got := EntityDistance(PointSupport(gmath.P3(2, 1.5, 5)), FaceSupport(top)); got > 1e-9 {
		t.Errorf("point on face = %.9f, want 0", got)
	}
}

// TestEntityDistanceCurveFace locks the curve→face path: a segment held parallel above a face clears
// it by the gap, and a segment piercing the face touches.
func TestEntityDistanceCurveFace(t *testing.T) {
	t.Parallel()
	block, err := SolidBlock(gmath.P3(0, 0, 0), gmath.P3(4, 3, 5), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	top := faceWithNormal(t, block, gmath.V3(0, 0, 1))

	above := geom.NewLineSegment(gmath.P3(1, 1, 8), gmath.P3(3, 2, 8)) // 3 above z=5
	if got := EntityDistance(CurveSupport(above), FaceSupport(top)); math.Abs(got-3) > 1e-9 {
		t.Errorf("segment over face = %.9f, want 3", got)
	}
	pierce := geom.NewLineSegment(gmath.P3(2, 1.5, 4), gmath.P3(2, 1.5, 6))
	if got := EntityDistance(CurveSupport(pierce), FaceSupport(top)); got > 1e-9 {
		t.Errorf("piercing segment = %.9f, want 0", got)
	}
}

// TestEntityDistanceInteriorSpherePlane locks the interior surface–surface solver against an exact
// convex-curved clearance: a sphere below a block's underside face. The nearest sphere point is
// off every tessellation vertex, so only the analytic reading hits the exact gap.
func TestEntityDistanceInteriorSpherePlane(t *testing.T) {
	t.Parallel()
	block, err := SolidBlock(gmath.P3(-5, -5, 3), gmath.P3(5, 5, 8), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	under := faceWithNormal(t, block, gmath.V3(0, 0, -1)) // z=3 face, normal −z
	sphere, err := SolidSphere(gmath.P3(0, 0, 0), 1, "s") // top of sphere at z=1
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	if got := EntityDistance(FaceSupport(sphere.Faces()[0]), FaceSupport(under)); math.Abs(got-2) > 1e-6 {
		t.Errorf("sphere–plane clearance = %.9f, want 2 (exact)", got)
	}
}

// faceWithNormal returns the block face whose (constant, planar) surface normal points along dir.
func faceWithNormal(t *testing.T, body *topo.Body, dir gmath.Vector3) *topo.Face {
	t.Helper()
	for _, f := range body.Faces() {
		n := f.Geometry().NormalAt(0, 0)
		if float64(n.Dot(dir)) > 0.99 {
			return f
		}
	}
	t.Fatalf("no face with normal %v", dir)
	return nil
}
