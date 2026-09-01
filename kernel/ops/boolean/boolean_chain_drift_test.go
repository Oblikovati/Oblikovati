// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Chained-boolean drift guard (M2 Phase 3, Oblikovati/Oblikovati#1336, acceptance #4 of #1320). Five
// sequential curved booleans must leave the cumulative volume within tolerance and every intermediate
// body a watertight manifold solid. The umbrella's worry is csg.js fragility — chained triangle-soup CSG
// accumulates noise and loses watertightness. The exact drilled-plate path (slice A) plus multi-hole
// drilling (this slice) keep every bore an EXACT curved B-rep, so the chain neither drifts nor opens: the
// test asserts the result stays exact (one analytic cylinder wall PER bore, never faceted CSG) and the
// final volume equals the slab minus five single-bore removals.

// chainRod builds a validated cylinder long enough to pierce the slab's two z-faces, centred at (cx, cy).
func chainRod(t *testing.T, cx, cy float64) *topo.Body {
	t.Helper()
	rod, err := brep.SolidCylinder(math.P3(cx, cy, -1), math.V3(0, 0, 1), 1.5, 8)
	if err != nil {
		t.Fatalf("SolidCylinder at (%g,%g): %v", cx, cy, err)
	}
	return rod
}

// chainSlab is the 32×20×6 plate every chain test bores; a fresh body each call so no test mutates
// another's operand. It is wide enough that all five bores sit clear of the edges and each other.
func chainSlab(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(math.P3(-16, -10, 0), math.P3(16, 10, 6), "slab")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return b
}

// cylinderFaceCount reports how many of a body's faces are analytic cylinders — the exactness witness: a
// drilled plate that fell back to CSG would have a faceted (planar-triangle) wall and report zero.
func cylinderFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}

// assertChainSolid fails unless b is a watertight manifold solid; a chain step that opens is the
// tjunctionFaceBudget watertightness regression acceptance #3 forbids.
func assertChainSolid(t *testing.T, b *topo.Body, step string) {
	t.Helper()
	r := ops.Validate(b)
	if !r.Valid || !r.Closed || !r.Manifold || !b.IsSolid() {
		t.Fatalf("%s: result not a watertight solid (valid=%v closed=%v manifold=%v solid=%v)",
			step, r.Valid, r.Closed, r.Manifold, b.IsSolid())
	}
}

// TestCurvedBooleanChainDriftBores bores five identical, non-overlapping holes through the plate in
// sequence. Each stays EXACT (the running body gains exactly one analytic cylinder wall per bore), so the
// cumulative volume is the plate minus five single-bore removals with no drift, and no body ever opens.
func TestCurvedBooleanChainDriftBores(t *testing.T) {
	t.Parallel()
	v0 := query.BodyGeometryProperties(chainSlab(t), ops.DefaultQuality()).Volume
	centers := []float64{-8, -4, 0, 4, 8} // spacing 4 > 2r=3: the bores never touch

	ref, err := ops.Boolean(ops.Cut, chainSlab(t), chainRod(t, centers[0], 0))
	if err != nil {
		t.Fatalf("reference bore: %v", err)
	}
	assertChainSolid(t, ref, "reference bore")
	removed := v0 - query.BodyGeometryProperties(ref, ops.DefaultQuality()).Volume

	res := chainSlab(t)
	for i, cx := range centers {
		out, err := ops.Boolean(ops.Cut, res, chainRod(t, cx, 0))
		if err != nil {
			t.Fatalf("bore %d at x=%g: %v", i, cx, err)
		}
		assertChainSolid(t, out, fmt.Sprintf("bore %d", i))
		if got := cylinderFaceCount(out); got != i+1 {
			t.Fatalf("bore %d: %d cylinder faces, want %d — a bore fell back to faceted CSG instead of an exact hole", i, got, i+1)
		}
		res = out
	}

	got, want := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume, v0-5*removed
	if drift := stdmath.Abs(got - want); drift > 0.005*v0 {
		t.Errorf("five chained bores: volume %.4f, want %.4f (drift %.4f > %.4f = 0.5%% of slab)",
			got, want, drift, 0.005*v0)
	}
}

// TestCurvedBooleanChainDriftOrderIndependent: the same five holes drilled in a shuffled order reach the
// same exact volume — chaining is path-independent, not just stable for one ordering.
func TestCurvedBooleanChainDriftOrderIndependent(t *testing.T) {
	t.Parallel()
	want := query.BodyGeometryProperties(chainSlab(t), ops.DefaultQuality()).Volume
	ref, _ := ops.Boolean(ops.Cut, chainSlab(t), chainRod(t, 0, 0))
	want -= 5 * (query.BodyGeometryProperties(chainSlab(t), ops.DefaultQuality()).Volume - query.BodyGeometryProperties(ref, ops.DefaultQuality()).Volume)

	res := chainSlab(t)
	for _, cx := range []float64{0, 8, -4, 4, -8} { // shuffled
		out, err := ops.Boolean(ops.Cut, res, chainRod(t, cx, 0))
		if err != nil {
			t.Fatalf("bore at x=%g: %v", cx, err)
		}
		res = out
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	if stdmath.Abs(got-want) > 0.005*want {
		t.Errorf("shuffled-order chain volume %.4f, want %.4f (drift > 0.5%%)", got, want)
	}
}
