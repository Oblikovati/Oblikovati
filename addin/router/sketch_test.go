// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/opregistry"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/compdef"
)

// emptyPartSession returns a router and a session with one active, empty part.
func emptyPartSession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	s := app.NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "scratch.obk", true); err != nil {
		t.Fatalf("add part: %v", err)
	}
	return New(opregistry.Default()), s
}

func TestSketchCreateAndRectangleThenExtrude(t *testing.T) {
	r, s := emptyPartSession(t)

	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &sk)
	if sk.SketchIndex != 0 || sk.Plane != "XY" {
		t.Fatalf("sketch.create = %+v, want index 0 plane XY", sk)
	}

	var rect wire.SketchRectangleResult
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &rect)
	if rect.Profiles != 1 {
		t.Fatalf("sketch.rectangle profiles = %d, want 1 closed profile", rect.Profiles)
	}

	var ext struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &ext)
	if ext.Bodies != 1 {
		t.Fatalf("extrude bodies = %d, want 1", ext.Bodies)
	}

	var tree wire.ModelTreeResult
	call(t, r, s, "model.tree", "{}", &tree)
	if tree.Sketches != 1 || tree.Bodies != 1 || len(tree.Features) != 1 {
		t.Fatalf("model.tree = %+v, want 1 sketch / 1 body / 1 feature", tree)
	}
}

// TestSketchSpineEnumerateEditSolveDelete exercises the F01 API spine end-to-end through
// the router: create a rectangle sketch, then list/get/enumerate/edit/solve/delete it.
func TestSketchSpineEnumerateEditSolveDelete(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})

	var list wire.ListSketchesResult
	call(t, r, s, "sketch.list", "{}", &list)
	if len(list.Sketches) != 1 {
		t.Fatalf("sketch.list = %d sketches, want 1", len(list.Sketches))
	}
	got := list.Sketches[0]
	if got.Plane != "XY" || got.Name == "" || got.EntityCount != 8 {
		t.Fatalf("sketch info = %+v, want plane XY, a name, 8 entities (4 lines + 4 points)", got)
	}
	if got.DOF != 8 {
		t.Fatalf("DOF = %d, want 8 (unconstrained rectangle: 4 points × 2)", got.DOF)
	}

	var gotOne wire.SketchInfo
	call(t, r, s, "sketch.get", `{"sketchIndex":0}`, &gotOne)
	if gotOne != got {
		t.Fatalf("sketch.get = %+v, want %+v from list", gotOne, got)
	}

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	if lines := countKind(ents.Entities, "line"); lines != 4 {
		t.Fatalf("entities: %d lines, want 4 (%+v)", lines, ents.Entities)
	}
	if pts := countKind(ents.Entities, "point"); pts != 4 {
		t.Fatalf("entities: %d points, want 4", pts)
	}

	var cons wire.ListConstraintsResult
	call(t, r, s, "sketch.constraints", `{"sketchIndex":0}`, &cons)
	if len(cons.Constraints) != 0 {
		t.Fatalf("constraints = %d, want 0 on a bare rectangle", len(cons.Constraints))
	}

	var ed wire.EditSketchResult
	call(t, r, s, "sketch.edit", `{"sketchIndex":0}`, &ed)
	if !ed.Editing {
		t.Fatal("sketch.edit: Editing = false, want true")
	}
	call(t, r, s, "sketch.exitEdit", `{"sketchIndex":0}`, &ed)
	if ed.Editing {
		t.Fatal("sketch.exitEdit: Editing = true, want false")
	}

	var solved wire.SolveSketchResult
	call(t, r, s, "sketch.solve", `{"sketchIndex":0}`, &solved)
	if solved.DOF != 8 || solved.Status != "under" {
		t.Fatalf("sketch.solve = %+v, want DOF 8 / status under", solved)
	}

	var ok wire.OKResult
	call(t, r, s, "sketch.delete", `{"sketchIndex":0}`, &ok)
	if !ok.OK {
		t.Fatal("sketch.delete: OK = false")
	}
	call(t, r, s, "sketch.list", "{}", &list)
	if len(list.Sketches) != 0 {
		t.Fatalf("after delete: %d sketches, want 0", len(list.Sketches))
	}
}

// countKind counts enumerated entities of the given wire kind.
func countKind(ents []wire.SketchEntityInfo, kind string) int {
	n := 0
	for _, e := range ents {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

func TestSketchCreateUnknownPlane(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "sketch.create", []byte(`{"plane":"AB"}`)); err == nil {
		t.Fatal("expected error for unknown plane")
	}
}

func TestSketchRectangleBadIndex(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "sketch.rectangle", []byte(`{"sketchIndex":5,"width":"1 cm","height":"1 cm"}`)); err == nil {
		t.Fatal("expected error for out-of-range sketch index")
	}
}
