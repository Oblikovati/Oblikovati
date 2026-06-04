// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// TestSketch3DSpine exercises the M22-F01 API spine end-to-end through the router:
// create a 3D sketch, list/get it, toggle edit, solve, set properties, and delete.
func TestSketch3DSpine(t *testing.T) {
	r, s := emptyPartSession(t)

	var created wire.CreateSketch3DResult
	call(t, r, s, "sketch3d.create", `{}`, &created)
	if created.SketchIndex != 0 {
		t.Fatalf("sketch3d.create index = %d, want 0", created.SketchIndex)
	}

	var list wire.ListSketches3DResult
	call(t, r, s, "sketch3d.list", "{}", &list)
	if len(list.Sketches) != 1 {
		t.Fatalf("sketch3d.list = %d sketches, want 1", len(list.Sketches))
	}
	got := list.Sketches[0]
	if got.Name != "3D Sketch1" || !got.Visible || got.EntityCount != 0 || got.DOF != 0 {
		t.Fatalf("sketch3d info = %+v, want name '3D Sketch1', visible, 0 entities, 0 DOF", got)
	}

	var one wire.Sketch3DInfo
	call(t, r, s, "sketch3d.get", `{"sketchIndex":0}`, &one)
	if one != got {
		t.Fatalf("sketch3d.get = %+v, want %+v from list", one, got)
	}

	var edit wire.EditSketch3DResult
	call(t, r, s, "sketch3d.edit", `{"sketchIndex":0}`, &edit)
	if !edit.Editing {
		t.Fatal("sketch3d.edit should enter edit mode")
	}
	call(t, r, s, "sketch3d.exitEdit", `{"sketchIndex":0}`, &edit)
	if edit.Editing {
		t.Fatal("sketch3d.exitEdit should leave edit mode")
	}

	var solve wire.SolveSketch3DResult
	call(t, r, s, "sketch3d.solve", `{"sketchIndex":0}`, &solve)
	if solve.Status != "well" || !solve.Converged || !solve.Healthy {
		t.Fatalf("solve of an empty sketch = %+v, want well/converged/healthy", solve)
	}

	var prop wire.Sketch3DInfo
	call(t, r, s, "sketch3d.setProperty", `{"sketchIndex":0,"property":"name","value":"Path"}`, &prop)
	if prop.Name != "Path" {
		t.Fatalf("setProperty name = %q, want Path", prop.Name)
	}
	call(t, r, s, "sketch3d.setProperty", `{"sketchIndex":0,"property":"dimensionsVisible","value":"false"}`, &prop)
	if prop.DimensionsVisible {
		t.Fatal("setProperty dimensionsVisible=false did not stick")
	}

	var ok wire.OKResult
	call(t, r, s, "sketch3d.delete", `{"sketchIndex":0}`, &ok)
	if !ok.OK {
		t.Fatal("sketch3d.delete should succeed")
	}
	call(t, r, s, "sketch3d.list", "{}", &list)
	if len(list.Sketches) != 0 {
		t.Fatalf("after delete, list = %d sketches, want 0", len(list.Sketches))
	}
}

// TestSketch3DEnumerationAndStatus checks entity enumeration and the non-mutating
// constraint-status analysis over a 3D sketch of free points built in-model.
func TestSketch3DEnumerationAndStatus(t *testing.T) {
	r, s := emptyPartSession(t)
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	sk := part.Sketches3D().Add()
	sk.AddPoint3D(math.P3(0, 0, 0))
	sk.AddPoint3D(math.P3(3, 0, 0))

	var ents wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &ents)
	if len(ents.Entities) != 2 || ents.Entities[0].Kind != "point" {
		t.Fatalf("entities = %+v, want 2 points", ents.Entities)
	}
	if pt := ents.Entities[1].Points[0]; pt[0] != 3 || pt[1] != 0 || pt[2] != 0 {
		t.Errorf("second point = %v, want [3,0,0]", pt)
	}

	// No driving constraints yet ⇒ two free 3D points ⇒ 6 DOF, under-constrained.
	var status wire.ConstraintStatusResult
	call(t, r, s, "sketch3d.constraintStatus", `{"sketchIndex":0}`, &status)
	if status.DOF != 6 || status.Status != "under" {
		t.Fatalf("status = %+v, want DOF 6 / under", status)
	}

	var dims wire.ListDimensions3DResult
	call(t, r, s, "sketch3d.dimensions", `{"sketchIndex":0}`, &dims)
	if len(dims.Dimensions) != 0 {
		t.Fatalf("dimensions = %+v, want none", dims.Dimensions)
	}
}

// TestSketch3DAddEntities exercises the discriminated 3D entity constructor over the
// router: add a point, line, circle and arc, then enumerate them.
func TestSketch3DAddEntities(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	var pt wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"point","points":[[1,2,3]]}`, &pt)
	if pt.Kind != "point" || len(pt.PointIDs) != 1 {
		t.Fatalf("add point = %+v", pt)
	}

	var line wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[3,0,4]]}`, &line)
	if line.Kind != "line" || len(line.PointIDs) != 2 {
		t.Fatalf("add line = %+v", line)
	}

	var circle wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0,0]],"radius":"10 mm"}`, &circle)
	if circle.Kind != "circle" {
		t.Fatalf("add circle = %+v", circle)
	}

	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"arc","points":[[0,0,0],[1,0,0],[0,1,0]],"ccw":true}`, &wire.AddSketch3DEntityResult{})

	var ents wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &ents)
	if len(ents.Entities) != 4 {
		t.Fatalf("entities = %d, want 4 (point/line/circle/arc)", len(ents.Entities))
	}
	if k := kindAt(ents.Entities, 2); k != "circle" {
		t.Errorf("entity 2 kind = %q, want circle", k)
	}
	// The circle's radius enumerates in cm (10 mm = 1 cm).
	if r := ents.Entities[2].Radius; r < 0.999 || r > 1.001 {
		t.Errorf("circle radius = %v cm, want ~1", r)
	}
}

// kindAt returns the kind of the entity at index i (or "" if out of range).
func kindAt(ents []wire.Sketch3DEntityInfo, i int) string {
	if i < 0 || i >= len(ents) {
		return ""
	}
	return ents[i].Kind
}

// TestSketch3DAddHelix exercises the helix constructor's four definition modes over the
// router and checks the resulting curve geometry.
func TestSketch3DAddHelix(t *testing.T) {
	r, s := emptyPartSession(t)
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	cases := []struct {
		name      string
		json      string
		wantTurns float64
		wantPitch float64
	}{
		{"pitchRevolution", `{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"pitchRevolution","pitch":"10 mm","revolutions":3}`, 3, 1.0},
		{"pitchHeight", `{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"pitchHeight","pitch":"10 mm","height":"30 mm"}`, 3, 1.0},
		{"revolutionHeight", `{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"revolutionHeight","revolutions":3,"height":"30 mm"}`, 3, 1.0},
		{"spiral", `{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"spiral","pitch":"2 mm","revolutions":5}`, 5, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var res wire.AddSketch3DEntityResult
			call(t, r, s, "sketch3d.addEntity", tc.json, &res)
			h := lastHelix(t, part)
			if h.Turns != tc.wantTurns {
				t.Errorf("%s turns = %v, want %v", tc.name, h.Turns, tc.wantTurns)
			}
			if stdmath.Abs(h.AxialPerTurn-tc.wantPitch) > 1e-9 {
				t.Errorf("%s pitch = %v cm, want %v", tc.name, h.AxialPerTurn, tc.wantPitch)
			}
		})
	}
}

// lastHelix returns the most-recently-added helix in the active part's first 3D sketch.
func lastHelix(t *testing.T, part *compdef.PartComponentDefinition) *sketch.HelicalCurve3D {
	t.Helper()
	ents := part.Sketches3D().Item(0).Entities()
	for i := len(ents) - 1; i >= 0; i-- {
		if h, ok := ents[i].(*sketch.HelicalCurve3D); ok {
			return h
		}
	}
	t.Fatal("no helix found")
	return nil
}

// TestSketch3DAddHelixErrors covers the helix mode/validation error paths.
func TestSketch3DAddHelixErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	bad := []string{
		`{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"bogus","revolutions":3}`,
		`{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"pitchHeight","pitch":"0 mm","height":"30 mm"}`,
		`{"sketchIndex":0,"kind":"helical","points":[[0,0,0]],"radius":"4 mm","mode":"pitchRevolution","pitch":"10 mm","revolutions":0}`,
	}
	for _, b := range bad {
		if _, err := r.Handle(s, "sketch3d.addEntity", []byte(b)); err == nil {
			t.Errorf("expected error for %s", b)
		}
	}
}

// TestSketch3DAddEntityErrors covers the malformed-input paths of the constructor.
func TestSketch3DAddEntityErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})

	if _, err := r.Handle(s, "sketch3d.addEntity", []byte(`{"sketchIndex":0,"kind":"line","points":[[0,0,0]]}`)); err == nil {
		t.Error("a line with one point should error")
	}
	if _, err := r.Handle(s, "sketch3d.addEntity", []byte(`{"sketchIndex":0,"kind":"bogus"}`)); err == nil {
		t.Error("an unknown kind should error")
	}
	if _, err := r.Handle(s, "sketch3d.addEntity", []byte(`{"sketchIndex":0,"kind":"circle","points":[[0,0,0]],"radius":"oops"}`)); err == nil {
		t.Error("a bad radius should error")
	}
}

// TestConstraint3DKindMapping covers the constraint/entity wire mappings directly (the
// wire path to add 3D constraints lands in M22-F05).
func TestConstraint3DKindMapping(t *testing.T) {
	a := sketch.NewPoint3D(math.P3(0, 0, 0))
	b := sketch.NewPoint3D(math.P3(1, 0, 0))
	c := sketch.NewPoint3D(math.P3(2, 0, 0))

	if kind, ids := constraint3DKind(sketch.NewCoincident3D(a, b)); kind != types.Geo3DCoincident || len(ids) != 2 {
		t.Errorf("coincident mapping = %q/%v", kind, ids)
	}
	if kind, ids := constraint3DKind(sketch.NewCollinear3D(a, b, c)); kind != types.Geo3DCollinear || len(ids) != 3 {
		t.Errorf("collinear mapping = %q/%v", kind, ids)
	}
	if kind, _ := constraint3DKind(sketch.NewConcentric3D(a, b)); kind != types.Geo3DConcentric {
		t.Errorf("concentric mapping = %q", kind)
	}

	if info := entity3DInfo(0, a); info.Kind != string(types.Sketch3DEntityPoint) || len(info.Points) != 1 {
		t.Errorf("entity3DInfo(point) = %+v", info)
	}
}
