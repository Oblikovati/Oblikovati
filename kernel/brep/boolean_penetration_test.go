// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestBarThroughBoredWallUnion is the kernel regression for #860 (the parametric-fan blade
// defect, distilled): a thin bar whose bottom is COPLANAR with a bored body's bottom is unioned
// on so it crosses the faceted bore wall — a partial penetration of a concave faceted surface.
// Before the arrangement-robustness fixes (collinear-edge imprint capture, coplanar imprint
// material-clip, T-junction welding, and filled-hole merge) the union welded the bar bottom as a
// coincident non-manifold membrane along the bore wall. It must come back a clean manifold solid
// whether the bar stops short of, exactly touches, or pokes through the bore wall.
func TestBarThroughBoredWallUnion(t *testing.T) {
	// Body: 5×5×1.5 box, bored (r=2.35) with a re-entrant hub (r=0.675) joined in the centre —
	// so the bar's outer end crosses the CONCAVE bored wall, the hard case.
	body := box(-2.5, -2.5, 0, 5, 5, 1.5)
	body = cutOrFatal(t, body, cylinderZAt(0, 0, -0.5, 2.0, 2.35, "bore"), "bore")
	body = joinOrFatal(t, body, cylinderZAt(0, 0, 0, 1.5, 0.675, "hub"), "hub")

	// The bar inner end is over the hub; the outer end (x) sweeps from inside the bore void, to
	// exactly the bore radius, to past it into the frame material.
	for _, tip := range []float64{2.30, 2.34, 2.35, 2.36, 2.40} {
		bar := box(0.6, -0.08, 0, tip-0.6, 0.16, 0.6)
		joined, err := brep.Boolean(brep.Union, body, bar)
		if err != nil {
			t.Fatalf("tip=%.2f: union: %v", tip, err)
		}
		assertCleanSolid(t, joined, tip)
	}
}

// assertCleanSolid fails if the body has any boundary (1-face) or non-manifold (>2-face) edge.
func assertCleanSolid(t *testing.T, b *topo.Body, tip float64) {
	t.Helper()
	if b == nil {
		t.Fatalf("tip=%.2f: nil result", tip)
	}
	open, nonManifold := 0, 0
	for _, e := range b.Edges() {
		switch n := len(e.Faces()); {
		case n < 2:
			open++
		case n > 2:
			nonManifold++
		}
	}
	if r := ops.Validate(b); !r.Valid || open != 0 || nonManifold != 0 {
		t.Errorf("tip=%.2f: not a clean manifold solid: valid=%v open=%d nonManifold=%d issues=%v",
			tip, r.Valid, open, nonManifold, r.Issues)
	}
}
