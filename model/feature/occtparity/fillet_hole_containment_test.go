// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
)

// TestFilletSingleHostObstacleWatertight pins the single-host mid-span obstacle win on corpus T6: the
// imported prism has a valid base plane (outer loop y∈[-13,12] contains the elliptical hole y∈[-10,10]),
// but filleting the base edge shrinks the outer loop's bottom to y=-7 while leaving the coplanar
// full-ellipse hole untouched, so without the rebuild the hole protrudes 3 units past its own boundary —
// the malformed face the tessellator meshed into the phantom "fill"/crack artifacts.
//
// The mid-span obstacle rebuild (ADR-4, ops.obstacleFacesFor) notches the host plane around the obstacle,
// splits the obstacle wall's rim, and bridges the gap with two wings and a corner-blend patch, so the
// filleted T6 is now a genuinely WATERTIGHT, hole-contained solid. HolesContained is deliberately kept as
// a diagnostic tripwire (NOT folded into Valid) until the Phase-2 corner engine also handles the
// dual-host and torus-band configurations, so this asserts the watertightness directly (IsSolid +
// HolesContained) rather than through Valid. The dual-host/torus cases are honest-rejected by the
// detection gate and stay on the baseline path.
func TestFilletSingleHostObstacleWatertight(t *testing.T) {
	t.Parallel()
	fixtureDir := CorpusFixtureDir()
	var rec Record
	for _, r := range Corpus() {
		if r.Grid == "simple" && r.Case == "T6" {
			rec = r
		}
	}
	body, err := importInput(filepath.Join(fixtureDir, rec.InputStep))
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if rep := ops.Validate(body); !rep.HolesContained {
		t.Errorf("imported T6 body: hole should be contained in its outer loop, got protrusion: %v", rep.Issues)
	}

	sets, ok := scoreLocate(rec, body)
	if !ok {
		t.Fatalf("could not locate picked edges")
	}
	res, filletOK, reason := runFillet(body, sets)
	if !filletOK || len(res) != 1 || res[0] == nil {
		t.Fatalf("fillet did not produce a solid: %s", reason)
	}
	if !res[0].IsSolid() {
		t.Errorf("filleted T6 body must be a single closed solid (the obstacle rebuild welds the wing/patch/wall shell)")
	}
	if rep := ops.Validate(res[0]); !rep.HolesContained {
		t.Errorf("filleted T6 body: the mid-span obstacle patch must make the base plane hole-contained; "+
			"got protrusion: %v", rep.Issues)
	}
}
