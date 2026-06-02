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
