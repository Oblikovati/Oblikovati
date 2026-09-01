// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
)

// The PBI-095 acceptance over the wire (M08 #315): a multi-section loft with
// guide rails and end conditions blends and recomputes through features.add.
// The model-layer behavior is pinned by the loft_* test files; this drives the
// same definition through the public surface.

func TestLoftWithRailsAndConditionsOverWire(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	// Section 0: 40 mm square at z=0; section 1: 20 mm square at z=50 (via an
	// offset work plane).
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"40 mm"}`, &wire.SketchRectangleResult{})
	var wp wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"50 mm"}`, &wp)
	call(t, r, s, "sketch.create", fmt.Sprintf(`{"workPlaneIndex":%d}`, wp.Index), &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":1,"width":"20 mm","height":"20 mm"}`, &wire.SketchRectangleResult{})
	// Rail: a straight guide on XZ from a base corner to a top corner.
	call(t, r, s, "sketch.create", `{"plane":"XZ"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":2,"kind":"line","points":[[20,0],[10,50]]}`, &wire.AddSketchEntityResult{})

	var res struct {
		Bodies int `json:"bodies"`
	}
	call(t, r, s, "features.add", `{"kind":"loft","args":{
		"sections":[{"sketchIndex":0,"profileIndex":0},{"sketchIndex":1,"profileIndex":0}],
		"rails":[{"pathSketchIndex":2,"pathIndex":0}],
		"first":{"condition":"tangent"},
		"operation":"new"}}`, &res)
	if res.Bodies != 1 {
		t.Fatalf("loft bodies = %d, want 1", res.Bodies)
	}
	var list wire.BodyListResult
	call(t, r, s, "body.list", `{}`, &list)
	if !list.Bodies[0].Solid {
		t.Fatal("lofted body should be a solid")
	}
	var valid wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0,"checkLevel":1}`, &valid)
	if !valid.Valid {
		t.Fatalf("lofted body invalid: %+v", valid.Problems)
	}
}
