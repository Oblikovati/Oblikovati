// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// extrudedSketchFixture: sketch 0 (a rectangle) consumed by an extrude, plus an unconsumed
// sketch 1. Returns the router/session and the extrude's feature id.
func extrudedSketchFixture(t *testing.T) (*Router, *app.Session, uint64) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"20 mm"}}`, &struct{}{})
	call(t, r, s, "sketch.create", `{"plane":"XZ"}`, &wire.CreateSketchResult{}) // sketch 1, unconsumed
	return r, s, lastFeatureID(t, r, s)
}

// TestSketchConsumedStateReported: sketch.get reports consumed/ownedBy for a sketch a feature
// uses, and clears them for an unconsumed sketch (#154).
func TestSketchConsumedStateReported(t *testing.T) {
	t.Parallel()
	r, s, _ := extrudedSketchFixture(t)

	var consumed wire.SketchInfo
	call(t, r, s, "sketch.get", `{"sketchIndex":0}`, &consumed)
	if !consumed.Consumed || consumed.OwnedBy == "" {
		t.Errorf("consumed sketch info = %+v, want consumed=true and a non-empty ownedBy", consumed)
	}

	var free wire.SketchInfo
	call(t, r, s, "sketch.get", `{"sketchIndex":1}`, &free)
	if free.Consumed || free.OwnedBy != "" {
		t.Errorf("unconsumed sketch info = %+v, want consumed=false and empty ownedBy", free)
	}
}

// TestSketchDependents: sketch.dependents lists the consuming feature for a used sketch and is
// empty for an unused one.
func TestSketchDependents(t *testing.T) {
	t.Parallel()
	r, s, extrudeID := extrudedSketchFixture(t)

	var deps wire.SketchDependentsResult
	call(t, r, s, "sketch.dependents", `{"sketchIndex":0}`, &deps)
	if len(deps.Dependents) != 1 || deps.Dependents[0].ID != extrudeID || deps.Dependents[0].Kind != "extrude" {
		t.Errorf("dependents of consumed sketch = %+v, want one extrude id %d", deps.Dependents, extrudeID)
	}

	call(t, r, s, "sketch.dependents", `{"sketchIndex":1}`, &deps)
	if len(deps.Dependents) != 0 {
		t.Errorf("dependents of unconsumed sketch = %+v, want none", deps.Dependents)
	}
}

// TestSketchDeleteRejectsConsumed: sketch.delete refuses a sketch a feature still uses and
// reports the dependent, but deletes an unused sketch (#154).
func TestSketchDeleteRejectsConsumed(t *testing.T) {
	t.Parallel()
	r, s, _ := extrudedSketchFixture(t)

	if _, err := r.Handle(s, "sketch.delete", []byte(`{"sketchIndex":0}`)); err == nil {
		t.Error("deleting a consumed sketch should fail")
	}
	// The unused sketch deletes fine.
	var ok wire.OKResult
	call(t, r, s, "sketch.delete", `{"sketchIndex":1}`, &ok)
	if !ok.OK {
		t.Error("deleting an unconsumed sketch should succeed")
	}
}

// TestSketchDependentsNoActivePart: the method errors without an active part.
func TestSketchDependentsNoActivePart(t *testing.T) {
	t.Parallel()
	r := New(opregistry.Default())
	s := app.NewSession()
	if _, err := r.Handle(s, "sketch.dependents", []byte(`{"sketchIndex":0}`)); err == nil {
		t.Error("sketch.dependents with no active part should fail")
	}
}
