// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/math"
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
