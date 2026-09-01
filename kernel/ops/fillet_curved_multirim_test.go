// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"os"
	"testing"

	"oblikovati.org/kernel/exchange"
	step "oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The multi-rim weld (fillet_curved_multirim.go, bfuseblend/B3): two pairwise-disjoint CLOSED
// Cylinder∧Plane rims in ONE op route through the proven single-rim band rebuild sequentially —
// the second rim's ReferenceKey resolves on the intermediate body because the rim rebuild carries
// every untouched edge's Lineage verbatim. DRAWEXE ground truth (b3_export receipts): 12 faces,
// per-face 4×10000 · 2×6151.55 · 2×8482.3 · 2×2827.43 · 2×1570.1, vprops 1.28484e6.

const multiRimFixture = "../../model/feature/occtparity/fixtures/bfuseblend/B3.step"

// importMultiRimFixture loads the bfuseblend/B3 through-cylinder solid (box + piercing cylinder fused,
// one closed concave rim per exit face).
func importMultiRimFixture(t *testing.T) *topo.Body {
	t.Helper()
	data, err := os.ReadFile(multiRimFixture)
	if err != nil {
		t.Skipf("fixture unavailable: %v", err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil || len(bodies) != 1 {
		t.Skipf("fixture import-divergence: %v (%d bodies)", err, len(bodies))
	}
	return bodies[0]
}

// multiRimFils resolves B3's two rim picks (y=0 and y=100 exit faces, r=5) into solved edgeFillets —
// the exact input assembleCurvedArmBody hands the dispatch.
func multiRimFils(t *testing.T, body *topo.Body) []edgeFillet {
	t.Helper()
	picks := []filletPick{
		{edge: edgeNearMidpoint(t, body, math.P3(20, 0, 50)), r0: 5, r1: 5},
		{edge: edgeNearMidpoint(t, body, math.P3(20, 100, 50)), r0: 5, r1: 5},
	}
	blends, miters, err := computeCorners(body, picks)
	if err != nil {
		t.Fatalf("computeCorners: %v", err)
	}
	fils, err := computeFillets(body, picks, blends, miters, FillConcaveOutward, nil)
	if err != nil {
		t.Fatalf("computeFillets: %v", err)
	}
	return fils
}

// TestIndependentClosedRimsWeldB3 drives the two-rim op end to end and certifies the weld: took=true,
// a valid closed manifold 12-face solid (the DRAWEXE face-count join), both cove bands present.
func TestIndependentClosedRimsWeldB3(t *testing.T) {
	t.Parallel()
	body := importMultiRimFixture(t)
	fils := multiRimFils(t, body)
	if len(fils) != 2 {
		t.Fatalf("resolved %d fils, want 2", len(fils))
	}
	out, reason, took := independentClosedRimsBody(body, fils)
	if !took || reason != "" {
		t.Fatalf("independentClosedRimsBody took=%v reason=%q, want the B3 weld", took, reason)
	}
	rep := Validate(out)
	if !rep.Valid || !rep.Closed || !rep.Manifold || !rep.HolesContained || !out.IsSolid() {
		t.Fatalf("B3 multi-rim weld not a watertight solid: %+v", rep.Issues)
	}
	if n := len(out.Faces()); n != 12 {
		t.Fatalf("B3 multi-rim weld has %d faces, want 12 (DRAWEXE nbshapes)", n)
	}
}

// TestIndependentClosedRimsGateDeclines proves the admission gate BOTH ways it must say no: a single
// rim is not this branch's business (loneRimPick owns it), and two fils sharing a host face (the same
// rim twice — a stand-in for two coves biting one face) must be rejected, never welded twice.
func TestIndependentClosedRimsGateDeclines(t *testing.T) {
	t.Parallel()
	body := importMultiRimFixture(t)
	fils := multiRimFils(t, body)
	if allDisjointClosedRims(fils[:1]) {
		t.Fatal("gate accepted a single rim — that pick belongs to loneRimPick, not the multi-rim weld")
	}
	if allDisjointClosedRims([]edgeFillet{fils[0], fils[0]}) {
		t.Fatal("gate accepted two rims sharing a host face — the composed one-face retrim is not built")
	}
	if _, _, took := independentClosedRimsBody(body, []edgeFillet{fils[0], fils[0]}); took {
		t.Fatal("independentClosedRimsBody took a shared-face rim pair — must leave it on the floor path")
	}
}
