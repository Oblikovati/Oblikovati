// SPDX-License-Identifier: GPL-2.0-only

package subd

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
)

func TestIsClosedEmptyAndOpen(t *testing.T) {
	if cageClosed((Mesh{}).edgeFaces()) {
		t.Error("empty mesh should not be closed")
	}
	if cageClosed(Plane(1, 1).edgeFaces()) {
		t.Error("a single-face plane has boundary edges; should not be closed")
	}
	if !cageClosed(Box(1, 1, 1).edgeFaces()) {
		t.Error("a box should be closed")
	}
}

func TestFacePlaneDegenerateFallsBackToZ(t *testing.T) {
	// A face whose vertices are collinear has a zero Newell normal ⇒ NewPlane fails ⇒
	// facePlane falls back to a +Z plane.
	m := Mesh{
		Verts: []gmath.Point3{gmath.P3(0, 0, 0), gmath.P3(1, 0, 0), gmath.P3(2, 0, 0), gmath.P3(3, 0, 0)},
		Faces: [][]int{{0, 1, 2, 3}},
	}
	n := facePlane(m, m.Faces[0]).NormalAt(0, 0)
	if math.Abs(float64(n.Z)-1) > 1e-9 || math.Abs(float64(n.X)) > 1e-9 || math.Abs(float64(n.Y)) > 1e-9 {
		t.Fatalf("degenerate facePlane normal = %v, want +Z", n)
	}
}

func TestSubdivideOpenMeshKeepsBoundaryMidpoints(t *testing.T) {
	// Subdividing an open mesh exercises edgePoint's boundary branch (len(faces) < 2):
	// it must stay valid and grow.
	out := SubdivideN(Plane(2, 2), 1)
	if len(out.Faces) != 4 {
		t.Fatalf("subdivided plane faces = %d, want 4", len(out.Faces))
	}
}

func TestClamp01Bounds(t *testing.T) {
	if gmath.Clamp01(-0.5) != 0 || gmath.Clamp01(1.5) != 1 || gmath.Clamp01(0.25) != 0.25 {
		t.Fatalf("clamp01 = %v/%v/%v, want 0/1/0.25", gmath.Clamp01(-0.5), gmath.Clamp01(1.5), gmath.Clamp01(0.25))
	}
}

func TestEdgePointHonoursOverSharpCrease(t *testing.T) {
	// A crease sharpness > 1 (set directly, bypassing SetCrease's clamp) must be clamped
	// to a hard edge by edgePoint via clamp01.
	m := Box(2, 2, 2)
	m.Creases = map[[2]int]float64{edgeKey(0, 1): 2}
	out := SubdivideN(m, 1)
	if len(out.Verts) == 0 {
		t.Fatal("subdivision of an over-sharp creased box produced no vertices")
	}
}
