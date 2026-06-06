// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

func xAxis() gmath.UnitVector3 {
	u, _ := gmath.NewUnitVector3(1, 0, 0)
	return u
}

// TestEllipse3D checks a 3D ellipse's two radius DOFs and its kernel curve.
func TestEllipse3D(t *testing.T) {
	s := NewSketches3D().Add()
	e := s.AddEllipse3D(gmath.P3(0, 0, 0), zAxis(), xAxis(), 5, 3)

	// center (3) + major + minor (2) = 5 DOF.
	if s.DegreesOfFreedom() != 5 {
		t.Errorf("a free 3D ellipse has 5 DOF, got %d", s.DegreesOfFreedom())
	}
	if dofs := e.scalarDOFs(); len(dofs) != 2 {
		t.Errorf("ellipse should contribute 2 scalar DOFs, got %d", len(dofs))
	}
	cu, err := e.Curve()
	if err != nil {
		t.Fatalf("Curve: %v", err)
	}
	if cu.MajorRadius != 5 || cu.MinorRadius != 3 {
		t.Errorf("kernel ellipse radii = %v/%v, want 5/3", cu.MajorRadius, cu.MinorRadius)
	}
	// The major-axis endpoint sits at center + majorR along +X.
	if p := cu.PointAt(0); p.DistanceTo(gmath.P3(5, 0, 0)) > 1e-9 {
		t.Errorf("ellipse PointAt(0) = %v, want (5,0,0)", p)
	}
}

// TestEllipticalArc3D checks a quarter elliptical arc's kernel curve endpoints.
func TestEllipticalArc3D(t *testing.T) {
	s := NewSketches3D().Add()
	e := s.AddEllipticalArc3D(gmath.P3(0, 0, 0), zAxis(), xAxis(), 4, 2, 0, math.Pi/2)

	cu, err := e.Curve()
	if err != nil {
		t.Fatalf("Curve: %v", err)
	}
	if start := cu.PointAt(0); start.DistanceTo(gmath.P3(4, 0, 0)) > 1e-9 {
		t.Errorf("arc start = %v, want (4,0,0)", start)
	}
	if end := cu.PointAt(1); end.DistanceTo(gmath.P3(0, 2, 0)) > 1e-9 {
		t.Errorf("arc end = %v, want (0,2,0)", end)
	}
	if len(e.scalarDOFs()) != 2 {
		t.Error("elliptical arc should contribute 2 scalar DOFs")
	}
}

// TestConics3DRoundTrip checks ellipse + elliptical arc survive marshal→apply.
func TestConics3DRoundTrip(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	s.AddEllipse3D(gmath.P3(1, 0, 0), zAxis(), xAxis(), 5, 3)
	s.AddEllipticalArc3D(gmath.P3(0, 0, 0), zAxis(), xAxis(), 4, 2, 0.2, 1.1)

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewSketches3D()
	if err := dst.ApplyRecipe3D(data); err != nil {
		t.Fatalf("apply: %v", err)
	}
	ents := dst.Item(0).Entities()
	el, ok := ents[0].(*Ellipse3D)
	if !ok || float64(el.MajorRadius) != 5 || float64(el.MinorRadius) != 3 {
		t.Errorf("restored ellipse wrong: %+v", ents[0])
	}
	ea, ok := ents[1].(*EllipticalArc3D)
	if !ok || ea.SweepAngle != 1.1 || ea.StartAngle != 0.2 {
		t.Errorf("restored elliptical arc wrong: %+v", ents[1])
	}
}
