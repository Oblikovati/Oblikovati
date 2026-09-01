// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// The #866 acceptance over the wire: the modelTolerance op annotates a face with a GD&T frame
// + datum through features.add without changing the body, and rejects an empty request.

func TestModelToleranceOverWire(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})

	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	face := keys.Bodies[0].Faces[0].Key

	var res struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add", mustJSON(t, map[string]any{
		"kind": "modelTolerance",
		"args": map[string]any{
			"frames": []map[string]any{{"geometry": face, "characteristic": "flatness", "value": "0.05 mm", "datums": []string{"A"}}},
			"datums": []map[string]any{{"geometry": face, "label": "A"}},
		},
	}), &res)
	if res.Bodies != 1 {
		t.Fatalf("modelTolerance changed body count to %d, want 1 (annotation passes the solid through)", res.Bodies)
	}
}

func TestModelToleranceOverWireRejectsEmpty(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "features.add", []byte(`{"kind":"modelTolerance","args":{}}`)); err == nil {
		t.Fatal("an empty modelTolerance (no frames, no datums) must error")
	}
}
