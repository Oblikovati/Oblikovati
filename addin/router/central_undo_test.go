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

// TestCentralSeamRecordsSketch3DEntityUndo proves 3D-sketch geometry added over the wire
// (sketch3d.addEntity) is undoable and replicated, exactly like its 2D parallel. Before #1426 the whole
// sketch3d authoring family was absent from the mutating table — silently non-undoable; wiring it through
// the MutatingMethod interface fixes the drift.
func TestCentralSeamRecordsSketch3DEntityUndo(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[3,0,4]]}`, &wire.AddSketch3DEntityResult{})

	st := undoState(t, r, s)
	if !st.CanUndo || st.NextUndo != "Add Sketch Geometry" {
		t.Fatalf("after sketch3d.addEntity state = %+v, want canUndo with nextUndo=Add Sketch Geometry", st)
	}

	call(t, r, s, "transaction.undo", "{}", nil)
	if st := undoState(t, r, s); st.NextUndo != "Create Sketch" {
		t.Fatalf("after one undo nextUndo = %q, want \"Create Sketch\" (only the entity reverted)", st.NextUndo)
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

// TestMutatingMethodsImplementInterface is the #1426 enforcement: the router's notion of "mutating" IS
// the MutatingMethod interface — every handler it records + replicates implements MutatingMethod, and no
// read-only handler does. Because the classification is the interface (not a side table), a mutating
// method cannot exist without implementing the contract (and declaring its UndoLabel), so it can never
// silently drift out of undo/replication.
func TestMutatingMethodsImplementInterface(t *testing.T) {
	r := New(opregistry.Default())
	for method, h := range r.handlers {
		_, implementsInterface := h.(MutatingMethod)
		_, classifiedMutating := r.MutatingMethods()[method]
		if implementsInterface != classifiedMutating {
			t.Errorf("%s: implements MutatingMethod=%v but recorded-as-mutating=%v — the interface must be the sole classifier",
				method, implementsInterface, classifiedMutating)
		}
	}
	// Every method the central seam records MUST satisfy the interface (this is what makes its UndoLabel
	// reachable). A regression that recorded a method by some other signal would trip here.
	for method := range r.MutatingMethods() {
		if _, ok := r.handlers[method].(MutatingMethod); !ok {
			t.Errorf("%s is recorded as a document edit but does not implement MutatingMethod", method)
		}
	}
}

// TestRegistrationHelpersProduceCorrectInterface is the #1426 drift guard at the registration seam: a
// handler registered through mutating() implements MutatingMethod (and carries its label); one registered
// through readOnly() deliberately does NOT, so the router never records or replicates it. This is the one
// pattern a document-editing method must follow — there is no second list to forget.
func TestRegistrationHelpersProduceCorrectInterface(t *testing.T) {
	r := New(opregistry.Default())
	nop := func(_ *app.Session, _ json.RawMessage) (json.RawMessage, error) { return nil, nil }

	r.readOnly("test.queryOnly", nop)
	if _, isMut := r.handlers["test.queryOnly"].(MutatingMethod); isMut {
		t.Error("a readOnly handler must NOT implement MutatingMethod (it would wrongly record + replicate)")
	}

	r.mutating("test.editsDoc", "Test Edit", nop)
	mut, isMut := r.handlers["test.editsDoc"].(MutatingMethod)
	if !isMut {
		t.Fatal("a mutating handler MUST implement MutatingMethod")
	}
	if mut.UndoLabel() != "Test Edit" {
		t.Errorf("UndoLabel = %q, want %q", mut.UndoLabel(), "Test Edit")
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
