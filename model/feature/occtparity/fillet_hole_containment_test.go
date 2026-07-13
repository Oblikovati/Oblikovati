// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
)

// TestFilletProtrudingHoleTripwire pins the malformed-face defect behind the elliptical-prism blend
// cases (②): the imported prism has a valid base plane (outer loop y∈[-13,12] contains the elliptical
// hole y∈[-10,10]), but filleting the base edge shrinks the outer loop's bottom to y=-7 while leaving the
// coplanar full-ellipse hole untouched, so the hole now protrudes 3 units past its own boundary. That
// malformed face is what the tessellator was struggling to mesh (the phantom "fill"/crack artifacts).
//
// ops.Validate reports this via HolesContained. It is a TRIPWIRE: the fillet trim that stops producing
// the protrusion is not yet written, so this test asserts the CURRENT (defective) state — imported body
// contained, filleted body NOT contained. When the fillet fix lands, flip the filleted assertion to true
// and fold HolesContained into Validate.Valid.
func TestFilletProtrudingHoleTripwire(t *testing.T) {
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
	rep := ops.Validate(res[0])
	if rep.HolesContained {
		t.Errorf("filleted T6 body: expected the base-plane hole to protrude past the shrunken outer loop " +
			"(the ② defect); HolesContained was true — the fillet trim may already be fixed, so fold " +
			"HolesContained into Valid and flip this assertion")
	}
}
