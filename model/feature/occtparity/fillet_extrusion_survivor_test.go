// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
)

// extrusionFilletFixtures are the oblique-elliptical-prism corpus cases (imported via the STEP
// SURFACE_OF_LINEAR_EXTRUSION fix, whose top/bottom rims are closed geom.EllipseFull edges). Filleting
// them formerly produced an INVALID result: the fillet rebuild's host-face copy (survivorCurve) only
// preserved geom.Arc3d survivor edges, so each closed-ellipse rim lost its curve and collapsed to a
// zero-length stub — degenerating the elliptical wall + its cap planes and tripping ops.Validate
// ("inconsistent orientation at edge N" on T6/T7, an open shell on U4). survivorCurve now carries the
// full-ellipse rim so the host faces survive the rebuild intact.
var extrusionFilletFixtures = []string{"F6", "T6", "T7", "U3", "U4"}

// TestExtrusionFixturesFilletToValidSolid pins the survivorCurve fix at the level that matters: each
// fixture's located fillet must build a VALID, CLOSED, MANIFOLD solid enclosing a positive volume —
// proving the closed elliptical rims survived the host-face copy (no collapsed zero-length stubs).
func TestExtrusionFixturesFilletToValidSolid(t *testing.T) {
	t.Parallel()
	fixtureDir := CorpusFixtureDir()
	byCase := map[string]Record{}
	for _, r := range Corpus() {
		if r.Grid == "simple" {
			byCase[r.Case] = r
		}
	}
	for _, id := range extrusionFilletFixtures {
		r, ok := byCase[id]
		if !ok {
			t.Fatalf("%s: no corpus record", id)
		}
		body, err := importInput(filepath.Join(fixtureDir, r.InputStep))
		if err != nil {
			t.Fatalf("%s: import failed: %v", id, err)
		}
		sets, ok := scoreLocate(r, body)
		if !ok {
			t.Fatalf("%s: could not locate picked edges", id)
		}
		res, filletOK, reason := runFillet(body, sets)
		if !filletOK || len(res) != 1 || res[0] == nil {
			t.Fatalf("%s: fillet did not produce a solid: %s", id, reason)
		}
		if rep := ops.Validate(res[0]); !rep.Valid || !rep.Closed || !rep.Manifold || !res[0].IsSolid() {
			t.Errorf("%s: not a valid closed manifold solid: %+v", id, rep)
		}
		if v := query.BodyGeometryProperties(res[0], ops.PropertyQuality()).Volume; v <= 0 {
			t.Errorf("%s: enclosed volume %.4f is not positive", id, v)
		}
	}
}
