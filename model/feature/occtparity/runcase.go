// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
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
	if hasVariableRadius(r) {
		t.Skipf("%s/%s: variable-radius (buildevol) pending Task 12", r.Grid, r.Case)
	}
	body, err := importInput(filepath.Join(fixtureDir, r.InputStep))
	if err != nil {
		t.Skipf("import-divergence (not a fillet defect): %v", err)
	}
	sets := locatePicks(t, r, body)
	res, filletOK, reason := runFillet(body, sets)
	assertCaseResult(t, r, res, filletOK, reason)
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

// locatePicks resolves each oracle pick to a body edge by geometry and pairs it with its
// radius. A pick that cannot be located is a hard failure: OCCT resolved it, so must we.
func locatePicks(t *testing.T, r Record, body *topo.Body) []feature.FilletEdgeSet {
	t.Helper()
	tol := importTol(body)
	sets := make([]feature.FilletEdgeSet, 0, len(r.Picks))
	for _, p := range r.Picks {
		e, err := locateEdge(body, p.Locator, tol)
		if err != nil {
			t.Fatalf("%s/%s: %v", r.Grid, r.Case, err)
		}
		radius := p.Radius
		sets = append(sets, feature.FilletEdgeSet{
			EdgeKeys: [][]byte{e.ReferenceKey()},
			Radius:   func() float64 { return radius },
		})
	}
	return sets
}

// runFillet drives the real feature path (base -> dress-up fillet -> recompute) and returns
// the result bodies, whether the fillet feature is healthy, and its reason when it is not.
func runFillet(body *topo.Body, sets []feature.FilletEdgeSet) ([]*topo.Body, bool, string) {
	fs := feature.NewPartFeatures(nil)
	feature.NewBaseFeatures(fs).AddBase(body)
	pf := feature.NewDressUpFeatures(fs).AddFilletSets(sets)
	fs.Recompute()
	return fs.Result(), pf.Health().OK(), pf.Health().Reason
}

// assertCaseResult validates the result solid, then asserts OCCT's area — mirroring OCCT's
// verdict semantics through classify.
func assertCaseResult(t *testing.T, r Record, res []*topo.Body, filletOK bool, reason string) {
	t.Helper()
	valid := filletOK && len(res) == 1 && res[0] != nil &&
		ops.BodyGeometryProperties(res[0], ops.PropertyQuality()).Volume > 0
	switch classify(r, true, filletOK, valid) {
	case FailFaulty:
		t.Fatalf("%s/%s: result not a valid solid: %s", r.Grid, r.Case, reason)
	case Pass:
		got := ops.BodyGeometryProperties(res[0], ops.PropertyQuality()).Area
		assertArea(t, r.Grid+"/"+r.Case, got, r.ExpectedArea, r.Deps)
	}
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
