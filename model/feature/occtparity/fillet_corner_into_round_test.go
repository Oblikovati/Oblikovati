// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
)

// cornerIntoRoundCases are corner-into-round cases (a planar-hosted cylinder fillet whose endpoint
// runs into a PRE-EXISTING curved round) — one per round surface type. The #1797 guard used to reject
// every such junction up front; build-then-certify (fillet.go) now BUILDS the corner and lets Validate
// certify it, so the asymmetric ones (the round's radius differs from the arm's, so it trims cleanly)
// become valid solids. Only the symmetric equal-radius corner still honest-rejects (pinned by the
// kernel/feature #1797 tests).
var cornerIntoRoundCases = map[string]string{
	"B1": "cylinder", "B9": "cone", "D3": "sphere", "F2": "torus",
}

// TestCornerIntoRoundBuildsValidSolid gates the build-then-certify relaxation: each asymmetric
// corner-into-round case must now build a VALID, CLOSED, MANIFOLD solid enclosing a POSITIVE volume,
// where the blanket guard previously honest-rejected it. Area parity against OCCT is the separate
// frozen P1 curved-fillet gap (asserted by the scoreboard) — here we assert only that the corner
// closes into a sound solid.
func TestCornerIntoRoundBuildsValidSolid(t *testing.T) {
	t.Parallel()
	fixtureDir := CorpusFixtureDir()
	seen := 0
	for _, r := range Corpus() {
		round, want := cornerIntoRoundCases[r.Case]
		if r.Grid != "simple" || !want {
			continue
		}
		seen++
		body, err := importInput(filepath.Join(fixtureDir, r.InputStep))
		if err != nil {
			t.Fatalf("%s: import failed: %v", r.Case, err)
		}
		sets, ok := scoreLocate(r, body)
		if !ok {
			t.Fatalf("%s: could not locate picked edges", r.Case)
		}
		res, filletOK, reason := runFillet(body, sets)
		if !filletOK || len(res) != 1 || res[0] == nil {
			t.Fatalf("%s (%s round): corner-into-round did not close into a solid: %s", r.Case, round, reason)
		}
		if rep := ops.Validate(res[0]); !rep.Valid || !rep.Closed || !rep.Manifold || !res[0].IsSolid() {
			t.Errorf("%s (%s round): not a valid closed manifold solid: %+v", r.Case, round, rep)
		}
		if v := query.BodyGeometryProperties(res[0], ops.PropertyQuality()).Volume; v <= 0 {
			t.Errorf("%s (%s round): enclosed volume %.4f is not positive", r.Case, round, v)
		}
	}
	if seen != len(cornerIntoRoundCases) {
		t.Fatalf("expected %d corner-into-round cases, ran %d", len(cornerIntoRoundCases), seen)
	}
}
