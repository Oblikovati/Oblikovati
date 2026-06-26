// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// activePartBodies returns the number of solid bodies on the session's active part — the
// observable the central-seam tests check before and after an undo to prove the model
// actually rolled back, not just that the cursor reports it can.
func activePartBodies(t *testing.T, s *app.Session) int {
	t.Helper()
	part, ok := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if !ok {
		t.Fatalf("active document content is %T, want *compdef.PartComponentDefinition", s.ActiveDocument().Content())
	}
	return len(part.SurfaceBodies().All())
}

// undoState reads the transaction.state control surface.
func undoState(t *testing.T, r *Router, s *app.Session) wire.UndoState {
	t.Helper()
	var st wire.UndoState
	call(t, r, s, "transaction.state", "{}", &st)
	return st
}

// TestCentralSeamRecordsFeatureUndo pins the headline fix: a feature created over the wire
// (features.add) is now one undo step, even though no opregistry handler calls recordEdit —
// the central seam in Handle records it. Before this change the extrude was silently
// un-undoable (recomputeResult recomputes but never recorded). Undo must drop the new body.
func TestCentralSeamRecordsFeatureUndo(t *testing.T) {
	r, s := seededSession(t) // active part with one rectangle profile on sketch 0
	if got := activePartBodies(t, s); got != 0 {
		t.Fatalf("seeded part already has %d bodies, want 0", got)
	}

	var ext extrudeBodies
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"distance":"5 mm"}}`, &ext)
	if ext.Bodies != 1 {
		t.Fatalf("extrude via API produced %d bodies, want 1", ext.Bodies)
	}

	st := undoState(t, r, s)
	if !st.CanUndo || st.NextUndo != "Add Feature" {
		t.Fatalf("after extrude state = %+v, want canUndo with nextUndo=Add Feature", st)
	}

	call(t, r, s, "transaction.undo", "{}", nil)
	if got := activePartBodies(t, s); got != 0 {
		t.Fatalf("after undo the part has %d bodies, want 0 (the extrude reverted)", got)
	}
	if st := undoState(t, r, s); st.CanUndo || !st.CanRedo {
		t.Fatalf("after undo state = %+v, want canRedo only", st)
	}
}

// TestCentralSeamRecordsSketchEntityUndo proves sketch geometry added over the wire
// (sketch.addEntity) is undoable — another path that previously recorded nothing.
func TestCentralSeamRecordsSketchEntityUndo(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, nil)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"10 mm"}`, nil)

	st := undoState(t, r, s)
	if !st.CanUndo || st.NextUndo != "Add Sketch Geometry" {
		t.Fatalf("after addEntity state = %+v, want canUndo with nextUndo=Add Sketch Geometry", st)
	}
}

// TestCentralSeamRecordsWorkPlaneUndo proves a work feature created over the wire
// (workPlanes.create) is undoable.
func TestCentralSeamRecordsWorkPlaneUndo(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, nil)

	st := undoState(t, r, s)
	if !st.CanUndo || st.NextUndo != "Create Work Plane" {
		t.Fatalf("after workPlanes.create state = %+v, want canUndo with nextUndo=Create Work Plane", st)
	}
}

// TestMutatingMethodLabels checks the undo-label contract of the registered mutating handlers,
// reading the live classification (MutatingMethods) the handlers declare — there is no separate table:
//   - the transaction-control methods must be broadcast-only (an empty label) — recording an undo step
//     while moving the cursor would corrupt the stream;
//   - the families the central seam unblocked must keep a non-empty label (a regression tripwire).
func TestMutatingMethodLabels(t *testing.T) {
	mut := New(opregistry.Default()).MutatingMethods()

	for _, m := range []string{
		wire.MethodTransactionUndo, wire.MethodTransactionRedo,
		wire.MethodTransactionEnd, wire.MethodTransactionAbort,
	} {
		if label, ok := mut[m]; !ok || label != "" {
			t.Errorf("transaction-control method %q must be mutating with an empty undo label, got ok=%v label=%q", m, ok, label)
		}
	}

	for _, m := range []string{
		wire.MethodFeaturesAdd, wire.MethodSketchAddEntity,
		wire.MethodWorkPlanesCreate, wire.MethodAssemblyFeaturesAdd,
	} {
		if mut[m] == "" {
			t.Errorf("method %q must record an undo step (non-empty label); the central seam regressed", m)
		}
	}
}

// TestMutatingDeclarationDrivesRecording is the #1426 drift guard: undo recording + replication are
// driven by the handler's OWN declaration (readOnly vs mutating), so a method cannot be mutating yet
// silently absent from a table — there is no table. A handler registered read-only is not in the
// mutating set; one registered mutating is, with its label. This replaces the old one-directional
// table-parity check, which could not catch a mutating handler missing from the table.
func TestMutatingDeclarationDrivesRecording(t *testing.T) {
	r := New(opregistry.Default())
	nop := func(_ *app.Session, _ json.RawMessage) (json.RawMessage, error) { return nil, nil }

	r.readOnly("test.queryOnly", nop)
	r.mutating("test.editsDoc", "Test Edit", nop)
	mut := r.MutatingMethods()

	if _, isMut := mut["test.queryOnly"]; isMut {
		t.Error("a read-only handler must not be classified mutating (it would wrongly record + replicate)")
	}
	if label, isMut := mut["test.editsDoc"]; !isMut || label != "Test Edit" {
		t.Errorf("a mutating handler must be classified mutating with its label, got ok=%v label=%q", isMut, label)
	}
}

// TestDuplicateRegistrationPanics guards against a copy-paste handler that would silently shadow another
// (and could change its mutation classification). set() is the one registration chokepoint.
func TestDuplicateRegistrationPanics(t *testing.T) {
	r := New(opregistry.Default())
	nop := func(_ *app.Session, _ json.RawMessage) (json.RawMessage, error) { return nil, nil }
	defer func() {
		if recover() == nil {
			t.Error("registering the same method twice must panic, not silently shadow the first handler")
		}
	}()
	r.readOnly("test.dup", nop)
	r.readOnly("test.dup", nop)
}
