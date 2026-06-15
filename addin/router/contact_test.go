// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/api/wire"
)

// TestInterferenceAnalysisOverWire checks the M12-F05 interference analysis: two unit boxes
// overlapping by 0.5 cm in X report one interfering pair with the right overlap volume; boxes
// that do not overlap report none (#362/#368).
func TestInterferenceAnalysisOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 0.5) // unit boxes, overlap x in [0.5,1]

	var res wire.InterferenceResultsResult
	call(t, r, s, "interference.analyze", `{}`, &res)
	if len(res.Results) != 1 {
		t.Fatalf("interference results = %d, want 1 overlapping pair", len(res.Results))
	}
	got := res.Results[0]
	if got.OccurrenceA != occs[0].ID() && got.OccurrenceB != occs[0].ID() {
		t.Errorf("pair = %d/%d, want box:1 (%d) involved", got.OccurrenceA, got.OccurrenceB, occs[0].ID())
	}
	if math.Abs(got.Volume-0.5) > 0.05 { // 0.5 x 1 x 1 cm overlap
		t.Errorf("overlap volume = %.4f, want ~0.5", got.Volume)
	}
	if math.Abs(res.TotalVolume-0.5) > 0.05 {
		t.Errorf("total volume = %.4f, want ~0.5", res.TotalVolume)
	}

	// Disjoint boxes: no interference.
	r2, s2, _, _ := assemblySessionWithBoxes(t, 0, 2)
	var none wire.InterferenceResultsResult
	call(t, r2, s2, "interference.analyze", `{}`, &none)
	if len(none.Results) != 0 {
		t.Errorf("disjoint boxes reported %d interferences, want 0", len(none.Results))
	}
}

// TestContactSetsOverWire drives the contact-set management + solver toggle over the wire.
func TestContactSetsOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 2)

	var cs wire.ContactSetResult
	call(t, r, s, "contactSets.create", mustJSON(t, wire.CreateContactSetArgs{Name: "group"}), &cs)
	call(t, r, s, "contactSets.addMember", mustJSON(t, wire.ContactMemberArgs{Set: cs.ContactSet.ID, Occurrence: occs[0].ID()}), &wire.ContactSetResult{})
	var added wire.ContactSetResult
	call(t, r, s, "contactSets.addMember", mustJSON(t, wire.ContactMemberArgs{Set: cs.ContactSet.ID, Occurrence: occs[1].ID()}), &added)
	if len(added.ContactSet.Members) != 2 {
		t.Fatalf("members = %v, want 2", added.ContactSet.Members)
	}

	var sv wire.ContactSolverResult
	call(t, r, s, "contactSolver.setEnabled", mustJSON(t, wire.ContactSolverEnableArgs{Enabled: true}), &sv)
	if !sv.Solver.Enabled || sv.Solver.SetCount != 1 {
		t.Errorf("solver state = %+v, want enabled with 1 set", sv.Solver)
	}

	var afterDel wire.ContactSetsResult
	call(t, r, s, "contactSets.delete", mustJSON(t, wire.ContactSetRef{ID: cs.ContactSet.ID}), &afterDel)
	if len(afterDel.ContactSets) != 0 {
		t.Errorf("after delete = %+v, want empty", afterDel.ContactSets)
	}
}
