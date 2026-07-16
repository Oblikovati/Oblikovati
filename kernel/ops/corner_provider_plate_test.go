// SPDX-License-Identifier: GPL-2.0-only

package ops

import "testing"

// Task P0 (plate-p0-brief.md): this is the SEAM test for the plate tier — it proves the marker/stub
// wiring is corpus-neutral (nothing behavioural changes) before any plate math lands (P1-P4). The
// provider itself always declines (Build returns ok=false), so every assertion here is about
// classification (Fits) and fall-through (resolveBlend still reaching coons4), never about a patch.

// TestPlateProviderFits pins the Fits truth table: true iff the loop is BOTH marked
// RailSignatureTangentPlate AND valence-4. A default (unmarked) valence-4 loop and a marked
// valence-3 loop must both decline — Signature is the ONLY classification signal (ADR-0051; the
// extractor→provider channel), not loop shape alone.
func TestPlateProviderFits(t *testing.T) {
	p := plateProvider{}

	unmarkedV4 := quarterCylLoop(t, 8) // Signature defaults to RailSignatureGeneral (0)
	if p.Fits(unmarkedV4) {
		t.Fatal("plateProvider.Fits must be false for an unmarked (Signature==General) valence-4 loop")
	}

	markedV3 := sphereTriLoop(t, 4)
	markedV3.Signature = RailSignatureTangentPlate
	if p.Fits(markedV3) {
		t.Fatal("plateProvider.Fits must be false for a marked valence-3 loop (needs valence 4 too)")
	}

	markedV4 := quarterCylLoop(t, 8)
	markedV4.Signature = RailSignatureTangentPlate
	if !p.Fits(markedV4) {
		t.Fatal("plateProvider.Fits must be true for a Signature==TangentPlate valence-4 loop")
	}
}

// TestPlateProviderBuildDeclines pins the P0 stub contract: Build ALWAYS declines (ok=false), even
// on a loop that Fits. The do-no-harm floor (ADR-0051) requires this — no plate math exists yet
// (P1-P4), so any acceptance here would fabricate a patch.
func TestPlateProviderBuildDeclines(t *testing.T) {
	loop := quarterCylLoop(t, 8)
	loop.Signature = RailSignatureTangentPlate
	p := plateProvider{}
	if !p.Fits(loop) {
		t.Fatal("fixture must Fit for this test to be meaningful")
	}
	_, _, ok := p.Build(loop, blendScale())
	if ok {
		t.Fatal("stub plateProvider.Build must always decline (ok=false) at P0")
	}
}

// The tier-order assertion (analyticSphere, plate, coons4, tri3) lives in
// TestResolveBlendTiersOrder (corner_resolve_test.go) — not duplicated here.

// TestMarkedN7LoopStillResolvesCoons4 is the corpus-neutrality gate at the resolveBlend seam: the
// real N7 fixture (n7CornerFill, corner_extract_tangent_test.go) extracts a loop stamped
// RailSignatureTangentPlate, but since plateProvider.Build always declines at P0, resolveBlend must
// fall through to coons4 exactly as it did before this task — corpus stays byte-identical at 55.
func TestMarkedN7LoopStillResolvesCoons4(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok || loop.Valence() != 4 {
		t.Fatalf("extractCurvedCorner: want a closed valence-4 N7 loop; ok=%v valence=%d", ok, loop.Valence())
	}
	if loop.Signature != RailSignatureTangentPlate {
		t.Fatalf("N7 loop must be stamped RailSignatureTangentPlate, got %v", loop.Signature)
	}
	patch, ok := resolveBlend(loop, res)
	if !ok || patch.Kind != BlendKindCoons4 {
		t.Fatalf("marked N7 loop must still resolve to coons4 (plate stub declines); ok=%v kind=%q", ok, patch.Kind)
	}
}
