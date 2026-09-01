// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// trisArea sums the absolute area of triangles over their own vertex list.
func trisArea(verts []math.Point2, tris [][3]int) float64 {
	var a float64
	for _, t := range tris {
		a += triArea(verts[t[0]], verts[t[1]], verts[t[2]])
	}
	return a
}

// TestUnionTrisTwoOverlappingSquares: outer 10×10 minus two 3×3 holes overlapping in a 1×1
// square ⇒ union area 9+9−1 = 17, material 83 — area-exact (#873).
func TestUnionTrisTwoOverlappingSquares(t *testing.T) {
	t.Parallel()
	outer := rectHole(0, 0, 10, 10)
	holes := [][]math.Point2{rectHole(3, 3, 6, 6), rectHole(5, 5, 8, 8)}
	verts, tris := unionTris(outer, holes)
	if got := trisArea(verts, tris); stdmath.Abs(got-83) > 1e-6 {
		t.Errorf("union area = %g, want 83 (100 − union 17)", got)
	}
}

// TestUnionTrisGridHoles: outer 10×10 minus a grid of 3 vertical + 3 horizontal crossing bars
// (width 0.4, length 8) ⇒ union 9.6+9.6−1.44 = 17.76, material 82.24 — the exact area earcut on
// the raw overlapping loops could not produce.
func TestUnionTrisGridHoles(t *testing.T) {
	t.Parallel()
	outer := rectHole(0, 0, 10, 10)
	var holes [][]math.Point2
	for _, x := range []float64{3, 5, 7} {
		holes = append(holes, rectHole(x-0.2, 1, x+0.2, 9))
	}
	for _, y := range []float64{3, 5, 7} {
		holes = append(holes, rectHole(1, y-0.2, 9, y+0.2))
	}
	verts, tris := unionTris(outer, holes)
	if got := trisArea(verts, tris); stdmath.Abs(got-82.24) > 1e-6 {
		t.Errorf("union area = %g, want 82.24 (100 − union 17.76)", got)
	}
}

// TestHolesOverlapDetection: crossing/overlapping loops are detected; disjoint ones are not (so
// the fast direct-earcut path is kept for clean faces).
func TestHolesOverlapDetection(t *testing.T) {
	t.Parallel()
	overlapping := [][]math.Point2{rectHole(3, 3, 6, 6), rectHole(5, 5, 8, 8)}
	if !holesOverlap(overlapping) {
		t.Error("overlapping holes not detected")
	}
	disjoint := [][]math.Point2{rectHole(1, 1, 3, 9), rectHole(4, 1, 6, 9), rectHole(7, 1, 9, 9)}
	if holesOverlap(disjoint) {
		t.Error("disjoint holes wrongly flagged as overlapping")
	}
}

// TestUnionHoledMeshArea lifts the union to 3D (the holedPlanarMesh overlap path) and checks the
// mesh area equals outer − union, exercising the orthonormal project/unproject round-trip.
func TestUnionHoledMeshArea(t *testing.T) {
	t.Parallel()
	normal := math.V3(0, 0, 1)
	lift := func(x, y float64) math.Point3 { return math.P3(x, y, 5) }
	outer3D := []math.Point3{lift(0, 0), lift(10, 0), lift(10, 10), lift(0, 10)}
	hole := func(x0, y0, x1, y1 float64) []math.Point3 {
		return []math.Point3{lift(x0, y0), lift(x1, y0), lift(x1, y1), lift(x0, y1)}
	}
	holes3D := [][]math.Point3{hole(3, 3, 6, 6), hole(5, 5, 8, 8)}
	m := unionHoledMesh(outer3D, holes3D, normal)
	if got := meshArea(m); stdmath.Abs(got-83) > 1e-4 {
		t.Errorf("3D mesh area = %g, want 83 (100 − union 17)", got)
	}
}

// TestHoledPlanarMeshRoutesOverlapToUnion checks the integrated tessellation entry routes an
// overlapping-hole face through the union path (area-exact), exercising holesOverlap + the
// projector handoff.
func TestHoledPlanarMeshRoutesOverlapToUnion(t *testing.T) {
	t.Parallel()
	normal := math.V3(0, 0, 1)
	flat := planeProjector(normal)
	lift := func(x, y float64) math.Point3 { return math.P3(x, y, 0) }
	outer3D := []math.Point3{lift(0, 0), lift(10, 0), lift(10, 10), lift(0, 10)}
	hole := func(x0, y0, x1, y1 float64) []math.Point3 {
		return []math.Point3{lift(x0, y0), lift(x1, y0), lift(x1, y1), lift(x0, y1)}
	}
	holes3D := [][]math.Point3{hole(3, 3, 6, 6), hole(5, 5, 8, 8)}
	m := holedPlanarMesh(project2D(outer3D, flat), outer3D, holes3D, flat, normal)
	if got := meshArea(m); stdmath.Abs(got-83) > 1e-4 {
		t.Errorf("holedPlanarMesh area = %g, want 83 (overlap routed to union)", got)
	}
}
