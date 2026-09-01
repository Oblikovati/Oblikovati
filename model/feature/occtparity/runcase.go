// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// RunCase drives one OCCT blend case through our real fillet feature and asserts OCCT's
// reference area. TODO and import-divergent cases are skipped so the gate reflects fillet
// parity — not OCCT-incompleteness, and not a STEP round-trip gap. Constant-radius only
// here; variable-radius (buildevol) is added in Task 12.
//
// Example:
//
//	RunCase(t, occtparity.Corpus()[0], occtparity.CorpusFixtureDir())
func RunCase(t *testing.T, r Record, fixtureDir string) {
	t.Helper()
	if r.TODO != "" {
		t.Skipf("OCCT marks incomplete: %s", r.TODO)
	}
	if reason, held := quarantineReason(r); held {
		t.Skipf("%s/%s: quarantined (held out of the green count): %s", r.Grid, r.Case, reason)
	}
	if hasVariableRadius(r) {
		// The feature layer supports variable radius (FilletEdgeSet.StartRadius/EndRadius/
		// RadiusPoints), but OCCT's updatevol law is defined in EDGE-PARAMETER space, which
		// STEP import discards (edges reparameterize to [0,1]) — the same problem the locator
		// solved with arc-length. Porting the law faithfully needs arc-length reparameterization
		// of each law point on both sides; until that lands, skip rather than ship a wrong law.
		t.Skipf("%s/%s: variable-radius (buildevol) law mapping is a tracked follow-up", r.Grid, r.Case)
	}
	body, err := importInput(filepath.Join(fixtureDir, r.InputStep))
	if err != nil {
		t.Skipf("import-divergence (not a fillet defect): %v", err)
	}
	sets, ok := scoreLocate(r, body)
	if !ok {
		// A pick OCCT resolved that we cannot locate on the imported body is a topology
		// divergence, not a fillet defect — skip so the gate reflects fillet parity (the
		// scoreboard's SKIP(import) keeps these visible).
		t.Skipf("%s/%s: a picked edge could not be located on the imported body", r.Grid, r.Case)
	}
	res, filletOK, reason := runFillet(body, sets)
	if why, pending := pendingCapabilityReason(r); pending {
		assertStillPending(t, r, res, filletOK, why)
		return
	}
	assertCaseResult(t, r, res, filletOK, reason)
}

// assertStillPending handles a case whose capability is not yet built (pending.go). It SKIPS with the
// engine's own decline reason — but first fails loudly if the case now reaches full OCCT parity, so a
// stale entry can never mask progress the engine has actually made. That direction of the ratchet is
// the whole reason this is a pending list and not suppression.
func assertStillPending(t *testing.T, r Record, res []*topo.Body, filletOK bool, why string) {
	t.Helper()
	if caseReachesParity(r, res, filletOK) {
		t.Fatalf("%s/%s now builds OCCT-parity geometry — DELETE its pendingCapability entry and "+
			"decrement pendingCapabilityCount (recorded as: %s)", r.Grid, r.Case, why)
	}
	t.Skipf("%s/%s: pending capability — %s", r.Grid, r.Case, why)
}

// caseReachesParity reports whether a case is a genuine green: a valid watertight solid whose area is
// within OCCT's tolerance. It mirrors ScoreCase's green decision exactly, so the pending ratchet and
// the scoreboard can never disagree about what "green" means.
func caseReachesParity(r Record, res []*topo.Body, filletOK bool) bool {
	props, ok := caseProperties(res, filletOK)
	if classify(r, true, filletOK, isWatertightSolid(res, filletOK, props, ok)) != Pass {
		return false
	}
	return areaWithin(props.Area, r.areaTarget(), r.Deps)
}

// hasVariableRadius reports whether a case needs the buildevol variable-radius path (Task 12).
func hasVariableRadius(r Record) bool {
	if r.Verb == "buildevol" {
		return true
	}
	for _, p := range r.Picks {
		if len(p.Law) > 0 {
			return true
		}
	}
	return false
}

// runFillet drives the real feature path (base -> dress-up fillet -> recompute) and returns
// the result bodies, whether the fillet feature is healthy, and its reason when it is not.
func runFillet(body *topo.Body, sets []feature.FilletEdgeSet) ([]*topo.Body, bool, string) {
	fs := feature.NewPartFeatures(nil)
	feature.NewBaseFeatures(fs).AddBase(body)
	// Author through the builder the Fillet tool and the API use, so the parity corpus
	// exercises the shipped path rather than a convenience wrapper of its own (#2052).
	pf := feature.NewDressUpFeatures(fs).AddFilletSetsCorner(sets, types.FilletCornerMiter)
	fs.Recompute()
	return fs.Result(), pf.Health().OK(), pf.Health().Reason
}

// assertCaseResult validates the result solid, then asserts OCCT's area — mirroring OCCT's
// verdict semantics through classify.
func assertCaseResult(t *testing.T, r Record, res []*topo.Body, filletOK bool, reason string) {
	t.Helper()
	props, ok := caseProperties(res, filletOK)
	switch classify(r, true, filletOK, isWatertightSolid(res, filletOK, props, ok)) {
	case FailFaulty:
		t.Fatalf("%s/%s: result not a valid solid: %s", r.Grid, r.Case, reason)
	case Pass:
		assertCaseArea(t, r.Grid+"/"+r.Case, props.Area, r)
	}
}

// caseProperties tessellates the single result body ONCE (Property quality) and returns its
// volume+area+centroid, or ok=false when the fillet is unhealthy / not a single body. The watertight
// gate (isWatertightSolid — Volume>0 is one conjunct alongside ops.Validate's full Valid && Closed &&
// Manifold && HolesContained && IsSolid bar) and the area gate then read the SAME tessellation instead
// of running two full passes — a wasteful ~2× that dominated N7 scoring before the canal tessellation
// fix (n7-tessellation-diagnosis.md §5).
func caseProperties(res []*topo.Body, filletOK bool) (ops.GeometryProperties, bool) {
	if !filletOK || len(res) != 1 || res[0] == nil {
		return ops.GeometryProperties{}, false
	}
	return query.BodyGeometryProperties(res[0], ops.PropertyQuality()), true
}

// importTol scales the locator match tolerance to the body so scaled fixtures (tscale
// SCALE=1000) still resolve: 1e-4 of the bounding diagonal.
func importTol(b *topo.Body) float64 { return 1e-4 * boundingDiag(b) }

// boundingDiag is the diagonal of the body's vertex bounding box (its characteristic size).
func boundingDiag(b *topo.Body) float64 {
	verts := b.Vertices()
	if len(verts) == 0 {
		return 1
	}
	lo, hi := verts[0].Point(), verts[0].Point()
	for _, v := range verts[1:] {
		p := v.Point()
		lo = math.P3(stdmath.Min(lo.X, p.X), stdmath.Min(lo.Y, p.Y), stdmath.Min(lo.Z, p.Z))
		hi = math.P3(stdmath.Max(hi.X, p.X), stdmath.Max(hi.Y, p.Y), stdmath.Max(hi.Z, p.Z))
	}
	return hi.DistanceTo(lo)
}
