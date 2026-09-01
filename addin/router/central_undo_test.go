// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/drawing"
)

// activeDrawingViews returns the number of views on the active drawing's active sheet — the
// observable the drawing central-seam test checks across an undo to prove the view actually
// rolled back, not just that the cursor reports it can.
func activeDrawingViews(t *testing.T, s *app.Session) int {
	t.Helper()
	c, ok := s.ActiveDocument().Content().(*drawing.Content)
	if !ok {
		t.Fatalf("active document content is %T, want *drawing.Content", s.ActiveDocument().Content())
	}
	return c.Sheets().Active().Views().Count()
}

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
	t.Parallel()
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

// TestCentralSeamRecordsSketchTextUndo proves sketch annotation text added over the wire (sketch.addText)
// is undoable — a sketch-authoring path missed by the original mutating table (#1426).
func TestCentralSeamRecordsSketchTextUndo(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addText", `{"sketchIndex":0,"anchor":[1,1],"text":"PART A","height":"5 mm"}`, &wire.AddEntityIDResult{})

	st := undoState(t, r, s)
	if !st.CanUndo || st.NextUndo != "Add Text" {
		t.Fatalf("after sketch.addText state = %+v, want canUndo with nextUndo=Add Text", st)
	}

	call(t, r, s, "transaction.undo", "{}", nil)
	if st := undoState(t, r, s); st.NextUndo != "Create Sketch" {
		t.Fatalf("after one undo nextUndo = %q, want \"Create Sketch\" (only the text reverted)", st.NextUndo)
	}
}

// TestCentralSeamRecordsBodyDeleteUndo proves deleting a body over the wire (body.delete, which records
// a DeleteBody feature) is one undo step that restores the body — body editing was read-only before #1426.
func TestCentralSeamRecordsBodyDeleteUndo(t *testing.T) {
	t.Parallel()
	r, s, def := twoBodyPartSession(t)

	call(t, r, s, "body.delete", `{"bodyIndex":0}`, &wire.BodyListResult{})
	if n := len(def.SurfaceBodies().All()); n != 1 {
		t.Fatalf("after body.delete: %d bodies, want 1", n)
	}

	st := undoState(t, r, s)
	if !st.CanUndo || st.NextUndo != "Delete Body" {
		t.Fatalf("after body.delete state = %+v, want canUndo with nextUndo=Delete Body", st)
	}

	call(t, r, s, "transaction.undo", "{}", nil)
	if n := len(def.SurfaceBodies().All()); n != 2 {
		t.Errorf("after undo: %d bodies, want 2 (the delete reverted)", n)
	}
}

// TestCentralSeamRecordsDrawingViewUndo proves a drawing edit over the wire (drawingViews.addBase)
// is now one undo step that reverts the view. The whole drawing authoring surface was classified
// mutating in #1447 for replication, but its undo labels were dead until DrawingContent gained
// recipe-snapshot support (#1448): with no MarshalSnapshot the central seam recorded nothing. This
// is the activation test for that support — addBase adds a view, undo removes it, redo restores it.
func TestCentralSeamRecordsDrawingViewUndo(t *testing.T) {
	t.Parallel()
	r, s := drawingViewSession(t) // a part with geometry + an active drawing referencing it
	if got := activeDrawingViews(t, s); got != 0 {
		t.Fatalf("fresh drawing already has %d views, want 0", got)
	}

	call(t, r, s, "drawingViews.addBase", `{"name":"FRONT","orientation":"front","scale":2,"centerXmm":120,"centerYmm":100}`, &wire.ViewResult{})
	if got := activeDrawingViews(t, s); got != 1 {
		t.Fatalf("after drawingViews.addBase: %d views, want 1", got)
	}

	st := undoState(t, r, s)
	if !st.CanUndo || st.NextUndo != "Add View" {
		t.Fatalf("after drawingViews.addBase state = %+v, want canUndo with nextUndo=Add View", st)
	}

	call(t, r, s, "transaction.undo", "{}", nil)
	if got := activeDrawingViews(t, s); got != 0 {
		t.Errorf("after undo: %d views, want 0 (the view reverted)", got)
	}

	call(t, r, s, "transaction.redo", "{}", nil)
	if got := activeDrawingViews(t, s); got != 1 {
		t.Errorf("after redo: %d views, want 1 (the view restored)", got)
	}
}

// TestCentralSeamRecordsAssemblyPlacementUndo proves placing a component over the wire
// (assembly.placeByDefinition) is one undo step that reverts the occurrence — the assembly authoring
// family was registered read-only before #1426, so wire-driven placements were silently non-undoable and
// not replicated to collaborators.
func TestCentralSeamRecordsAssemblyPlacementUndo(t *testing.T) {
	t.Parallel()
	r, s, asm, _ := assemblySessionWithBoxes(t) // empty assembly, kept active
	pin := openPartDoc(t, s, "pin.obk")         // a doc-backed component the snapshot can reference

	var placed wire.OccurrenceResult
	args := fmt.Sprintf(`{"document":%d,"name":"pin:1","transform":%s}`, uint64(pin.ID()), transformJSON(2, 0, 0))
	call(t, r, s, "assembly.place", args, &placed)
	if got := asm.Occurrences().Count(); got != 1 {
		t.Fatalf("after assembly.place occurrences = %d, want 1", got)
	}

	st := undoState(t, r, s)
	if !st.CanUndo || st.NextUndo != "Place Component" {
		t.Fatalf("after assembly.place state = %+v, want canUndo with nextUndo=Place Component", st)
	}

	call(t, r, s, "transaction.undo", "{}", nil)
	if got := asm.Occurrences().Count(); got != 0 {
		t.Errorf("after undo occurrences = %d, want 0 (the placement reverted)", got)
	}
}

// TestCentralSeamRecordsSketch3DEntityUndo proves 3D-sketch geometry added over the wire
// (sketch3d.addEntity) is undoable and replicated, exactly like its 2D parallel. Before #1426 the whole
// sketch3d authoring family was absent from the mutating table — silently non-undoable; wiring it through
// the MutatingMethod interface fixes the drift.
func TestCentralSeamRecordsSketch3DEntityUndo(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
