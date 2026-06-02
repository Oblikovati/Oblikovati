// SPDX-License-Identifier: GPL-2.0-only

package subd

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
)

func approx(a, b, tol float64) bool { return a-b < tol && b-a < tol }

func TestBoxConvertsToValidSolid(t *testing.T) {
	body := ToBody(Box(2, 2, 2), "box")
	if !body.IsSolid() {
		t.Error("a closed box cage should convert to a solid")
	}
	if got := len(body.Faces()); got != 6 {
		t.Errorf("box has %d faces, want 6", got)
	}
	if got := len(ops.BoundaryEdges(body)); got != 0 {
		t.Errorf("box has %d boundary edges, want 0", got)
	}
	if r := ops.Validate(body); !r.Valid {
		t.Errorf("box body failed validation: %+v", r)
	}
}

func TestPlaneIsOpenSurface(t *testing.T) {
	body := ToBody(Plane(2, 2), "plane")
	if body.IsSolid() {
		t.Error("a single-quad plane cage should be an open surface body")
	}
	if got := len(ops.BoundaryEdges(body)); got != 4 {
		t.Errorf("plane has %d boundary edges, want 4", got)
	}
}

func TestSubdivisionGrowsAndStaysValid(t *testing.T) {
	once := Subdivide(Box(2, 2, 2))
	// One Catmull–Clark step turns 6 faces into 6×4 = 24 quads.
	if got := len(once.Faces); got != 24 {
		t.Errorf("subdivided box has %d faces, want 24", got)
	}
	body := ToBody(once, "subbox")
	if !body.IsSolid() {
		t.Error("subdivided closed box should still be a solid")
	}
	if r := ops.Validate(body); !r.Valid {
		t.Errorf("subdivided box failed validation: %+v", r)
	}
}

func TestSmoothSubdivisionShrinksTowardLimit(t *testing.T) {
	// Without creases, Catmull–Clark rounds the cube: the corner at (0,0,0) moves
	// inward toward the centroid, so the refined body's range box shrinks.
	box := Box(2, 2, 2)
	refined := SubdivideN(box, 2)
	before := ToBody(box, "a").RangeBox().Diagonal()
	after := ToBody(refined, "b").RangeBox().Diagonal()
	if !(after.X < before.X && after.Y < before.Y && after.Z < before.Z) {
		t.Errorf("smooth subdivision should shrink the box: before %v, after %v", before, after)
	}
}

func TestCreasingKeepsCornerSharp(t *testing.T) {
	// Crease all edges of one corner (0,0,0): vertices 0's three edges become sharp,
	// so the corner is a fixed point and stays put under subdivision, while the
	// uncreased opposite corner rounds inward.
	box := Box(2, 2, 2)
	box.Creases = map[[2]int]float64{
		edgeKey(0, 1): 1, edgeKey(0, 3): 1, edgeKey(0, 4): 1,
	}
	refined := SubdivideN(box, 3)
	// The creased corner (0,0,0) is preserved exactly.
	if !nearPoint(refined.Verts[0], math.P3(0, 0, 0), 1e-9) {
		t.Errorf("creased corner moved to %v, want (0,0,0)", refined.Verts[0])
	}
	// A smooth box (no creases) does NOT preserve that corner.
	smooth := SubdivideN(Box(2, 2, 2), 3)
	if nearPoint(smooth.Verts[0], math.P3(0, 0, 0), 1e-6) {
		t.Error("smooth corner should have moved inward, but stayed put")
	}
}

func nearPoint(a, b math.Point3, tol float64) bool {
	return approx(a.X, b.X, tol) && approx(a.Y, b.Y, tol) && approx(a.Z, b.Z, tol)
}
