// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// The #330 acceptance over the wire: a body splits into multiple solids
// (typed with the frozen SplitType spellings) and two bodies combine.

// splitFixture extrudes a 40×30×50 mm box and adds a z=25 mm offset work plane.
func splitFixture(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"25 mm"}`, &wire.CreateWorkPlaneResult{})
	return r, s
}

type splitAddResult struct {
	Bodies int `json:"bodies"`
}

// TestSplitBodyThenCombineOverWire: type splitBody yields two solids; combine
// joins them back into one.
func TestSplitBodyThenCombineOverWire(t *testing.T) {
	t.Parallel()
	r, s := splitFixture(t)
	var res splitAddResult
	call(t, r, s, "features.add", `{"kind":"splitSolid","args":{"workPlaneIndex":3,"type":"splitBody"}}`, &res)
	if res.Bodies != 2 {
		t.Fatalf("splitBody bodies = %d, want 2", res.Bodies)
	}
	var joined splitAddResult
	call(t, r, s, "features.add", `{"kind":"combine","args":{"targetIndex":0,"toolIndex":1,"operation":"join"}}`, &joined)
	if joined.Bodies != 1 {
		t.Fatalf("combine bodies = %d, want 1", joined.Bodies)
	}
}

// TestSplitFacesOverWire: type splitFaces keeps one body (volume untouched)
// but the box gains the imprint faces.
func TestSplitFacesOverWire(t *testing.T) {
	t.Parallel()
	r, s := splitFixture(t)
	var res splitAddResult
	call(t, r, s, "features.add", `{"kind":"splitSolid","args":{"workPlaneIndex":3,"type":"splitFaces"}}`, &res)
	if res.Bodies != 1 {
		t.Fatalf("splitFaces bodies = %d, want 1", res.Bodies)
	}
	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	if n := len(keys.Bodies[0].Faces); n != 10 {
		t.Errorf("imprinted box faces = %d, want 10", n)
	}
	var v wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0}`, &v)
	if !v.Valid {
		t.Fatalf("imprinted body invalid: %+v", v.Problems)
	}
}

// TestSplitTrimSolidOverWire: type trimSolid keeps one side (default positive).
func TestSplitTrimSolidOverWire(t *testing.T) {
	t.Parallel()
	r, s := splitFixture(t)
	var res splitAddResult
	call(t, r, s, "features.add", `{"kind":"splitSolid","args":{"workPlaneIndex":3,"type":"trimSolid"}}`, &res)
	if res.Bodies != 1 {
		t.Fatalf("trimSolid bodies = %d, want 1", res.Bodies)
	}
	if _, err := r.Handle(s, "features.add",
		[]byte(`{"kind":"splitSolid","args":{"workPlaneIndex":3,"type":"halve"}}`)); err == nil {
		t.Fatal("an unknown split type must be rejected")
	}
}
