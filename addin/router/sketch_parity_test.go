// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
)

// TestSplineFitMethodOverWire: a spline created with a fit method reports it
// in the entity enumeration; unknown spellings are rejected (M06-F11, #626).
func TestSplineFitMethodOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var added wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity",
		`{"sketchIndex":0,"kind":"spline","points":[[0,0],[1,1],[2,0]],"fitMethod":"chord"}`, &added)

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	found := false
	for _, e := range ents.Entities {
		if e.ID == added.EntityID {
			found = true
			if e.FitMethod != "chord" {
				t.Errorf("fit method = %q, want chord", e.FitMethod)
			}
			if e.MoveableStatus != "freeToMove" {
				t.Errorf("moveable status = %q, want freeToMove for the unconstrained spline", e.MoveableStatus)
			}
		}
	}
	if !found {
		t.Fatal("the created spline is missing from the enumeration")
	}

	if _, err := r.Handle(s, "sketch.addEntity",
		[]byte(`{"sketchIndex":0,"kind":"spline","points":[[0,0],[1,1],[2,0]],"fitMethod":"bouncy"}`)); err == nil {
		t.Error("an unknown fit method must be rejected")
	}
}

// TestSplineHandleOverWire: activate → edit → deactivate, with the handle
// visible in the entity enumeration while active.
func TestSplineHandleOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var added wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity",
		`{"sketchIndex":0,"kind":"spline","points":[[0,5],[1,6],[2,5]]}`, &added)

	var h wire.SplineHandleInfo
	call(t, r, s, "sketch.setSplineHandle",
		fmt.Sprintf(`{"sketchIndex":0,"spline":%d,"fitPointIndex":1,"active":true,"tangent":[0,1],"weight":2}`, added.EntityID), &h)
	if h.HandleID == 0 || len(h.Tangent) != 2 {
		t.Fatalf("handle info = %+v, want an id and a 2D tangent", h)
	}
	if h.Weight < 1.9 || h.Weight > 2.1 {
		t.Errorf("handle weight = %v, want ~2", h.Weight)
	}

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	seen := false
	for _, e := range ents.Entities {
		if e.Kind == "splineHandle" && e.ID == h.HandleID {
			seen = true
		}
	}
	if !seen {
		t.Error("the active handle must enumerate as a splineHandle entity")
	}

	// Fresh DTO: the deactivation reply is {} (all omitempty), which would
	// not reset a reused var (the router-test JSON merge trap).
	var off wire.SplineHandleInfo
	call(t, r, s, "sketch.setSplineHandle",
		fmt.Sprintf(`{"sketchIndex":0,"spline":%d,"fitPointIndex":1,"active":false}`, added.EntityID), &off)
	if off.HandleID != 0 {
		t.Errorf("deactivation returned handle id %d, want 0", off.HandleID)
	}
}

// TestTextBoxAnchorOverWire: the auto-created anchor enumerates as a
// non-deletable textBox constraint and deleteConstraint refuses it.
func TestTextBoxAnchorOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "sketch.addText",
		`{"sketchIndex":0,"anchor":[1,1],"text":"label","height":"5 mm"}`, nil)

	var cons wire.ListConstraintsResult
	call(t, r, s, "sketch.constraints", `{"sketchIndex":0}`, &cons)
	anchorIndex := -1
	for _, c := range cons.Constraints {
		if c.Kind == "textBox" {
			anchorIndex = c.Index
			if c.Deletable {
				t.Error("the text-box anchor must report deletable=false")
			}
		}
	}
	if anchorIndex < 0 {
		t.Fatal("adding text must surface its anchor constraint in the enumeration")
	}
	if _, err := r.Handle(s, "sketch.deleteConstraint",
		[]byte(fmt.Sprintf(`{"sketchIndex":0,"constraintIndex":%d}`, anchorIndex))); err == nil {
		t.Error("deleting the system-owned anchor must be refused")
	}
}

// TestCustomConstraintOverWire: an add-in tags entities; the tag enumerates
// with its owner; an anonymous tag is rejected.
func TestCustomConstraintOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	lineID := ents.Entities[0].ID

	var added wire.AddConstraintResult
	call(t, r, s, "sketch.addConstraint",
		fmt.Sprintf(`{"sketchIndex":0,"kind":"custom","entities":[%d],"clientId":"com.x.cam","name":"toolpath-edge"}`, lineID), &added)
	if added.Kind != "custom" {
		t.Errorf("added kind = %q, want custom", added.Kind)
	}

	var cons wire.ListConstraintsResult
	call(t, r, s, "sketch.constraints", `{"sketchIndex":0}`, &cons)
	raw, _ := json.Marshal(cons)
	found := false
	for _, c := range cons.Constraints {
		if c.Kind == "custom" {
			found = true
			if c.ClientID != "com.x.cam" || c.Name != "toolpath-edge" || !c.Deletable {
				t.Errorf("custom row = %+v, want owner/name/deletable", c)
			}
		}
	}
	if !found {
		t.Fatalf("custom constraint missing from enumeration: %s", raw)
	}

	if _, err := r.Handle(s, "sketch.addConstraint",
		[]byte(fmt.Sprintf(`{"sketchIndex":0,"kind":"custom","entities":[%d]}`, lineID))); err == nil {
		t.Error("an anonymous custom constraint must be rejected")
	}
}
