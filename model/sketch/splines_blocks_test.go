// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati/math"
)

func TestSplineEditsViaPoints(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	fit := s.Splines().AddByPoints([]math.Point2{math.P2(0, 0), math.P2(1, 2), math.P2(3, 1)}, false)
	if !fit.IsFitType() || fit.PointCount() != 3 || fit.Closed {
		t.Fatalf("fit spline wrong: fit=%v n=%d closed=%v", fit.IsFitType(), fit.PointCount(), fit.Closed)
	}
	// Editing a defining point moves the spline.
	fit.Points[1].SetPosition(math.P2(1, 9))
	if !fit.Points[1].Position().IsEqualTo(math.P2(1, 9), tol) {
		t.Error("spline point edit did not take")
	}
	ctrl := s.Splines().AddByControlPoints([]math.Point2{math.P2(0, 0), math.P2(2, 0)}, true)
	if ctrl.IsFitType() || !ctrl.Closed {
		t.Error("control-point spline flags wrong")
	}
	if s.Splines().Count() != 2 || s.Splines().Item(0) != fit {
		t.Error("spline collection tracking wrong")
	}
	// Each spline point is a solver variable.
	if len(s.AllPoints()) != 5 { // 3 + 2
		t.Errorf("AllPoints = %d, want 5", len(s.AllPoints()))
	}
}

func TestBlockInstancesAndUpdatesWithDefinition(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	def := s.Blocks().DefineBlock("bolt")
	// A scratch sketch supplies entities to put in the definition.
	scratch := NewSketches().Add(XYPlane())
	def.Add(scratch.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0)))
	def.Add(scratch.Circles().AddByCenterRadius(math.P2(0, 0), 0.5))

	inst := s.Blocks().Insert(def, math.Translation3(math.V2(10, 0)))
	if s.Blocks().DefinitionCount() != 1 || s.Blocks().InstanceCount() != 1 {
		t.Fatal("block collection counts wrong")
	}
	if inst.Definition() != def || inst.EntityCount() != 2 {
		t.Fatalf("instance not bound to definition: count=%d", inst.EntityCount())
	}
	if !inst.Transform().TransformPoint(math.P2(0, 0)).IsEqualTo(math.P2(10, 0), tol) {
		t.Error("instance transform wrong")
	}

	// Editing the definition is reflected live in the instance.
	def.Add(scratch.Lines().AddByTwoPoints(math.P2(1, 0), math.P2(1, 1)))
	if inst.EntityCount() != 3 {
		t.Errorf("instance entity count = %d after editing definition, want 3", inst.EntityCount())
	}
	if def.Name() != "bolt" || def.ID() == 0 || def.EntityCount() != 3 {
		t.Error("definition accessors wrong")
	}
	// The instance is a sketch entity.
	if s.EntityCount() != 1 {
		t.Errorf("sketch entity count = %d, want 1 (the instance)", s.EntityCount())
	}

	inst.SetTransform(math.Identity3())
	if !inst.Transform().IsEqualTo(math.Identity3(), tol) {
		t.Error("SetTransform ignored")
	}
}
