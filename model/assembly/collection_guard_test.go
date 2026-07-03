// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"testing"

	"oblikovati.org/math"
)

// TestContractCollectionsGuardProbingIndices is the table-driven guard the hand-typed
// bounds checks never had (#1655): every contract-facing collection in this package must
// return a true nil interface — never panic — for Item(-1) and Item(Count()), because
// add-ins probe collections with raw indices.
func TestContractCollectionsGuardProbingIndices(t *testing.T) {
	reps, occs, cs, js := newReps()
	place(occs, "a:1", math.Identity4())
	reps.CaptureDesignView("dv", nil)
	reps.CapturePositional("pos")
	reps.CaptureLOD("lod")
	reps.CreateModelState("ms", "", "", "")
	contacts := NewContactSolver()
	contacts.Create("pair")

	probes := []struct {
		name  string
		count int
		item  func(i int) any
	}{
		{"DesignViews", reps.DesignViews().Count(), func(i int) any { return reps.DesignViews().Item(i) }},
		{"Positionals", reps.Positionals().Count(), func(i int) any { return reps.Positionals().Item(i) }},
		{"LevelsOfDetail", reps.LevelsOfDetail().Count(), func(i int) any { return reps.LevelsOfDetail().Item(i) }},
		{"ModelStates", reps.ModelStates().Count(), func(i int) any { return reps.ModelStates().Item(i) }},
		{"ConstraintSet", cs.Count(), func(i int) any { return cs.Item(i) }},
		{"JointSet", js.Count(), func(i int) any { return js.Item(i) }},
		{"DSJointSet", NewDSJointSet().Count(), func(i int) any { return NewDSJointSet().Item(i) }},
		{"ContactSolver", contacts.Count(), func(i int) any { return contacts.Item(i) }},
		{"InterferenceResults", 1, func(i int) any {
			return InterferenceResults{Results: []InterferenceResult{{}}}.Item(i)
		}},
	}
	for _, p := range probes {
		if got := p.item(-1); got != nil {
			t.Errorf("%s.Item(-1) = %#v, want nil", p.name, got)
		}
		if got := p.item(p.count); got != nil {
			t.Errorf("%s.Item(Count()=%d) = %#v, want nil", p.name, p.count, got)
		}
		if p.count > 0 && p.item(0) == nil {
			t.Errorf("%s.Item(0) = nil with %d elements, want the element", p.name, p.count)
		}
	}
}
