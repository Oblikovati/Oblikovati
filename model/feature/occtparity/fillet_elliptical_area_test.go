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

// leakGuardQuarantinedByHoleLoop are leakGuardGreenedCases members whose result ALSO carries the
// unrelated #2007 malformed hole-loop defect (ops.Validate().HolesContained == false; quarantine.go).
// The leak-guard min-area re-mesh repaired their AREA number (no more constrained-Delaunay fill
// inflation) — that fix stands, unregressed — but it never touched this independent topology defect.
// The hardened gate (isWatertightSolid, watertight.go) correctly holds them at SkipQuarantine rather
// than PASS. S9/T3 have since been GREENED by the single-boss setback tiling (#2007, footprint absorbed
// watertight), leaving only the dual-host U4 held; S3/S7/S9/T3/T6/X3 are now all genuinely watertight.
var leakGuardQuarantinedByHoleLoop = map[string]bool{"U4": true}

// TestLeakGuardedPlaneMeshGreensAreaCases pins the min-area conformance guard: each case must score
// PASS — or, for the 3 that also carry the independent #2007 hole-loop defect, SkipQuarantine (never
// FailArea/FailFaulty) — proving the hole-filled re-mesh still does not inflate the planar face area.
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
		want := Pass
		if leakGuardQuarantinedByHoleLoop[id] {
			want = SkipQuarantine
		}
		if got := ScoreCase(r, dir); got != want {
			t.Errorf("simple/%s scored %v, want %v (conformance plane re-mesh must not inflate a holed face)", id, got, want)
		}
	}
}
