// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/health"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

func TestCombineJoinsTwoBodiesForReal(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	// Two disjoint prisms in the running state, then Combine them.
	NewBaseFeatures(fs).AddBase(buildPrism(squarePoly(0), sketch.XYPlane(), span{near: 0, far: 1}, 0, "a"))
	NewBaseFeatures(fs).AddBase(buildPrism(squarePoly(10), sketch.XYPlane(), span{near: 0, far: 1}, 0, "b"))
	combine := NewModifyFeatures(fs).AddCombine(0, 1, ops.Join)
	fs.Recompute()

	if !combine.Health().OK() {
		t.Fatalf("combine sick: %+v", combine.Health())
	}
	// The two prisms are joined into one body (12 faces); 1 body remains.
	if len(fs.Result()) != 1 || len(fs.Result()[0].Faces()) != 12 {
		t.Errorf("combine result = %d bodies; want 1 with 12 faces", len(fs.Result()))
	}
	if ops.Validate(fs.Result()[0]).Manifold == false {
		t.Error("combined body should be manifold")
	}
}

func TestCombineRejectsBadIndices(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(prismBody())
	bad := NewModifyFeatures(fs).AddCombine(0, 5, ops.Join) // tool index out of range
	fs.Recompute()
	if bad.Health().Status != health.Sick {
		t.Errorf("combine with bad indices = %v, want sick", bad.Health().Status)
	}
}

func TestDirectEditsResolveThenDefer(t *testing.T) {
	body := prismBody()
	face := body.Faces()[0].ReferenceKey()
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(body)
	mod := NewModifyFeatures(fs)
	feats := map[string]*PartFeature{
		"split":        mod.AddSplit([][]byte{face}),
		"move-face":    mod.AddMoveFace([][]byte{face}),
		"face-offset":  mod.AddFaceOffset([][]byte{face}),
		"delete-face":  mod.AddDeleteFace([][]byte{face}),
		"replace-face": mod.AddReplaceFace([][]byte{face}),
		"thicken":      mod.AddThicken([][]byte{face}),
	}
	fs.Recompute()
	for name, pf := range feats {
		if pf.Health().Status != health.Warning {
			t.Errorf("%s = %v, want warning (resolved + deferred)", name, pf.Health().Status)
		}
		if pf.Kind() != name {
			t.Errorf("kind = %q, want %q", pf.Kind(), name)
		}
	}
}

// squarePoly returns a unit square offset by dx; planeXY is the XY sketch plane.
func squarePoly(dx float64) []math.Point2 {
	return []math.Point2{{X: dx, Y: 0}, {X: dx + 1, Y: 0}, {X: dx + 1, Y: 1}, {X: dx, Y: 1}}
}

func TestCombineDefinitionAccessible(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	c := NewModifyFeatures(fs).AddCombine(0, 1, ops.Cut)
	if c.Definition().(*CombineFeature).Definition().Operation != ops.Cut {
		t.Error("combine definition not accessible")
	}
}
