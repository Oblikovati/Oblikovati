// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// The per-case gate for OCCT tests/blend/simple/N4 — the first case welded by the GENERAL corner-weld layer
// (kernel/ops/cornerweld_*.go). N4 is a 100³ box with a full r=20 × h=50 cylinder standing on its vertical
// corner (270° of it protrudes), filleted r=5 on three edges meeting at one trihedral vertex: the concave
// boss-wall ∧ box-wall ruling, the CONVEX boss cap-rim arc, and the concave band where the cap plane meets
// the box wall.
//
// DRAWEXE 8.0.0 receipt — `restore data/CFI_e5678fil.rle s ; tscale s 0 0 0 10 ; explode s e ;
// blend result s 5 s_4 5 s_13 5 s_2 ; nbshapes ; sprops ; vprops ; checkshape`:
// valid SOLID, 1 shell, 14 faces / 14 wires / 22 vertices / 34 edges, area 64287.2, volume 1.04694e6.
//
// The decisive thing this asserts beyond the area scoreboard is the RIM CONTINUATION. Only the 90° piece of
// the 270° cap rim is picked, and its far vertex is a G1 seam on the boss wall — so a weld that terminates
// the arm there would leave the rest of the rim sharp and produce a DIFFERENT solid. OCCT's blend runs the
// fillet over the whole tangent chain, emitting the band as two torus faces split at the wall-face seam
// (76.3° over the first wall face, EXACTLY 180° over the second). Both are checked below, so a regression
// that drops the continuation, or that mis-picks which half of the 180° span the rail runs along, fails loud.

// TestN4CornerWeldLayerWatertight is the whole-body gate: watertight, fold-free, OCCT's face count and area.
func TestN4CornerWeldLayerWatertight(t *testing.T) {
	body := caseResultBody(t, "N4")
	assertWatertight(t, "N4", body, 14)
	assertWholeBodyFoldFree(t, "N4", body)
	assertWholeBodyArea(t, "N4", body, 64287.2)
	assertN4RimBandIsContinuedAndSplit(t, body)
}

// assertN4RimBandIsContinuedAndSplit checks the convex cap-rim fillet is present as TWO torus faces of the
// same tube (major R−r = 15, minor r = 5) whose spans are the oracle's — 76.3° over the first boss-wall face
// and exactly 180° over the second. The 180° face is the load-bearing one: it is the span the
// arm only reaches by running through the G1 seam, and its area (U·r·(R·π/2 + r) with U = π) also pins that
// the rail took the correct half of the contact circle — the wrong half meshes a visibly different band.
func assertN4RimBandIsContinuedAndSplit(t *testing.T, body *topo.Body) {
	t.Helper()
	var spans []float64
	for _, f := range body.Faces() {
		tor, ok := f.Geometry().(geom.Torus)
		if !ok || stdmath.Abs(tor.MajorRadius-15) > 1e-3 || stdmath.Abs(tor.MinorRadius-5) > 1e-3 {
			continue
		}
		spans = append(spans, faceMeshArea2(f)/(tor.MinorRadius*(tor.MajorRadius*stdmath.Pi/2+tor.MinorRadius)))
	}
	if len(spans) != 2 {
		t.Fatalf("N4 has %d R−r=15 cap-rim torus faces, want 2 (the rim continuation split at the boss-wall seam)", len(spans))
	}
	half := stdmath.Max(spans[0], spans[1])
	if stdmath.Abs(half-stdmath.Pi) > 0.01 {
		t.Fatalf("N4's continued rim band spans %.4f rad, want π (the whole second boss-wall face)", half)
	}
	// The oracle's own total: (190.242+448.65)/(r·(R·π/2+r)) = 4.4737 rad = 256.3° — the 270° rim less the
	// slice the corner patch consumes. Ours reads 4.4519 (−0.5%, the corner-patch redistribution of §3).
	if total := spans[0] + spans[1]; total < 4.42 || total > 4.53 {
		t.Fatalf("N4's rim band spans %.4f rad in total, want ≈4.4737 (the oracle's 270° rim less the corner patch)", total)
	}
}
