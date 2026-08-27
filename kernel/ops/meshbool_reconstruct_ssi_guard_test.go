// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/meshbool"
	m "oblikovati.org/math"
)

// The disjoint-region cut layer (ADR-0054). A boolean whose exact soup is correct can still
// reconstruct to a VALID-but-wrong-volume analytic B-rep when a boundary run is an OBLIQUE
// conic surface-surface intersection (a cylinder cut by a tilted plane = ellipse): the ellipse
// is exact as a curve, but each incident face samples it independently, so the two copies do
// not weld and the closed solid encloses the wrong region (SlottedScrew's slanted-hex bore
// exit removed ~5 mm³ too little). weldableSSICurve confines SSI reuse to LINES and CIRCLES —
// exact and identical from both faces — so reconstruction DECLINES an oblique-conic run and the
// caller falls back to the exact faceted boolean. These lock that boundary: an oblique cut
// declines, while a perpendicular cut (circular exits) — even across a disjoint void — rebuilds.

// TestReconstructDeclinesObliqueConicCut: a cylinder cut through a box at an angle needs an
// ELLIPSE at each face exit. Reconstruction must decline (leaving the caller on the exact
// faceted path) rather than emit a valid-but-wrong analytic solid.
func TestReconstructDeclinesObliqueConicCut(t *testing.T) {
	box, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(1, 1, 1), "box")
	if err != nil {
		t.Fatalf("box: %v", err)
	}
	oblique, err := brep.SolidCylinder(m.P3(0.5, 0.5, -0.2), m.V3(0.35, 0, 1), 0.15, 1.6)
	if err != nil {
		t.Fatalf("cyl: %v", err)
	}
	if _, ok := reconstructBoolean(box, oblique, meshbool.Difference, DefaultQuality()); ok {
		t.Fatal("oblique-conic cut reconstructed; must decline (ellipse SSI does not weld — ADR-0054 Layer 4)")
	}
	// The faceted boolean still produces the correct result — declining loses nothing.
	res, err := Boolean(Cut, box, oblique)
	if err != nil {
		t.Fatalf("faceted fallback: %v", err)
	}
	if r := Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("faceted fallback is not a valid solid: %+v", r)
	}
}

// TestReconstructPerpendicularDisjointCutRebuilds: a block with a slot (a void), cut by a
// PERPENDICULAR cylinder crossing the slot. The bore is two disjoint segments, but every exit
// is an exact CIRCLE, so reconstruction rebuilds a valid analytic solid keeping BOTH bore walls
// — disjointness alone is not the blocker; the oblique conic is.
func TestReconstructPerpendicularDisjointCutRebuilds(t *testing.T) {
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
	// exact remaining volume: block 1.0 minus slot (0.4·1.2·0.2=0.096, clipped to block = 0.4·1·0.2=0.08)
	// minus the bore through the solid (height 1 - slot 0.2 = 0.8): pi·0.15²·0.8.
	want := 1.0 - 0.4*1.0*0.2 - stdmath.Pi*0.15*0.15*0.8
	if v := BodyGeometryProperties(recon, PropertyQuality()).Volume; stdmath.Abs(v-want) > 5e-3*want {
		t.Fatalf("disjoint cut volume = %.5f, want ~%.5f", v, want)
	}
}
