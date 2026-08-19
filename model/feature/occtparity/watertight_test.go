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
	t.Parallel()
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
	t.Parallel()
	for _, name := range []string{"B3", "I9"} {
		res, filletOK, props, ok := rawFilletResult(t, name)
		if !isWatertightSolid(res, filletOK, props, ok) {
			t.Errorf("%s: isWatertightSolid rejected a genuine watertight solid", name)
		}
	}
}

// TestIsWatertightSolidRejectsInterpenetration is the #2079 regression: R8 and W9 (the deep concave
// boss-base rims whose R+r cove spills onto the side walls) build a body that is topologically valid —
// Valid && Closed && Manifold && HolesContained && IsSolid — yet drives the plate face and the side wall
// straight THROUGH each other (interpenetration 3-15 model units, far past any tessellation error). The
// old topology-only gate called that watertight; the hardened isWatertightSolid rejects it via the
// self-intersection scan. The three parts below pin each half: the blind spot, the defect, the fix.
func TestIsWatertightSolidRejectsInterpenetration(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"R8", "W9"} {
		res, filletOK, props, ok := rawFilletResult(t, name)
		if rep := ops.Validate(res[0]); !(rep.Valid && rep.Closed && rep.Manifold && rep.HolesContained && res[0].IsSolid()) {
			t.Fatalf("%s: expected topology-valid (the #2079 blind spot the old gate trusted); got %+v", name, rep)
		}
		if len(ops.SelfIntersections(res[0], ops.PropertyQuality())) == 0 {
			t.Fatalf("%s: expected a self-intersection (the #2079 defect); found none", name)
		}
		if isWatertightSolid(res, filletOK, props, ok) {
			t.Errorf("%s: isWatertightSolid must REJECT a self-intersecting body, not report it watertight (#2079)", name)
		}
	}
}

// TestTheCorpusHoldsNoCase pins the quarantine list EMPTY — the corpus's last blind spot, closed.
//
// WHY THIS IS THE ASSERTION NOW. A quarantined case is SKIPPED by RunCase before it ever builds, so it is
// invisible not just to the area gate but to every corpus-wide invariant this harness owns — watertightness,
// retracing loops, self-crossing loops, tangent-chain debt. Holding one buys honesty about a known defect at
// the price of blinding everything else about that case (the D8 precedent: an honest FAIL was preferred to a
// hold for exactly this reason). simple/H6 was the last hold; the arc band's rolling-ball seat + run-out
// termination retired it, and H6 now scores on its own merit — Pass at −0.00031% against DRAWEXE, and
// carrying 0 free edges where it used to hide 642.
//
// The MECHANISM is guarded alongside, so re-holding a case stays a one-line change that works: a synthetic
// key must still report held with its reason, and a key that is not on the list must not.
func TestTheCorpusHoldsNoCase(t *testing.T) {
	t.Parallel()
	for k, reason := range quarantined {
		t.Errorf("%s/%s is still quarantined (%q) — a held case is SKIPPED, so every other corpus invariant "+
			"is blind to it; score it honestly instead", k.grid, k.name, reason)
	}
	if _, held := quarantineReason(Record{Grid: "simple", Case: "H6"}); held {
		t.Errorf("simple/H6 still reads as quarantined")
	}
}

// TestQuarantineMechanismStillHolds keeps quarantine.go's lookup honest while its list is empty, so the
// next case that needs holding is one map entry away rather than a rediscovery.
func TestQuarantineMechanismStillHolds(t *testing.T) {
	t.Parallel()
	key := quarantineKey{grid: "zz", name: "ZZ"}
	quarantined[key] = "synthetic hold"
	defer delete(quarantined, key)
	if reason, held := quarantineReason(Record{Grid: "zz", Case: "ZZ"}); !held || reason != "synthetic hold" {
		t.Errorf("quarantineReason on a held key = (%q, %v), want (\"synthetic hold\", true)", reason, held)
	}
	if _, held := quarantineReason(Record{Grid: "zz", Case: "OTHER"}); held {
		t.Errorf("quarantineReason reported an unlisted case as held")
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
