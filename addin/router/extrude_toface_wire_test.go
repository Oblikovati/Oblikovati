// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"oblikovati.org/api/wire"
)

// featureAddResult mirrors the opregistry featureResult shape returned by features.add.
type featureAddResult struct {
	Bodies  int    `json:"bodies"`
	Healthy bool   `json:"healthy"`
	Reason  string `json:"reason"`
}

// TestExtrudeToFaceOverWire drives the new "to-face" extent through features.add (the path an
// add-in or MCP client uses): extrude a profile up to an existing body's planar face, given by its
// reference key. The new body must terminate at that face, not at a typed distance (#1226 engine
// → API). This is the add-in-drivable form of Extrude To-Face.
func TestExtrudeToFaceOverWire(t *testing.T) {
	r, s := boxPartSession(t) // base body: 40×30×50 mm, top face at z = 5 cm

	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	top := faceKeyAtZ(keys.Bodies[0].Faces, 4.9)
	if top == "" {
		t.Fatal("no top face found on the base body")
	}

	// A second profile on XY, extruded "to face" up to the base body's top face.
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":1,"width":"10 mm","height":"10 mm"}`, &wire.SketchRectangleResult{})
	args, _ := json.Marshal(map[string]any{
		"kind": "extrude",
		"args": map[string]any{
			"sketchIndex": 1, "profileIndex": 0, "extent": "to-face", "toFace": top, "operation": "new",
		},
	})
	var res featureAddResult
	call(t, r, s, "features.add", string(args), &res)
	if !res.Healthy {
		t.Fatalf("to-face extrude unhealthy: %s", res.Reason)
	}
	if res.Bodies != 2 {
		t.Fatalf("bodies = %d, want 2 (base + to-face body)", res.Bodies)
	}

	// The new body must reach the target plane (top face z ≈ 5), proving it terminated there.
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	if got := maxFaceZ(keys.Bodies[1].Faces); got < 4.9 || got > 5.1 {
		t.Errorf("to-face body top z = %v, want ~5 (terminated at the picked face)", got)
	}
}

// TestExtrudeToFaceRequiresTarget: the to-face extent without a toFace reference is a clear error,
// not a silent default.
func TestExtrudeToFaceRequiresTarget(t *testing.T) {
	r, s := boxPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":1,"width":"10 mm","height":"10 mm"}`, &wire.SketchRectangleResult{})
	if _, err := r.Handle(s, "features.add", []byte(`{"kind":"extrude","args":{"sketchIndex":1,"extent":"to-face"}}`)); err == nil {
		t.Fatal("to-face extrude without a toFace target should error")
	}
}

// faceKeyAtZ returns the key of the first face whose representative point sits at or above minZ.
func faceKeyAtZ(faces []wire.TopologyRef, minZ float64) string {
	for _, f := range faces {
		if len(f.Point) == 3 && f.Point[2] >= minZ {
			return f.Key
		}
	}
	return ""
}

// maxFaceZ returns the greatest representative-point z over the faces.
func maxFaceZ(faces []wire.TopologyRef) float64 {
	max := -1e18
	for _, f := range faces {
		if len(f.Point) == 3 && f.Point[2] > max {
			max = f.Point[2]
		}
	}
	return max
}
