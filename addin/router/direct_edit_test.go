// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"oblikovati.org/api/wire"
)

// The consolidated direct edit over the wire (M09-F04 PBI-108, #332).

func TestDirectEditScaleOverWire(t *testing.T) {
	r, s, _ := filletBoxFixture(t)
	var res struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add", `{"kind":"directEdit","args":{"operation":"scale","scale":2,"base":[0,0,0]}}`, &res)
	if res.Bodies != 1 {
		t.Fatalf("scale bodies = %d, want 1", res.Bodies)
	}
	var box wire.BodyRangeBoxResult
	call(t, r, s, "body.rangeBox", `{"bodyIndex":0}`, &box)
	// 40×30×50 mm doubled about the origin → max corner (8, 6, 10) cm.
	if box.Max[2] < 9.9 || box.Max[2] > 10.1 {
		t.Errorf("scaled box max z = %g, want 10", box.Max[2])
	}
}

func TestDirectEditSizeOverWire(t *testing.T) {
	r, s, _ := filletBoxFixture(t)
	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	var top string
	for _, f := range keys.Bodies[0].Faces {
		if len(f.Point) == 3 && f.Point[2] > 4.9 {
			top = f.Key
		}
	}
	args, _ := json.Marshal(map[string]any{
		"kind": "directEdit",
		"args": map[string]any{
			"operation": "size", "faceRefs": []string{top},
			"direction": []float64{0, 0, 1}, "distance": "10 mm",
		},
	})
	var res struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add", string(args), &res)
	if res.Bodies != 1 {
		t.Fatalf("size bodies = %d, want 1", res.Bodies)
	}
	var v wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0}`, &v)
	if !v.Valid {
		t.Fatalf("sized body invalid: %+v", v.Problems)
	}

	if _, err := r.Handle(s, "features.add",
		[]byte(`{"kind":"directEdit","args":{"operation":"shrinkwrap"}}`)); err == nil {
		t.Fatal("an unknown direct-edit operation must be rejected")
	}
}
