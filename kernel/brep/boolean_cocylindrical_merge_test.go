// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The cocylindrical wall merge's corpus (ADR-0061 stage 5). Two kept faces on ONE surface that share
// a boundary are one face; two that share none stay two.

// cylinderWalls counts a body's analytic cylinder faces.
func cylinderWalls(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}

// TestBoreContinuingABoreIsOneWall: a blind bore drilled from the top, then continued to the bottom by
// a second coaxial drill of the SAME radius. The two tools' walls lie on one surface and meet at the
// circle where the first ended, which bounds nothing — the bore is one hole.
func TestBoreContinuingABoreIsOneWall(t *testing.T) {
	t.Parallel()
	blk, err := SolidBlock(math.P3(-3, -3, 0), math.P3(3, 3, 6), "blk")
	if err != nil {
		t.Fatalf("block: %v", err)
	}
	upper, _ := SolidCylinder(math.P3(0, 0, 3), math.V3(0, 0, 1), 1, 4)
	lower, _ := SolidCylinder(math.P3(0, 0, -1), math.V3(0, 0, 1), 1, 4)
	step, err := Boolean(Difference, blk, upper)
	if err != nil {
		t.Fatalf("first bore: %v", err)
	}
	through, err := Boolean(Difference, step, lower)
	if err != nil {
		t.Fatalf("continuing bore: %v", err)
	}
	assertWatertight(t, through)
	if n := cylinderWalls(through); n != 1 {
		t.Errorf("the continued bore has %d cylinder walls, want 1", n)
	}
}

// TestTwoBoresWithNoSharedEdgeStayTwoFaces is the NEGATIVE row: two bores on ONE surface — same axis,
// same radius, so geom.SurfacesCoincide answers yes — with a wall of material between them. They share
// no edge, so nothing dissolves and the body keeps both walls. A merge that fired on the surface alone
// would weld two holes into one and delete the material between them.
func TestTwoBoresWithNoSharedEdgeStayTwoFaces(t *testing.T) {
	t.Parallel()
	blk, err := SolidBlock(math.P3(-3, -3, 0), math.P3(3, 3, 10), "blk")
	if err != nil {
		t.Fatalf("block: %v", err)
	}
	bottom, _ := SolidCylinder(math.P3(0, 0, -1), math.V3(0, 0, 1), 1, 4) // z ∈ [−1, 3]
	top, _ := SolidCylinder(math.P3(0, 0, 7), math.V3(0, 0, 1), 1, 4)     // z ∈ [7, 11]
	one, err := Boolean(Difference, blk, bottom)
	if err != nil {
		t.Fatalf("first bore: %v", err)
	}
	both, err := Boolean(Difference, one, top)
	if err != nil {
		t.Fatalf("second bore: %v", err)
	}
	assertWatertight(t, both)
	if n := cylinderWalls(both); n != 2 {
		t.Errorf("two bores separated by material have %d cylinder walls, want 2 — they share no edge", n)
	}
}

// TestMergedWallCarriesBothParentsKeys: every reference key that existed on either parent still
// resolves on the merged face (ADR-0043 — a pick must survive the operation that consumed it).
func TestMergedWallCarriesBothParentsKeys(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	b, _ := SolidCylinder(math.P3(0, 0, 4), math.V3(0, 0, 1), 2, 3)
	keyA, keyB := wallKeyOf(t, a), wallKeyOf(t, b)
	res, err := Boolean(Union, a, b)
	if err != nil {
		t.Fatalf("union: %v", err)
	}
	for _, k := range [][]byte{keyA, keyB} {
		f, ok := res.FindFaceByKey(k)
		if !ok {
			t.Fatalf("the merged wall does not resolve parent key %q", string(k))
		}
		if _, isCyl := f.Geometry().(geom.Cylinder); !isCyl {
			t.Errorf("parent key %q resolved to a %T, not the merged cylinder wall", string(k), f.Geometry())
		}
	}
}

// wallKeyOf is the reference key of a cylinder primitive's wall face.
func wallKeyOf(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return f.ReferenceKey()
		}
	}
	t.Fatal("the body has no cylinder wall")
	return nil
}

// TestMergedWallCarriesTheChartItsLoopsDetermine: the merged face's parametric trim is the union of
// the two in the covering space, and it is RECORDED — a reader must not have to re-derive it, and a
// pair whose fused loops determine none is declined instead (chartedMerge).
func TestMergedWallCarriesTheChartItsLoopsDetermine(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	b, _ := SolidCylinder(math.P3(0, 0, 4), math.V3(0, 0, 1), 2, 3)
	res, err := Boolean(Union, a, b)
	if err != nil {
		t.Fatalf("union: %v", err)
	}
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); !ok {
			continue
		}
		assertChartSpansTheUnion(t, f)
		return
	}
	t.Fatal("the union kept no cylinder wall")
}

// assertChartSpansTheUnion checks the merged wall's chart covers BOTH parents' axial runs: v reaches
// from the lower band's base to the upper band's top, which is what "the charts were unioned" means.
func assertChartSpansTheUnion(t *testing.T, f *topo.Face) {
	t.Helper()
	chart := f.Chart()
	if len(chart) == 0 {
		t.Fatal("the merged wall carries no chart; its loops determine one and it must be recorded")
	}
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, c := range chart {
		for _, p := range c {
			lo, hi = stdmath.Min(lo, float64(p.Y)), stdmath.Max(hi, float64(p.Y))
		}
	}
	if stdmath.Abs(lo) > 1e-9 || stdmath.Abs(hi-7) > 1e-9 {
		t.Errorf("the merged chart's v span is [%v, %v], want [0, 7] — the union of the two bands", lo, hi)
	}
}

// TestMergeDeclinesRatherThanGuessAChart records that the merge is a post-conditioned operation: a
// recorder is threaded to it, and the code it reports names the configuration it left alone.
func TestMergeDeclinesRatherThanGuessAChart(t *testing.T) {
	t.Parallel()
	if CodeCocylindricalMergeUndecided == "" {
		t.Fatal("the merge's decline has no code to report")
	}
	rec := &diag.Recorder{}
	rec.Recordf(CodeCocylindricalMergeUndecided, diag.Defect, "probe")
	if !rec.Has(CodeCocylindricalMergeUndecided) {
		t.Error("the merge's decline code does not reach a recorder")
	}
}
