// SPDX-License-Identifier: GPL-2.0-only

package router

import (
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

// TestMutatingMethodConsistency guards the single source of truth against drift, so the
// central seam keeps holding as methods are added:
//   - every mutating method must be a real router handler (a typo'd constant is dead);
//   - the four transaction-control methods must be broadcast-only (an empty label) — recording
//     an undo step while moving the cursor would corrupt the stream;
//   - the families this fix unblocked must keep a non-empty label (a regression tripwire).
func TestMutatingMethodConsistency(t *testing.T) {
	r := New(opregistry.Default())
	for method := range mutatingMethods {
		if _, ok := r.handlers[method]; !ok {
			t.Errorf("mutatingMethods[%q] is not a registered router handler", method)
		}
	}

	for _, m := range []string{
		wire.MethodTransactionUndo, wire.MethodTransactionRedo,
		wire.MethodTransactionEnd, wire.MethodTransactionAbort,
	} {
		if mutatingMethods[m] != "" {
			t.Errorf("transaction-control method %q must have an empty undo label, got %q", m, mutatingMethods[m])
		}
	}

	for _, m := range []string{
		wire.MethodFeaturesAdd, wire.MethodSketchAddEntity,
		wire.MethodWorkPlanesCreate, wire.MethodAssemblyFeaturesAdd,
	} {
		if mutatingMethods[m] == "" {
			t.Errorf("method %q must record an undo step (non-empty label); the central seam regressed", m)
		}
	}
}
