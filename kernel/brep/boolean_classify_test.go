// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// classifyPolygonPrism builds an n-gon prism body for classification fixtures.
func classifyPolygonPrism(t *testing.T, r float64, n int, z0, z1 float64, name string) *topo.Body {
	t.Helper()
	verts := make([]math.Point3, 0, n*2)
	for _, z := range []float64{z0, z1} {
		for i := 0; i < n; i++ {
			a := 2 * stdmath.Pi * float64(i) / float64(n)
			verts = append(verts, math.P3(r*stdmath.Cos(a), r*stdmath.Sin(a), z))
		}
	}
	bottom, top := make([]int, n), make([]int, n)
	for i := 0; i < n; i++ {
		bottom[i], top[i] = n-1-i, n+i
	}
	faces := [][]int{bottom, top}
	for i := 0; i < n; i++ {
		faces = append(faces, []int{i, (i + 1) % n, (i+1)%n + n, i + n})
	}
	return subd.ToBody(subd.Mesh{Verts: verts, Faces: faces}, name)
}

// TestInsideSolidWindingBasics: the winding classifier must agree with the obvious answers on a
// box and on a holed (annular) prism — including a query point down the hole, which only a
// hole-aware classification gets right (#1599).
func TestInsideSolidWindingBasics(t *testing.T) {
	blk, _ := SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "b")
	if !insideSolid(blk, math.P3(0.5, 0.5, 0.5)) || insideSolid(blk, math.P3(2, 2, 2)) {
		t.Error("box winding classification wrong at the trivial points")
	}
	ring := classifyRing(t)
	if !insideSolid(ring, math.P3(0.74, 0, 0.3)) {
		t.Error("point in the ring material classified outside")
	}
	if insideSolid(ring, math.P3(0, 0, 0.3)) {
		t.Error("point down the ring's hole classified inside")
	}
}

// classifyRing cuts an inner prism out of an outer one — the annular fixture whose top/bottom
// planes coincide with other operands in the beltb/faceplate regressions.
func classifyRing(t *testing.T) *topo.Body {
	t.Helper()
	outer := classifyPolygonPrism(t, 0.8, 64, 0, 0.6, "outer")
	inner := classifyPolygonPrism(t, 0.68, 32, -0.05, 0.65, "inner")
	ring, err := Boolean(Difference, outer, inner)
	if err != nil {
		t.Fatalf("ring cut: %v", err)
	}
	return ring
}

// TestInsideSolidCoplanarQueryPoint is the beltb regression (#1599): a query point lying ON a
// face's PLANE — the systematic case when two operands share a top/bottom plane — must classify
// by the rest of the boundary. Evaluating the solid-angle fan at a coplanar point instead trips
// the atan2 sign-of-zero degeneracy and fabricates a spurious ±2π, which put hole points of the
// beltb ring "inside" and tore the union open.
func TestInsideSolidCoplanarQueryPoint(t *testing.T) {
	ring := classifyRing(t)
	if insideSolid(ring, math.P3(0, 0.0587, 0)) {
		t.Error("point in the hole ON the ring's bottom plane classified inside (coplanar ±2π degeneracy)")
	}
	if insideSolid(ring, math.P3(2, 0, 0)) {
		t.Error("point outside the ring ON its bottom plane classified inside")
	}
	if insideSolid(ring, math.P3(0, 0, 0.6)) {
		t.Error("point in the hole ON the ring's top plane classified inside")
	}
}

// TestInsideSolidEdgeGrazingDirection is the adversarial fixture for the RETIRED fixed-ray parity
// classifier (#1599): a query point placed so a ray along the old magic direction
// (0.5773, 0.5774, 0.5775) passes exactly through a box edge (two faces' shared boundary) or a
// corner (three faces'), where parity counting was ambiguous. The winding number integrates the
// whole boundary — there is no direction to graze — so these must classify exactly.
func TestInsideSolidEdgeGrazingDirection(t *testing.T) {
	blk, _ := SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "b")
	dir := math.V3(0.5773, 0.5774, 0.5775)
	inside := math.P3(1, 2, 2).TranslateBy(dir.Scale(-1)) // the old ray exits exactly through the top edge midpoint
	if !insideSolid(blk, inside) {
		t.Errorf("interior point %v (old ray through an edge) classified outside", inside)
	}
	outside := math.P3(2, 2, 2).TranslateBy(dir.Scale(0.5)) // beyond the corner; the reversed old ray passes through it
	if insideSolid(blk, outside) {
		t.Errorf("exterior point %v (old ray through a corner) classified inside", outside)
	}
}
