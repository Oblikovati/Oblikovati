// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

// leakGuardGreenedCases are the corpus cases whose filleted body carried a planar face with a hole
// (the oblique elliptical cylinder's rim on T6/U4; other holed planes elsewhere) that the conformance
// repair re-meshed with a constrained Delaunay which FILLED the hole — a finely-discretized rim leaks
// past constrainedDelaunay's constraint recovery — inflating the face area and making the whole-body
// area both wrong and non-convergent (T6 oscillated +0.68%↔+4.2% with refinement). The conforming plane
// re-mesh now takes whichever of the Delaunay or the deterministic ear-clip triangulation covers LESS
// area (a defect only ever ADDS area, so the minimum is correct), which greens every case below.
var leakGuardGreenedCases = []string{"S3", "S7", "S9", "T3", "T6", "U4", "X3"}

// TestLeakGuardedPlaneMeshGreensAreaCases pins the min-area conformance guard: each case must now score
// PASS (a valid solid whose surface area is within OCCT's tolerance), proving the hole-filled re-mesh no
// longer inflates the planar face area.
func TestLeakGuardedPlaneMeshGreensAreaCases(t *testing.T) {
	byCase := map[string]Record{}
	for _, r := range Corpus() {
		if r.Grid == "simple" {
			byCase[r.Case] = r
		}
	}
	dir := CorpusFixtureDir()
	for _, id := range leakGuardGreenedCases {
		r, ok := byCase[id]
		if !ok {
			t.Fatalf("simple/%s not found in corpus", id)
		}
		if got := ScoreCase(r, dir); got != Pass {
			t.Errorf("simple/%s scored %v, want PASS (conformance plane re-mesh must not inflate a holed face)", id, got)
		}
	}
}
