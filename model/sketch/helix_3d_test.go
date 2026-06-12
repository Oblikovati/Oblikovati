// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	"oblikovati.org/kernel/geom"

	gmath "oblikovati.org/math"
)

// TestHelicalCurve3D checks a helix's DOFs (origin + radius), kernel curve, and total
// height.
func TestHelicalCurve3D(t *testing.T) {
	s := NewSketches3D().Add()
	h := s.AddHelix3D(gmath.P3(0, 0, 0), zAxis(), 4, 10, 0, 3, false)

	// origin (3) + start radius (1) = 4 DOF.
	if s.DegreesOfFreedom() != 4 {
		t.Errorf("a free helix has 4 DOF, got %d", s.DegreesOfFreedom())
	}
	if h.Height() != 30 {
		t.Errorf("Height = %v, want 30 (pitch 10 × 3 turns)", h.Height())
	}
	cu, err := h.Curve()
	if err != nil {
		t.Fatalf("Curve: %v", err)
	}
	want := math.Hypot(2*math.Pi*4, 10) * 3
	helix, isAnalytic := cu.(geom.Helix3d)
	if !isAnalytic {
		t.Fatalf("a natural constant helix must stay analytic, got %T", cu)
	}
	if l := helix.Length(); math.Abs(l-want) > 1e-9 {
		t.Errorf("kernel helix length = %v, want %v", l, want)
	}
	// The start radius is a solver DOF the helix contributes.
	if dofs := h.scalarDOFs(); len(dofs) != 1 || dofs[0] != &h.StartRadius {
		t.Error("helix should contribute its start radius as a scalar DOF")
	}
}

// TestHelicalCurve3DRoundTrip checks a helix survives marshal→apply with its shape intact.
func TestHelicalCurve3DRoundTrip(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	s.AddHelix3D(gmath.P3(1, 2, 3), zAxis(), 5, 8, 1.5, 4, true)

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewSketches3D()
	if err := dst.ApplyRecipe3D(data); err != nil {
		t.Fatalf("apply: %v", err)
	}
	ents := dst.Item(0).Entities()
	h, ok := ents[0].(*HelicalCurve3D)
	if !ok {
		t.Fatalf("restored entity is %T, want *HelicalCurve3D", ents[0])
	}
	if float64(h.StartRadius) != 5 || h.AxialPerTurn != 8 || h.RadialPerTurn != 1.5 || h.Turns != 4 || !h.Clockwise {
		t.Errorf("restored helix shape mismatch: %+v", h)
	}
	if h.Origin.Position() != gmath.P3(1, 2, 3) {
		t.Errorf("restored helix origin = %v, want (1,2,3)", h.Origin.Position())
	}
}

// TestPerpendicularTo checks the angle-0 reference is perpendicular to the axis for both
// the general case and the near-X-axis fallback.
func TestPerpendicularTo(t *testing.T) {
	for _, axis := range [][3]float64{{0, 0, 1}, {1, 0, 0}, {-1, 0, 0}, {0.9, 0.1, 0.42}} {
		u, _ := gmath.NewUnitVector3(gmath.Scalar(axis[0]), gmath.Scalar(axis[1]), gmath.Scalar(axis[2]))
		perp := perpendicularTo(u)
		if math.Abs(float64(perp.Dot(u.AsVector()))) > 1e-9 {
			t.Errorf("perpendicularTo(%v) = %v not perpendicular", axis, perp)
		}
		if float64(perp.Length()) < 1e-9 {
			t.Errorf("perpendicularTo(%v) is degenerate", axis)
		}
	}
}
