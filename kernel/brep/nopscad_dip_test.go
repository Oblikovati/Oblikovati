// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// dipOctagon is the NopSCADlib DIP cross-section (vitamins/dip.scad dip(), pdip(8) ⇒
// size=[8.89, 6.35, 3] mm): the hull of a wide thin rect (the parting-line flash) and a
// narrower tall rect (the body) ⇒ an octagon with the DIP "wings". XZ plane, CCW, cm.
func dipOctagon() []math.Point3 {
	return []math.Point3{
		math.P3(-0.2675, 0, 0.15), math.P3(-0.3175, 0, 0.0125), math.P3(-0.3175, 0, -0.0125),
		math.P3(-0.2675, 0, -0.15), math.P3(0.2675, 0, -0.15), math.P3(0.3175, 0, -0.0125),
		math.P3(0.3175, 0, 0.0125), math.P3(0.2675, 0, 0.15),
	}
}

const dipHalfLen = 0.4445 // size.x/2 = 8.89/2 mm

// TestNopDipCSG models the DIP IC package body by stacking the colliding booleans NopSCADlib
// uses: the chamfered hull octagon is extruded along the length, a central 4 mm slot is CUT —
// which DISCONNECTS the bar into two separate wings — then a lower cube and an upper block are
// JOINED back in along faces COPLANAR with the slot walls, re-welding the two wings into one
// watertight solid. Exercises (a) a cut that splits a body into two lumps and (b) a union that
// re-welds along coplanar faces — the kind of feature interaction simpler parts don't surface.
//
// The pin-1 index notch (a blind half-cylinder cut into the top of one end) is split out into
// TestBlindStraddleCurvedCutWatertight below: it currently breaks watertightness and is the
// open spec for that bug, so it is kept out of this (passing) body coverage.
func TestNopDipCSG(t *testing.T) {
	t.Parallel()
	body := prismBodyAlongY(dipOctagon(), -dipHalfLen, dipHalfLen, "dip-bar")

	// Central 4 mm slot, over-sized in height + length ⇒ splits the bar into two wings.
	body = cutOrFatal(t, body, box(-0.2, -0.5, -0.2, 0.4, 1.0, 0.4), "dip slot")

	// Re-weld: a lower cube (z −1.5..0.9 mm) and an upper block (z 0..1.5 mm) fill the slot
	// exactly — their x=±0.2 walls are coplanar with the wing inner walls.
	body = joinOrFatal(t, body, box(-0.2, -dipHalfLen, -0.15, 0.4, 2*dipHalfLen, 0.24), "dip lower spine")
	body = joinOrFatal(t, body, box(-0.2, -dipHalfLen, 0, 0.4, 2*dipHalfLen, 0.15), "dip upper block")

	requireValidNopSolid(t, "dip", body)

	// Slot + refill is volume-neutral ⇒ the body is exactly the octagon prism.
	octArea := nopPolygonArea([]math.Point3{ // shoelace on (X,Z): pass Z as the Y coordinate
		math.P3(-0.2675, 0.15, 0), math.P3(-0.3175, 0.0125, 0), math.P3(-0.3175, -0.0125, 0),
		math.P3(-0.2675, -0.15, 0), math.P3(0.2675, -0.15, 0), math.P3(0.3175, -0.0125, 0),
		math.P3(0.3175, 0.0125, 0), math.P3(0.2675, 0.15, 0),
	})
	want := octArea * 2 * dipHalfLen
	if got := vol(body); stdmath.Abs(got-want)/want > 0.01 {
		t.Errorf("dip body volume = %.6f cm^3, want ~%.6f (octagon prism; slot+refill neutral)", got, want)
	}
}

// TestBlindStraddleCurvedCutWatertight is the open spec for a planar-B-rep boolean bug found by
// the DIP index notch. Cutting a vertical cylinder that is BLIND (its bottom cap lies inside the
// body) AND STRADDLES a boundary face (its axis lies in the body's +Y end plane, so its flat
// diametral side is coincident with that boundary) leaves the result NON-watertight: the curved
// cut's imprint onto the TOP plane collapses into ~4 near-degenerate (≈0-length) sliver edges
// clustered at one node on the cut arc, which never stitch closed.
//
// It is narrow and coordinate-sensitive — through-cuts, interior blind cuts, and the same notch
// on a RECTANGULAR prism all stay watertight; only blind + boundary-straddle + a non-rectangular
// end cap trips it.
//
// Root cause (deep-dived 2026-06-09): the kept sub-faces all use the cylinder facet vertex
// (e.g. (0.08334,0.31978,0.15)) consistently, but the final shell has unpaired sliver edges at
// (0.0835x,0.3199x,0.15) — a CROSS-FACE vertex disagreement of ~2.5e-4 between the top-plane
// cut arc and the cut-wall/end sub-faces at the tool-facet ∧ end-boundary triple junction.
// splitRingTJunctions then inserts the off vertex into the top loop as a non-pairable spur.
// Two principled fixes were tried and REVERTED (neither closes it, proving it is a genuine
// arrangement-decomposition tangle, not a weld gap): a stitch crack-heal (re-weld when open) up
// to tol 3e-3, and a 3e-4 vertex-snap in arrange2d. The real fix must make the imprint/clip emit
// consistent vertices across faces at such junctions — a substantial change for a dedicated,
// full-suite-guarded session. Skipped until then.
//
// Repro distilled from NopSCADlib dip()'s pin-1 index notch (circle d=3 at y=size.x/2).
func TestBlindStraddleCurvedCutWatertight(t *testing.T) {
	t.Parallel()
	body := prismBodyAlongY(dipOctagon(), -dipHalfLen, dipHalfLen, "bar")
	body = cutOrFatal(t, body, cylinderZAt(0, dipHalfLen, 0, 0.16, 0.15, "notch"), "index notch")
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Fatalf("blind straddle notch left %d boundary edges, want 0 (watertight)", len(open))
	}
}
