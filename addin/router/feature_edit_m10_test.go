// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
)

// The #704 acceptance over the wire: features.get exposes the new editable scalars and
// features.edit drives a recompute with correct geometry.

func TestFreeformLevelEditableOverWire(t *testing.T) {
	r, s, id := freeformBoxViaAPI(t)

	var got wire.FeatureDetailResult
	call(t, r, s, "features.get", fmt.Sprintf(`{"id":%d}`, id), &got)
	if len(got.Feature.Scalars) != 1 || got.Feature.Scalars[0].Label != "Level" || !got.Feature.Scalars[0].Integer {
		t.Fatalf("freeform scalars = %+v, want one integer Level", got.Feature.Scalars)
	}

	call(t, r, s, "features.edit", fmt.Sprintf(`{"id":%d,"scalars":[{"index":0,"value":"0"}]}`, id), &got)
	if faces := partBodyFaces(t, s); faces != 6 {
		t.Errorf("after features.edit level→0: %d faces, want 6 (the raw cage)", faces)
	}
}

func TestBossScalarsEditableOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &struct{}{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"distance":"20 mm"}}`, &struct{}{})

	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	top := ""
	for _, f := range keys.Bodies[0].Faces {
		if len(f.Point) == 3 && f.Point[2] > 1.9 { // the 20 mm (=2 cm) top face
			top = f.Key
		}
	}
	if top == "" {
		t.Fatal("fixture: no top face key found")
	}
	// Reference keys carry raw bytes — marshal the args, never %q-format them.
	args, err := json.Marshal(map[string]interface{}{
		"kind": "boss",
		"args": map[string]interface{}{"faceRef": top, "diameter": "8 mm", "height": "5 mm"},
	})
	if err != nil {
		t.Fatal(err)
	}
	call(t, r, s, "features.add", string(args), &struct{}{})

	tree := modelTreeOf(t, r, s)
	bossID := tree.Features[len(tree.Features)-1].ID
	var got wire.FeatureDetailResult
	call(t, r, s, "features.get", fmt.Sprintf(`{"id":%d}`, bossID), &got)
	if len(got.Feature.Scalars) != 2 {
		t.Fatalf("boss scalars = %+v, want diameter + height", got.Feature.Scalars)
	}
	call(t, r, s, "features.edit", fmt.Sprintf(`{"id":%d,"scalars":[{"index":1,"value":"9 mm"}]}`, bossID), &got)
	if got.Feature.Scalars[1].Value != 9 {
		t.Errorf("edited boss height = %g, want 9 (mm)", got.Feature.Scalars[1].Value)
	}
}
