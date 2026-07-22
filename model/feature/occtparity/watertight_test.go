// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestIsWatertightSolidRejectsHoleProtrusion pins the ROOT CAUSE the hardened gate closes (#2007): S6's
// filleted result has props.Volume>0 and a coincidentally-in-tolerance area, so the OLD gate (ok &&
// props.Volume>0) scored it PASS — but ops.Validate reports HolesContained=false (a hole loop protrudes
// past its own outer loop, a malformed planar face that poisons the tessellator). isWatertightSolid must
// reject it even though the legacy predicate would still accept it.
func TestIsWatertightSolidRejectsHoleProtrusion(t *testing.T) {
	res, filletOK, props, ok := rawFilletResult(t, "S6")
	if !ok || props.Volume <= 0 {
		t.Fatalf("S6: expected the legacy predicate (ok && Volume>0) to hold, got ok=%v volume=%v — audit premise changed", ok, props.Volume)
	}
	if rep := ops.Validate(res[0]); rep.HolesContained {
		t.Fatalf("S6: expected ops.Validate().HolesContained=false (the #2007 malformed hole-loop defect); got true — audit premise changed")
	}
	if isWatertightSolid(res, filletOK, props, ok) {
		t.Fatalf("S6: isWatertightSolid must reject a HolesContained=false result")
	}
}

// TestIsWatertightSolidAcceptsGenuineSolid confirms the hardened gate does not regress a genuinely
// watertight solid: B3 (planar corner, do-no-harm fingerprint pin) and I9 (rim-fillet Arc3d path) both
// have HolesContained=true and must still pass.
func TestIsWatertightSolidAcceptsGenuineSolid(t *testing.T) {
	for _, name := range []string{"B3", "I9"} {
		res, filletOK, props, ok := rawFilletResult(t, name)
		if !isWatertightSolid(res, filletOK, props, ok) {
			t.Errorf("%s: isWatertightSolid rejected a genuine watertight solid", name)
		}
	}
}

// TestQuarantinedFalseGreensReportSkipQuarantine pins the quarantine.go hold: S6/S9/T3/U3/U4 (#2007,
// malformed hole-loop, area coincidentally within Deps) must report SkipQuarantine through the real
// ScoreCase path — never Pass, and never surfaced as a new FailFaulty red either (the H6 precedent: held
// out of the green count, not turned into a new failure).
func TestQuarantinedFalseGreensReportSkipQuarantine(t *testing.T) {
	dir := CorpusFixtureDir()
	for _, c := range []string{"S6", "S9", "T3", "U3", "U4"} {
		if got := ScoreCase(findCorpusRecord(t, "simple", c), dir); got != SkipQuarantine {
			t.Errorf("simple/%s: ScoreCase = %v, want SkipQuarantine", c, got)
		}
	}
}

// rawFilletResult drives the real fillet feature for one simple-grid case OUTSIDE the quarantine
// short-circuit (unlike ScoreCase/RunCase, which skip a quarantined case before it ever runs), so
// isWatertightSolid can be probed directly against a quarantined case's raw result.
func rawFilletResult(t *testing.T, name string) ([]*topo.Body, bool, ops.GeometryProperties, bool) {
	t.Helper()
	r := findCorpusRecord(t, "simple", name)
	body, err := importInput(filepath.Join(CorpusFixtureDir(), r.InputStep))
	if err != nil {
		t.Fatalf("%s: import failed: %v", name, err)
	}
	sets, ok := scoreLocate(r, body)
	if !ok {
		t.Fatalf("%s: could not locate picked edges", name)
	}
	res, filletOK, _ := runFillet(body, sets)
	props, propsOK := caseProperties(res, filletOK)
	return res, filletOK, props, propsOK
}
