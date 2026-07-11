// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
)

// Scoreboard tallies outcomes across records using the supplied per-record evaluator. It is
// pure counting, so it is testable with a fake evaluator and reused by the reporter.
func Scoreboard(records []Record, run func(Record) Outcome) map[Outcome]int {
	tally := map[Outcome]int{}
	for _, r := range records {
		tally[run(r)]++
	}
	return tally
}

// ScoreCase evaluates one case's outcome WITHOUT asserting — the non-gating counterpart to
// RunCase, for the scoreboard. It mirrors RunCase's decisions: TODO/variable-radius/import or
// locate failures are skips; an invalid result is FailFaulty; a valid result within OCCT's
// tolerance is Pass, otherwise FailArea.
func ScoreCase(r Record, fixtureDir string) Outcome {
	if r.TODO != "" {
		return SkipTODO
	}
	if hasVariableRadius(r) {
		return Incomplete // variable-radius (buildevol) pending Task 12
	}
	body, err := importInput(filepath.Join(fixtureDir, r.InputStep))
	if err != nil {
		return SkipImportDivergence
	}
	sets, ok := scoreLocate(r, body)
	if !ok {
		return SkipImportDivergence // an unlocatable pick is a fixture/harness gap, not a fillet defect
	}
	res, filletOK, _ := runFillet(body, sets)
	valid := filletOK && len(res) == 1 && res[0] != nil &&
		ops.BodyGeometryProperties(res[0], ops.PropertyQuality()).Volume > 0
	if o := classify(r, true, filletOK, valid); o != Pass {
		return o
	}
	got := ops.BodyGeometryProperties(res[0], ops.PropertyQuality()).Area
	if areaWithin(got, r.ExpectedArea, r.Deps) {
		return Pass
	}
	return FailArea
}

// scoreLocate resolves every pick to an edge fillet set, returning ok=false if any pick cannot
// be located (the non-fatal counterpart to locatePicks).
func scoreLocate(r Record, body *topo.Body) ([]feature.FilletEdgeSet, bool) {
	tol := importTol(body)
	sets := make([]feature.FilletEdgeSet, 0, len(r.Picks))
	for _, p := range r.Picks {
		e, err := locateEdge(body, p.Locator, tol)
		if err != nil {
			return nil, false
		}
		radius := p.Radius
		sets = append(sets, feature.FilletEdgeSet{
			EdgeKeys: [][]byte{e.ReferenceKey()},
			Radius:   func() float64 { return radius },
		})
	}
	return sets, true
}
