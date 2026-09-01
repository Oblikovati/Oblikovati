// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestSimplifyFillVoidsOverWire runs simplify(fillVoids) on a box (a no-op fill that must keep
// a valid solid) and rejects an empty request (M20-F13 #490).
func TestSimplifyFillVoidsOverWire(t *testing.T) {
	t.Parallel()
	r, s, _ := filletBoxFixture(t)
	call(t, r, s, "features.add", `{"kind":"simplify","args":{"fillVoids":true}}`, &struct {
		Bodies int `json:"bodies"`
	}{})
	var v wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0}`, &v)
	if !v.Valid {
		t.Fatalf("simplified body invalid: %+v", v.Problems)
	}
	if _, err := r.Handle(s, "features.add", []byte(`{"kind":"simplify","args":{}}`)); err == nil {
		t.Error("expected an error for an empty simplify (no faces, no fillVoids)")
	}
}

// TestUnwrapOverWire extrudes a cylinder, unwraps its side face, and checks a second (sheet)
// body appears — the flattened patch (M20-F13 #490).
func TestUnwrapOverWire(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0]],"radius":"20 mm"}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"30 mm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})

	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	var side string
	for _, f := range keys.Bodies[0].Faces {
		if len(f.Point) == 3 && f.Point[2] > 0.1 && f.Point[2] < 2.9 { // mid-height ⇒ the cylindrical side
			side = f.Key
		}
	}
	if side == "" {
		t.Fatal("no cylindrical side face found")
	}
	call(t, r, s, "features.add", mustJSON(t, map[string]any{"kind": "unwrap", "args": map[string]any{"faceRef": side}}), &struct {
		Bodies int `json:"bodies"`
	}{})
	if n := modelTreeOf(t, r, s).Bodies; n != 2 {
		t.Errorf("after unwrap body count = %d, want 2 (cylinder + flat patch)", n)
	}
}
