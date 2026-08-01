// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
)

// b1SeamCases are the OCCT blend cases whose surviving periodic faces carry a closed seam edge
// (start-vertex == end-vertex, a full circle) that the fillet re-weld formerly oriented
// inconsistently — rejected as "inconsistent orientation at edge N". The closed-seam flip in
// edgeCatalog.use now assembles them into VALID manifold solids. See
// docs/superpowers/specs/2026-07-12-b1-closed-seam-orientation-design.md.
//
// Validity ONLY is asserted here: area parity for several of these is gated on separate,
// pre-existing defects the orientation fix EXPOSED (P2 torus full-surface tessellation; P1
// curved-fillet accuracy) — tracked separately, not this fix's job. Y1 is excluded: its seam is
// fillet-generated (not an imported survivor) and it carries a distinct Euler-characteristic
// defect (P3).
var b1SeamCases = map[string]bool{
	"R9": true, "S1": true, "S3": true, "S4": true, "S6": true, "S7": true,
	"S9": true, "T1": true, "T3": true, "T4": true, "T9": true, "X3": true,
}

// TestB1ClosedSeamValidSolid is the regression gate for the closed-seam orientation fix: every
// listed B1 case must fillet to a single valid solid (positive volume, healthy feature).
func TestB1ClosedSeamValidSolid(t *testing.T) {
	t.Parallel()
	fixtureDir := CorpusFixtureDir()
	seen := 0
	for _, r := range Corpus() {
		if r.Grid != "simple" || !b1SeamCases[r.Case] {
			continue
		}
		seen++
		body, err := importInput(filepath.Join(fixtureDir, r.InputStep))
		if err != nil {
			t.Fatalf("%s: import failed (expected a clean imported body): %v", r.Case, err)
		}
		sets, ok := scoreLocate(r, body)
		if !ok {
			t.Fatalf("%s: could not locate picked edges on the imported body", r.Case)
		}
		res, filletOK, reason := runFillet(body, sets)
		valid := filletOK && len(res) == 1 && res[0] != nil &&
			ops.BodyGeometryProperties(res[0], ops.PropertyQuality()).Volume > 0
		if !valid {
			t.Errorf("%s: fillet did not produce a valid solid (filletOK=%v n=%d): %s",
				r.Case, filletOK, len(res), reason)
		}
	}
	if seen != len(b1SeamCases) {
		t.Fatalf("expected %d B1 seam cases in the corpus, ran %d", len(b1SeamCases), seen)
	}
}
