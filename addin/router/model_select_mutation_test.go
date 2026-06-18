// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// boxFaceRefs builds a box and returns two of its faces' selection reference strings (the WorkRef
// shape model.selection reports), plus the session/router.
func boxFaceRefs(t *testing.T) (*Router, *app.Session, string, string) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	faces := part.SurfaceBodies().Item(0).Faces()
	if len(faces) < 2 {
		t.Fatalf("box has %d faces, want >= 2", len(faces))
	}
	return r, s, string(feature.FaceRef(faces[0].ReferenceKey())), string(feature.FaceRef(faces[1].ReferenceKey()))
}

// TestModelSelectMutationOverWire drives the #157 selection round trip: select two faces by
// reference, deselect one, add it back, then clear — asserting the new selection each step.
func TestModelSelectMutationOverWire(t *testing.T) {
	r, s, ref0, ref1 := boxFaceRefs(t)

	var sel wire.SelectionResult
	call(t, r, s, "model.select", mustJSON(t, wire.SelectArgs{Refs: []string{ref0, ref1}, Mode: "replace"}), &sel)
	if sel.Count != 2 {
		t.Fatalf("select replace = %+v, want 2", sel)
	}

	call(t, r, s, "model.deselect", mustJSON(t, wire.DeselectArgs{Refs: []string{ref1}}), &sel)
	if sel.Count != 1 || sel.Refs[0] != ref0 {
		t.Errorf("after deselect = %+v, want just ref0", sel)
	}

	call(t, r, s, "model.select", mustJSON(t, wire.SelectArgs{Refs: []string{ref1}, Mode: "add"}), &sel)
	if sel.Count != 2 {
		t.Errorf("after add = %+v, want 2", sel)
	}

	call(t, r, s, "model.clearSelection", "{}", &sel)
	if sel.Count != 0 {
		t.Errorf("after clear = %+v, want 0", sel)
	}
}

// TestModelSelectUnknownReferenceFails: an unresolvable reference is a rejection, not a silent no-op.
func TestModelSelectUnknownReferenceFails(t *testing.T) {
	r, s, _, _ := boxFaceRefs(t)
	if _, err := r.Handle(s, "model.select", []byte(`{"refs":["face/ZZZZ"]}`)); err == nil {
		t.Error("selecting an unknown face reference should fail")
	}
}
