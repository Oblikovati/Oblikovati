// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// twoExtrudePart builds a part with two independent extrudes (sketch1→extrude1, then
// sketch2→extrude2), returning the session, part, and both features in creation order — the
// fixture for edit-scope rollback (editing extrude1 must hide extrude2).
func twoExtrudePart(t *testing.T) (*Session, *compdef.PartComponentDefinition, *feature.PartFeature, *feature.PartFeature) {
	t.Helper()
	s, def := emptyPartSession(t)
	mkBox := func(originX float64) *feature.PartFeature {
		sk := def.Sketches().Add(sketch.XYPlane())
		c0 := sk.Points().Add(math.P2(originX, 0))
		c1 := sk.Points().Add(math.P2(originX+1, 0))
		c2 := sk.Points().Add(math.P2(originX+1, 1))
		c3 := sk.Points().Add(math.P2(originX, 1))
		sk.Lines().Add(c0, c1)
		sk.Lines().Add(c1, c2)
		sk.Lines().Add(c2, c3)
		sk.Lines().Add(c3, c0)
		return feature.NewExtrudeFeatures(def.Features()).
			AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 1 })
	}
	f1 := mkBox(0)
	f2 := mkBox(3)
	def.Recompute()
	return s, def, f1, f2
}

// TestBeginEditFeatureRollsBackToEditedFeature is the issue-#132 edit-scope regression:
// editing an earlier feature rolls the part back to it (its successor and everything after
// are excluded), and the scope reports later nodes as hidden. Commit restores full evaluation.
func TestBeginEditFeatureRollsBackToEditedFeature(t *testing.T) {
	t.Parallel()
	s, def, f1, f2 := twoExtrudePart(t)
	if def.Features().EndOfPartIndex() != -1 {
		t.Fatalf("part should start fully evaluated, EOP=%d", def.Features().EndOfPartIndex())
	}

	s.BeginEditFeature(FeatureHandle{Feature: f1})

	if _, active := s.EditScopeSeq(); !active {
		t.Fatal("edit scope should be active while editing")
	}
	if !s.EditScopeHides(f2.Seq()) {
		t.Error("the later feature must be hidden by the edit scope")
	}
	if s.EditScopeHides(f1.Seq()) {
		t.Error("the edited feature itself must NOT be hidden")
	}
	// The engine is rolled back to exclude f2 (index 1).
	if got := def.Features().EndOfPartIndex(); got != 1 {
		t.Errorf("EOP during edit = %d, want 1 (f2 excluded)", got)
	}

	if err := s.OK(); err != nil {
		t.Fatalf("commit edit: %v", err)
	}
	if _, active := s.EditScopeSeq(); active {
		t.Error("edit scope must clear on commit")
	}
	if got := def.Features().EndOfPartIndex(); got != -1 {
		t.Errorf("EOP after commit = %d, want -1 (full evaluation restored)", got)
	}
}

// TestEditScopeRestoredOnCancel: cancelling a feature edit also restores full evaluation.
func TestEditScopeRestoredOnCancel(t *testing.T) {
	t.Parallel()
	s, def, f1, _ := twoExtrudePart(t)
	s.BeginEditFeature(FeatureHandle{Feature: f1})
	s.CancelTool()
	if _, active := s.EditScopeSeq(); active {
		t.Error("edit scope must clear on cancel")
	}
	if got := def.Features().EndOfPartIndex(); got != -1 {
		t.Errorf("EOP after cancel = %d, want -1", got)
	}
}

// TestEditScopeHidesTrailingSketch: editing the first feature hides a sketch created later.
func TestEditScopeHidesTrailingSketch(t *testing.T) {
	t.Parallel()
	s, def, f1, _ := twoExtrudePart(t)
	later := def.Sketches().Add(sketch.XYPlane()) // created after both features
	def.Recompute()

	s.BeginEditFeature(FeatureHandle{Feature: f1})
	if !s.EditScopeHides(later.Seq()) {
		t.Error("a sketch created after the edited feature must be hidden")
	}
}

// TestSecondEditRestoresFirstScopeMarker is the scope-nesting regression: switching to a
// second edit without closing the first once captured the first edit's rolled-back marker as
// "the value to restore" and then had its scope cleared by the first tool's cancel — leaving
// the part permanently rolled back (later features never evaluated again) after commit.
func TestSecondEditRestoresFirstScopeMarker(t *testing.T) {
	t.Parallel()
	s, def, f1, f2 := twoExtrudePart(t)
	s.BeginEditFeature(FeatureHandle{Feature: f1}) // rolls back: f2 excluded
	s.BeginEditFeature(FeatureHandle{Feature: f2}) // switch edits without closing the first
	if err := s.OK(); err != nil {
		t.Fatalf("commit the second edit: %v", err)
	}
	if _, active := s.EditScopeSeq(); active {
		t.Error("edit scope must clear after the second edit commits")
	}
	if got := def.Features().EndOfPartIndex(); got != -1 {
		t.Errorf("EOP after switching edits = %d, want -1 — the first edit's marker leaked", got)
	}
}

// TestEditScopeRestoresMidHistoryMarker: a user's own end-of-part marker, parked mid-history
// before the edit opened, is restored by identity when the edit closes — not rolled to the end.
func TestEditScopeRestoresMidHistoryMarker(t *testing.T) {
	t.Parallel()
	s, def, f1, f2 := twoExtrudePart(t)
	if err := def.Features().SetEndOfPart(f2); err != nil {
		t.Fatalf("park the marker at f2: %v", err)
	}
	def.Recompute()

	s.BeginEditFeature(FeatureHandle{Feature: f1})
	if err := s.OK(); err != nil {
		t.Fatalf("commit the edit: %v", err)
	}
	if got := def.Features().EndOfPartIndex(); got != 1 {
		t.Errorf("EOP after the edit = %d, want 1 (the user's mid-history marker)", got)
	}
}

// TestEditSketchRefusedDuringOpenEdit: entering a sketch while a feature edit is open would
// overwrite that edit's scope, so it is refused — and the refusal surfaces as the notice
// instead of a silent no-op.
func TestEditSketchRefusedDuringOpenEdit(t *testing.T) {
	t.Parallel()
	s, def, f1, _ := twoExtrudePart(t)
	s.BeginEditFeature(FeatureHandle{Feature: f1})

	s.BeginEditSketch(SketchHandle{Sketch: def.Sketches().Item(0)})
	if s.InSketch() {
		t.Error("entering a sketch while a feature edit is open must be refused")
	}
	if s.Notice() == "" {
		t.Error("the refused sketch edit must surface a notice explaining why")
	}
}
