// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestTangentContactUnionStaysManifold is the M20-F01 regression for a tangent/grazing union.
// When two features joined onto the same body touch along a single line (zero volumetric
// overlap) — e.g. a faceted boss whose rim grazes a flat mounting-lug wall, which is exactly
// what the 28BYJ-48 geared stepper's data-sheet dimensions produce (boss radius == offset −
// lug_w/2) — the naive weld leaves an edge bordered by FOUR faces (two from each operand), a
// non-manifold pinch. The Weiler radial-edge sew resolves it EXACTLY (no displacement): the
// coincident dihedrals are paired by filled wedge and the shared contact vertices are cut into
// per-shell coincident duplicates, so the result is a valid solid AND carries the distinct edge
// identities a downstream re-weld needs — each case must still be a valid solid after a top-rim
// fillet. Clearance and overlap (the non-degenerate neighbours) must stay clean too (ADR-0047).
func TestTangentContactUnionStaysManifold(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		lugBack float64 // Y of the lug wall facing the boss; boss rim reaches y = -0.45
	}{
		{"clearance", -0.50}, // wall short of the boss — two separate contacts on the can
		{"tangent", -0.45},   // wall exactly on the boss rim — the degenerate pinch
		{"overlap", -0.40},   // wall past the boss rim — features interpenetrate
	} {
		t.Run(tc.name, func(t *testing.T) {
			can := cylinderZAt(0, -0.8, 0, 1.9, 1.4, "can")   // Ø2.8 can offset −0.8 in Y
			boss := cylinderZAt(0, 0, -0.15, 0, 0.45, "boss") // Ø0.9 boss on the axis, rim at y=−0.45
			body := joinOrFatal(t, can, boss, "boss")
			lug := box(-2.1, -1.15, -0.085, 4.2, tc.lugBack-(-1.15), 0.085)
			body = joinOrFatal(t, body, lug, "lug")
			assertManifoldSolid(t, body, tc.name)
			assertFilletable(t, body, tc.name)
		})
	}
}

// assertFilletable rounds the body's highest edge (the can's top rim, far from the contact) and
// fails unless the result is a valid solid — the end-to-end proof that the tangent fix left no
// coincident edge for the fillet's re-weld to collapse back to non-manifold.
func assertFilletable(t *testing.T, b *topo.Body, label string) {
	t.Helper()
	var top *topo.Edge
	bz := -1e18
	for _, e := range b.Edges() {
		vs := e.Vertices()
		if z := (vs[0].Point().Z + vs[len(vs)-1].Point().Z) / 2; z > bz {
			bz, top = z, e
		}
	}
	out, err := ops.FilletEdges(b, [][]byte{top.ReferenceKey()}, 0.1)
	if err != nil {
		t.Fatalf("%s: fillet after tangent union failed: %v", label, err)
	}
	if r := ops.Validate(out); !r.Valid {
		t.Fatalf("%s: filleted body invalid: %v", label, r.Issues)
	}
}

// TestTangentBoxEdgeUnion exercises the resolver on the simplest exact tangency: two boxes
// kitty-corner so they meet ONLY along one vertical line (no shared face, no overlap), each
// also joined to a base plate so the result is one body. The shared vertical edge is bordered
// by four planar faces (two from each box) and must split into two manifold dihedrals.
func TestTangentBoxEdgeUnion(t *testing.T) {
	t.Parallel()
	base := box(-1, -1, 0, 2, 2, 0.2) // a plate both blocks sit on (keeps it one body)
	a := box(-1, -1, 0, 1, 1, 1)      // x[-1,0] y[-1,0]
	b := box(0, 0, 0, 1, 1, 1)        // x[0,1]  y[0,1]; touches `a` only along the (0,0) vertical line
	body := joinOrFatal(t, base, a, "a")
	body = joinOrFatal(t, body, b, "b")
	assertManifoldSolid(t, body, "box-corner-tangent")
}

// assertManifoldSolid fails unless every edge borders exactly two faces and ops.Validate passes.
func assertManifoldSolid(t *testing.T, b *topo.Body, label string) {
	t.Helper()
	open, nonManifold := 0, 0
	for _, e := range b.Edges() {
		switch n := len(e.Faces()); {
		case n < 2:
			open++
		case n > 2:
			nonManifold++
		}
	}
	if open != 0 || nonManifold != 0 {
		t.Fatalf("%s: not manifold: open=%d nonManifold=%d", label, open, nonManifold)
	}
}
