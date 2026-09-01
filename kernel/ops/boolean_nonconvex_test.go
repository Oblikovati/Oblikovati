// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
)

// Regression for Oblikovati/Oblikovati#1315: classify decided containment from vertex-in-solid only,
// which is unsound for non-convex bodies. The fixture below is the canonical counterexample — every
// vertex of the inner body lies inside the outer body's MATERIAL, yet the inner body crosses the
// outer's boundary, so the pair must be classified `intersecting`, not contained.

// zPrism extrudes a closed XY polygon between z0 and z1 into a solid prism (winding per the proven
// brep prismBody helper: bottom reversed, top forward, quad sides).
func zPrism(poly []m.Point2, z0, z1 float64, feat string) *topo.Body {
	n := len(poly)
	verts := make([]m.Point3, 0, n*2)
	for _, p := range poly {
		verts = append(verts, m.P3(p.X, p.Y, m.Scalar(z0)))
	}
	for _, p := range poly {
		verts = append(verts, m.P3(p.X, p.Y, m.Scalar(z1)))
	}
	bottom, top := make([]int, n), make([]int, n)
	for i := range poly {
		bottom[i] = n - 1 - i
		top[i] = n + i
	}
	faces := [][]int{bottom, top}
	for i := range poly {
		next := (i + 1) % n
		faces = append(faces, []int{i, next, next + n, i + n})
	}
	return subd.ToBody(subd.Mesh{Verts: verts, Faces: faces}, feat)
}

// uSlotPrism builds the U-shaped outer body: the block [0,10]x[0,10] with the top-centre slot
// [3,7]x[4,10] removed, extruded z in [0,4]. Material volume = (100-24)*4 = 304.
func uSlotPrism() *topo.Body {
	poly := []m.Point2{
		m.P2(0, 0), m.P2(10, 0), m.P2(10, 10),
		m.P2(7, 10), m.P2(7, 4), m.P2(3, 4), m.P2(3, 10), m.P2(0, 10),
	}
	return zPrism(poly, 0, 4, "u-slot")
}

// TestClassifyNonConvexVertexInsideButBoundaryCrosses is the core #1315 assertion: the bar's eight
// corners all lie in the U's two arms (inside its material), but the bar spans the empty slot and so
// crosses the slot walls. classify must report `intersecting`, not targetContainsTool.
func TestClassifyNonConvexVertexInsideButBoundaryCrosses(t *testing.T) {
	t.Parallel()
	outer := uSlotPrism()
	bar, err := brep.SolidBlock(m.P3(2, 5.5, 1), m.P3(8, 6.5, 3), "bar")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	// Precondition: every vertex of the bar is inside the U's material (the unsound test passes).
	if !allVerticesInside(bar, outer) {
		t.Fatal("test premise broken: not all bar vertices are inside the U material")
	}
	// Precondition: the boundaries genuinely cross (the bar pierces the slot walls).
	if !query.BoundariesCross(outer, bar) {
		t.Fatal("test premise broken: boundaries do not cross")
	}
	if rel := classify(outer, bar); rel != intersecting {
		t.Errorf("classify = %v, want intersecting (vertex-only containment is unsound here)", rel)
	}
	if rel := classify(bar, outer); rel != intersecting {
		t.Errorf("classify(bar,outer) = %v, want intersecting", rel)
	}
}

// TestNonConvexJoinVolumeMatchesAnalytic checks the boolean now routed through booleanGeneral yields
// the correct union Volume: V(A) + V(B) - V(A∩B) = 304 + 12 - 4 = 312.
func TestNonConvexJoinVolumeMatchesAnalytic(t *testing.T) {
	t.Parallel()
	outer := uSlotPrism()
	bar, err := brep.SolidBlock(m.P3(2, 5.5, 1), m.P3(8, 6.5, 3), "bar")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	joined, err := Boolean(Join, outer, bar)
	if err != nil {
		t.Fatalf("Boolean(Join): %v", err)
	}
	got := query.BodyGeometryProperties(joined, DefaultQuality()).Volume
	const want = 312.0
	if math.Abs(got-want) > 1e-6*want {
		t.Errorf("union volume = %g, want %g", got, want)
	}
	if !Validate(joined).ValidSolid() {
		t.Errorf("union is not a valid closed manifold solid")
	}
}

// TestGenuineContainmentStillFastPaths guards against a perf/behaviour regression: a strictly interior
// tool (no boundary crossing) must still be recognized as contained, taking the fast path.
func TestGenuineContainmentStillFastPaths(t *testing.T) {
	t.Parallel()
	outer, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(10, 10, 10), "outer")
	if err != nil {
		t.Fatalf("outer: %v", err)
	}
	inner, err := brep.SolidBlock(m.P3(3, 3, 3), m.P3(6, 6, 6), "inner")
	if err != nil {
		t.Fatalf("inner: %v", err)
	}
	if rel := classify(outer, inner); rel != targetContainsTool {
		t.Errorf("classify = %v, want targetContainsTool (strict interior)", rel)
	}
	if query.BoundariesCross(outer, inner) {
		t.Error("strictly interior tool should not cross the outer boundary")
	}
}
