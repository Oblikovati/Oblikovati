// SPDX-License-Identifier: GPL-2.0-only

package ops

import "testing"

// Task C0 (canal-c0-brief.md): this is the SEAM test for the canal tier — it proves the
// RailLoop.Canal payload + stub wiring is corpus-neutral (nothing behavioural changes) before any
// canal math lands (C1-C3). The provider itself always declines (Build returns ok=false), so every
// assertion here is about classification (Fits) and fall-through (resolveBlend still reaching
// coons4), never about a patch. Supersedes corner_provider_plate_test.go (deleted with the plate
// stub, ADR-C3): canalProvider takes the plate's old tier slot.

// TestCanalProviderFits pins the Fits truth table: true iff the loop carries a non-nil Canal payload
// AND is valence-4. A default (Canal==nil) valence-4 loop and a Canal-populated valence-3 loop must
// both decline — the payload pointer is the ONLY classification signal (ADR-C2; canal-corner-seam-
// architecture.md §2), not loop shape alone.
func TestCanalProviderFits(t *testing.T) {
	p := canalProvider{}

	unmarkedV4 := quarterCylLoop(t, 8) // Canal defaults to nil
	if p.Fits(unmarkedV4) {
		t.Fatal("canalProvider.Fits must be false for an unmarked (Canal==nil) valence-4 loop")
	}

	markedV3 := sphereTriLoop(t, 4)
	markedV3.Canal = &CanalCorner{Radius: 4}
	if p.Fits(markedV3) {
		t.Fatal("canalProvider.Fits must be false for a Canal-populated valence-3 loop (needs valence 4 too)")
	}

	markedV4 := quarterCylLoop(t, 8)
	markedV4.Canal = &CanalCorner{Radius: 5}
	if !p.Fits(markedV4) {
		t.Fatal("canalProvider.Fits must be true for a Canal-populated valence-4 loop")
	}
}

// TestCanalProviderBuildDeclines pins the C0 stub contract: Build ALWAYS declines (ok=false), even
// on a loop that Fits. The do-no-harm floor (ADR-0051) requires this — no canal math exists yet
// (C1-C3), so any acceptance here would fabricate a patch.
func TestCanalProviderBuildDeclines(t *testing.T) {
	loop := quarterCylLoop(t, 8)
	loop.Canal = &CanalCorner{Radius: 5}
	p := canalProvider{}
	if !p.Fits(loop) {
		t.Fatal("fixture must Fit for this test to be meaningful")
	}
	_, _, ok := p.Build(loop, blendScale())
	if ok {
		t.Fatal("stub canalProvider.Build must always decline (ok=false) at C0")
	}
}

// The tier-order assertion (analyticSphere, canal, coons4, tri3) lives in
// TestResolveBlendTiersOrder (corner_resolve_test.go) — not duplicated here.

// TestMarkedN7LoopStillResolvesCoons4 is the corpus-neutrality gate at the resolveBlend seam: the
// real N7 fixture (n7CornerFill, corner_extract_tangent_test.go) extracts a loop with Canal
// populated, but since canalProvider.Build always declines at C0, resolveBlend must fall through to
// coons4 exactly as it did before this task (and as it did under the plate stub before it) — corpus
// stays byte-identical at 55.
func TestMarkedN7LoopStillResolvesCoons4(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok || loop.Valence() != 4 {
		t.Fatalf("extractCurvedCorner: want a closed valence-4 N7 loop; ok=%v valence=%d", ok, loop.Valence())
	}
	if loop.Canal == nil {
		t.Fatal("N7 loop must carry a populated Canal payload")
	}
	patch, ok := resolveBlend(loop, res)
	if !ok || patch.Kind != BlendKindCoons4 {
		t.Fatalf("Canal-marked N7 loop must still resolve to coons4 (canal stub declines); ok=%v kind=%q", ok, patch.Kind)
	}
}
