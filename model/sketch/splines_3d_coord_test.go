// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
)

// TestEquationCurve3DCylindricalEvaluation: (r,θ,z)=(2,t,1) traces a radius-2 circle at z=1 (#1846).
func TestEquationCurve3DCylindricalEvaluation(t *testing.T) {
	s := NewSketches3D().Add()
	e, err := s.AddEquationCurve3DIn(types.CoordinateSystemCylindrical, "2", "t", "1", 0, stdmath.Pi/2)
	if err != nil {
		t.Fatalf("AddEquationCurve3DIn: %v", err)
	}
	if e.CoordinateSystem() != types.CoordinateSystemCylindrical {
		t.Errorf("CoordinateSystem() = %v, want cylindrical", e.CoordinateSystem())
	}
	if got := e.At(0); !got.IsEqualTo(gmath.P3(2, 0, 1), 1e-9) {
		t.Errorf("at t=0 = %v, want (2,0,1)", got)
	}
	if got := e.At(stdmath.Pi / 2); !got.IsEqualTo(gmath.P3(0, 2, 1), 1e-9) {
		t.Errorf("at t=π/2 = %v, want (0,2,1)", got)
	}
}

// TestEquationCurve3DSphericalEvaluation: (r,θ,φ)=(1,0,t) sweeps the prime meridian from +Z to −Z
// through +X (#1846).
func TestEquationCurve3DSphericalEvaluation(t *testing.T) {
	s := NewSketches3D().Add()
	e, err := s.AddEquationCurve3DIn(types.CoordinateSystemSpherical, "1", "0", "t", 0, stdmath.Pi)
	if err != nil {
		t.Fatalf("AddEquationCurve3DIn: %v", err)
	}
	cases := []struct {
		t    float64
		want gmath.Point3
	}{
		{0, gmath.P3(0, 0, 1)},
		{stdmath.Pi / 2, gmath.P3(1, 0, 0)},
		{stdmath.Pi, gmath.P3(0, 0, -1)},
	}
	for _, c := range cases {
		if got := e.At(c.t); !got.IsEqualTo(c.want, 1e-9) {
			t.Errorf("spherical at t=%g = %v, want %v", c.t, got, c.want)
		}
	}
}

// TestEquationCurve3DDefaultIsCartesian: the plain factory keeps x/y/z (#1846 AC2).
func TestEquationCurve3DDefaultIsCartesian(t *testing.T) {
	s := NewSketches3D().Add()
	e, _ := s.AddEquationCurve3D("t", "2", "3", 0, 1)
	if e.CoordinateSystem() != types.CoordinateSystemCartesian {
		t.Errorf("default coordinate system = %v, want cartesian", e.CoordinateSystem())
	}
	if got := e.At(0.5); !got.IsEqualTo(gmath.P3(0.5, 2, 3), 1e-9) {
		t.Errorf("cartesian at t=0.5 = %v, want (0.5,2,3)", got)
	}
}

// TestEquationCurve3DCoordinateSystemRoundTrips: the coordinate system survives marshal→apply so a
// saved cylindrical/spherical curve reloads with the same geometry (#1846 AC3).
func TestEquationCurve3DCoordinateSystemRoundTrips(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	if _, err := s.AddEquationCurve3DIn(types.CoordinateSystemSpherical, "1", "0", "t", 0, stdmath.Pi); err != nil {
		t.Fatalf("AddEquationCurve3DIn: %v", err)
	}

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewSketches3D()
	if err := dst.ApplyRecipe3D(data); err != nil {
		t.Fatalf("apply: %v", err)
	}
	ents := dst.Item(0).Entities()
	eq, ok := ents[0].(*EquationCurve3D)
	if !ok || eq.CoordinateSystem() != types.CoordinateSystemSpherical {
		t.Fatalf("restored entity = %+v, want a spherical equation curve", ents[0])
	}
	if got := eq.At(stdmath.Pi / 2); !got.IsEqualTo(gmath.P3(1, 0, 0), 1e-9) {
		t.Errorf("restored spherical curve at t=π/2 = %v, want (1,0,0)", got)
	}
}
