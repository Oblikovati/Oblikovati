// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"oblikovati.org/api/wire"
)

// The #331 face-edit extensions over the wire: the moveFace rotate arm and the
// faceOffset approximation input.

func TestMoveFaceRotateOverWire(t *testing.T) {
	r, s, _ := filletBoxFixture(t)
	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	var top string
	for _, f := range keys.Bodies[0].Faces {
		if len(f.Point) == 3 && f.Point[2] > 4.9 {
			top = f.Key
		}
	}
	if top == "" {
		t.Fatal("no top face found")
	}
	args, _ := json.Marshal(map[string]interface{}{
		"kind": "moveFace",
		"args": map[string]interface{}{
			"faceRefs":  []string{top},
			"axisPoint": []float64{0, 0, 5},
			"axisDir":   []float64{0, 1, 0},
			"angle":     "5 deg",
		},
	})
	var res struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add", string(args), &res)
	if res.Bodies != 1 {
		t.Fatalf("rotate-face bodies = %d, want 1", res.Bodies)
	}
	var v wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0}`, &v)
	if !v.Valid {
		t.Fatalf("rotated body invalid: %+v", v.Problems)
	}
}

func TestFaceOffsetApproximationOverWire(t *testing.T) {
	r, s, _ := filletBoxFixture(t)
	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	face := keys.Bodies[0].Faces[0].Key
	args, _ := json.Marshal(map[string]interface{}{
		"kind": "faceOffset",
		"args": map[string]interface{}{
			"faceRefs": []string{face}, "distance": "2 mm", "approximation": "neverTooThick",
		},
	})
	var res struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add", string(args), &res)
	if res.Bodies != 1 {
		t.Fatalf("faceOffset bodies = %d, want 1", res.Bodies)
	}

	bad, _ := json.Marshal(map[string]interface{}{
		"kind": "faceOffset",
		"args": map[string]interface{}{"faceRefs": []string{face}, "distance": "2 mm", "approximation": "sloppy"},
	})
	if _, err := r.Handle(s, "features.add", bad); err == nil {
		t.Fatal("an unknown approximation must be rejected")
	}
}
