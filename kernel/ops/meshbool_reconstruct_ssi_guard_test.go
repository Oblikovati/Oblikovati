// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/ops/query"
	m "oblikovati.org/math"
)

// The corner-crossing decline boundary (ADR-0056 Layer 4). A CLEAN oblique cut — a cylinder
// bored through a box at an angle, one ellipse per pierced face — now reconstructs analytically
// (the ellipse SSI edge welds, and orientRunEdge picks the arc half the run traces; see
// meshbool_reconstruct_ssi_weld_test.go). What still declines is a cut whose ellipse crosses an
// operand CORNER: the exit ellipse splits across two adjacent faces, doubling a face loop into an
// Euler-inadmissible assembly (χ too large for one shell). reconstructBoolean self-validates and
// declines that, so the caller falls back to the exact faceted boolean — no wrong geometry. These
// lock the boundary: a corner-crossing oblique cut declines; a perpendicular cut (circular exits),
// even across a disjoint void, rebuilds.

// TestReconstructDeclinesCornerCrossingObliqueCut: an oblique bore whose exit ellipse crosses a box
// corner (the tool axis leaves through the top AND a side) assembles an Euler-inadmissible B-rep.
// Reconstruction must decline (leaving the caller on the exact faceted path) rather than emit it.
func TestReconstructDeclinesCornerCrossingObliqueCut(t *testing.T) {
	t.Parallel()
	box, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(1, 1, 1), "box")
	if err != nil {
		t.Fatalf("box: %v", err)
	}
	oblique, err := brep.SolidCylinder(m.P3(0.5, 0.5, -0.2), m.V3(0.35, 0, 1), 0.15, 1.6)
	if err != nil {
		t.Fatalf("cyl: %v", err)
	}
	if _, ok := reconstructBoolean(box, oblique, meshbool.Difference, DefaultQuality()); ok {
		t.Fatal("corner-crossing oblique cut reconstructed; must decline (Euler-inadmissible — ADR-0056 Layer 4)")
	}
	// Declining loses nothing: the exact faceted boolean still produces a valid solid that
	// removes real material (the tool axis pierces the box, so the result is strictly smaller
	// than the box). The CSG BSP engine's exact-volume correctness is covered separately, on
	// clean planar operands, by csg_test.go.
	res, err := Boolean(Cut, box, oblique)
	if err != nil {
		t.Fatalf("faceted fallback: %v", err)
	}
	if r := Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("faceted fallback is not a valid solid: %+v", r)
	}
	if v := query.BodyGeometryProperties(res, DefaultQuality()).Volume; v <= 0.9 || v >= 1.0 {
		t.Fatalf("faceted fallback volume = %.5f, want a box (1.0) with a bore removed (0.9..1.0)", v)
	}
}

// TestReconstructPerpendicularDisjointCutRebuilds: a block with a slot (a void), cut by a
// PERPENDICULAR cylinder crossing the slot. The bore is two disjoint segments, but every exit
// is an exact CIRCLE, so reconstruction rebuilds a valid analytic solid keeping BOTH bore walls
// — disjointness alone is not the blocker; the oblique conic is.
func TestReconstructPerpendicularDisjointCutRebuilds(t *testing.T) {
	t.Parallel()
	block, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(1, 1, 1), "block")
	if err != nil {
		t.Fatalf("block: %v", err)
	}
	slot, err := brep.SolidBlock(m.P3(0.3, -0.1, 0.4), m.P3(0.7, 1.1, 0.6), "slot")
	if err != nil {
		t.Fatalf("slot: %v", err)
	}
	target, err := Boolean(Cut, block, slot)
	if err != nil {
		t.Fatalf("slot cut: %v", err)
	}
	tool, err := brep.SolidCylinder(m.P3(0.5, 0.5, -0.1), m.V3(0, 0, 1), 0.15, 1.2)
	if err != nil {
		t.Fatalf("cyl: %v", err)
	}
	recon, ok := reconstructBoolean(target, tool, meshbool.Difference, DefaultQuality())
	if !ok {
		t.Fatal("perpendicular disjoint cut declined; circular exits must reconstruct")
	}
	if r := Validate(recon); !r.Valid || !r.Closed || !r.Manifold || !recon.IsSolid() {
		t.Fatalf("disjoint cut is not a valid closed manifold solid: %+v", r)
	}
	if n := cylinderFaceCount(recon); n != 2 {
		t.Fatalf("disjoint cut kept %d cylinder bore walls, want 2 (one per segment)", n)
	}
	// exact remaining Volume: block 1.0 minus slot (0.4·1.2·0.2=0.096, clipped to block = 0.4·1·0.2=0.08)
	// minus the bore through the solid (height 1 - slot 0.2 = 0.8): pi·0.15²·0.8.
	want := 1.0 - 0.4*1.0*0.2 - stdmath.Pi*0.15*0.15*0.8
	if v := query.BodyGeometryProperties(recon, PropertyQuality()).Volume; stdmath.Abs(v-want) > 5e-3*want {
		t.Fatalf("disjoint cut volume = %.5f, want ~%.5f", v, want)
	}
}
