// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
)

// b2WindingCases are the box-corner corpus cases whose curved fillet re-weld formerly produced an
// INVALID solid — the blend cylinder/sphere patches were wound the same way as their neighbours, so
// ops.Validate rejected them with "inconsistent orientation at edge N". orientFilletShell (B2) now
// normalizes the shell winding via a shared-edge 2-colouring before the topology is built.
var b2WindingCases = map[string]bool{
	"K6": true, "K9": true, "L1": true, "L3": true, "L4": true, "L6": true, "L7": true,
}

// TestB2WindingProducesOutwardSolid gates the B2 fix at the level that actually matters — the mesh.
// Each case must build a VALID, CLOSED, MANIFOLD solid whose tessellation encloses a POSITIVE
// volume: positive signed volume proves the triangle normals resolve outward, i.e. the winding
// normalization did not leave an inside-out patch. (Area parity against OCCT is the separate, frozen
// P1 curved-fillet gap and is asserted by the scoreboard, not here.)
func TestB2WindingProducesOutwardSolid(t *testing.T) {
	t.Parallel()
	fixtureDir := CorpusFixtureDir()
	seen := 0
	for _, r := range Corpus() {
		if r.Grid != "simple" || !b2WindingCases[r.Case] {
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
			t.Fatalf("%s: fillet did not produce a solid: %s", r.Case, reason)
		}
		if rep := ops.Validate(res[0]); !rep.Valid || !rep.Closed || !rep.Manifold || !res[0].IsSolid() {
			t.Errorf("%s: not a valid closed manifold solid: %+v", r.Case, rep)
		}
		if v := query.BodyGeometryProperties(res[0], ops.PropertyQuality()).Volume; v <= 0 {
			t.Errorf("%s: enclosed volume %.4f is not positive — winding left an inside-out patch", r.Case, v)
		}
	}
	if seen != len(b2WindingCases) {
		t.Fatalf("expected %d B2 winding cases, ran %d", len(b2WindingCases), seen)
	}
}
