// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// The fillet edge-set form over the wire (M09-F01 PBI-099, #323): features.add
// fillet accepts edgeSets mixing constant and variable radius sets.

// filletBoxFixture extrudes a 40×30×50 mm box and returns the reference keys
// of its vertical edges (midpoint z = 2.5 cm).
func filletBoxFixture(t *testing.T) (*Router, *app.Session, []string) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct {
		Bodies int `json:"bodies"`
	}{})
	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	var vertical []string
	for _, e := range keys.Bodies[0].Edges {
		if len(e.Point) == 3 && e.Point[2] > 2.4 && e.Point[2] < 2.6 {
			vertical = append(vertical, e.Key)
		}
	}
	if len(vertical) != 4 {
		t.Fatalf("vertical edges = %d, want 4", len(vertical))
	}
	return r, s, vertical
}

type filletSetsAddResult struct {
	Bodies int `json:"bodies"`
}

// TestFilletEdgeSetsOverWire adds a constant + variable edge-set fillet via
// features.add (keys carry raw bytes, so the args are marshalled, not quoted).
func TestFilletEdgeSetsOverWire(t *testing.T) {
	r, s, vertical := filletBoxFixture(t)
	args, _ := json.Marshal(map[string]interface{}{
		"kind": "fillet",
		"args": map[string]interface{}{
			"edgeSets": []map[string]interface{}{
				{"edgeRefs": []string{vertical[0]}, "radius": "4 mm"},
				{"edgeRefs": []string{vertical[1]}, "startRadius": "2 mm", "endRadius": "7 mm"},
			},
		},
	})
	var res filletSetsAddResult
	call(t, r, s, "features.add", string(args), &res)
	if res.Bodies != 1 {
		t.Fatalf("edge-set fillet bodies = %d, want 1", res.Bodies)
	}
	var v wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0}`, &v)
	if !v.Valid {
		t.Fatalf("edge-set filleted body invalid: %+v", v.Problems)
	}
}

// TestFilletEdgeSetRejectsMixedSpelling: a set giving both a constant radius
// and a variable pair is a precise error.
func TestFilletEdgeSetRejectsMixedSpelling(t *testing.T) {
	r, s, vertical := filletBoxFixture(t)
	args, _ := json.Marshal(map[string]interface{}{
		"kind": "fillet",
		"args": map[string]interface{}{
			"edgeSets": []map[string]interface{}{
				{"edgeRefs": []string{vertical[0]}, "radius": "4 mm", "startRadius": "2 mm", "endRadius": "7 mm"},
			},
		},
	})
	if _, err := r.Handle(s, "features.add", args); err == nil {
		t.Fatal("a set with both radius and startRadius/endRadius must be rejected")
	}
}
