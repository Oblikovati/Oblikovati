// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestU4DualHostWeldClearsWatertightBar pins the ROOT CAUSE the #2007 audit named, now FIXED: U4's
// filleted result used to have props.Volume>0 and a coincidentally-in-tolerance area but
// HolesContained=false (a hole loop protruding past its own outer loop — a malformed planar face) — the
// last case for which the OLD gate (ok && props.Volume>0) diverged from the hardened isWatertightSolid.
// The U4-5 dual-host multi-rail weld (kernel/ops/fillet_obstacle_dual*.go) rebuilds the corner watertight,
// so U4 now reports HolesContained=true and isWatertightSolid ACCEPTS it — the exemplar flipped from the
// surviving guard to a genuine solid, and this is its regression lock (U4 must STAY watertight).
func TestU4DualHostWeldClearsWatertightBar(t *testing.T) {
	res, filletOK, props, ok := rawFilletResult(t, "U4")
	if !ok || props.Volume <= 0 {
		t.Fatalf("U4: expected the legacy predicate (ok && Volume>0) to hold, got ok=%v volume=%v", ok, props.Volume)
	}
	if rep := ops.Validate(res[0]); !rep.HolesContained {
		t.Fatalf("U4: expected ops.Validate().HolesContained=true after the U4-5 dual-host weld; got false — the fix regressed")
	}
	if !isWatertightSolid(res, filletOK, props, ok) {
		t.Fatalf("U4: isWatertightSolid must accept the welded dual-host solid")
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

// TestQuarantinedFalseGreensReportSkipQuarantine pins the quarantine.go hold: a held case must report
// SkipQuarantine through the real ScoreCase path — never Pass, and never surfaced as a new FailFaulty red
// either (held out of the green count, not turned into a new failure). S6/S9/T3 were GREENED by the
// single-boss setback tiling; U3 by the dipArcOrder obstacle-path fix; U4 by the U4-5 dual-host multi-rail
// weld — leaving H6 (concave open-torus fillet inverted, ROOT 2) as the surviving held case this guards.
func TestQuarantinedFalseGreensReportSkipQuarantine(t *testing.T) {
	dir := CorpusFixtureDir()
	for _, c := range []string{"H6"} {
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
