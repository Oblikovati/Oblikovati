// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestSurfacesCoincideProvesIdenticalPrimitives: each analytic primitive is proven coincident with a
// differently-anchored copy of ITSELF, and refused against a copy that differs in any one parameter.
// The pairs are the degenerate-overlap contacts the boolean must cover rather than intersect.
func TestSurfacesCoincideProvesIdenticalPrimitives(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(10)
	pl, _ := NewPlane(math.P3(0, 0, 2), math.V3(0, 0, 1))
	plSame, _ := NewPlane(math.P3(5, 5, 2), math.V3(0, 0, -1)) // same plane, other anchor and sense
	plOff, _ := NewPlane(math.P3(0, 0, 2.5), math.V3(0, 0, 1))
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	cylSame, _ := NewCylinder(math.P3(0, 0, 9), math.V3(0, 0, 1), 2) // same axis line, other v origin
	cylWide, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2.5)
	cylShift, _ := NewCylinder(math.P3(1, 0, 0), math.V3(0, 0, 1), 2)
	cone, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.4)
	coneSame, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.4)
	coneOther, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, -1), 0.4) // the other nappe
	sph, _ := NewSphere(math.P3(1, 2, 3), 4)
	sphSame, _ := NewSphere(math.P3(1, 2, 3), 4)
	sphBig, _ := NewSphere(math.P3(1, 2, 3), 4.5)

	for _, tc := range []struct {
		name string
		a, b Surface
		want bool
	}{
		{"plane with itself, other anchor and sense", pl, plSame, true},
		{"plane offset along its normal", pl, plOff, false},
		{"cylinder with itself, other v origin", cyl, cylSame, true},
		{"cylinder of another radius", cyl, cylWide, false},
		{"cylinder on a parallel axis", cyl, cylShift, false},
		{"cone with itself", cone, coneSame, true},
		{"the opposite nappe", cone, coneOther, false},
		{"sphere with itself", sph, sphSame, true},
		{"sphere of another radius", sph, sphBig, false},
		{"a plane and a cylinder", pl, cyl, false},
	} {
		if got := SurfacesCoincide(tc.a, tc.b, res); got != tc.want {
			t.Errorf("%s: SurfacesCoincide = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestSurfacesCoincideIsSymmetric: identity is symmetric, and the predicate must be too, or which
// operand the boolean asks about would change the answer.
func TestSurfacesCoincideIsSymmetric(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(10)
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	other, _ := NewCylinder(math.P3(0, 0, 4), math.V3(0, 0, -1), 2)
	if SurfacesCoincide(cyl, other, res) != SurfacesCoincide(other, cyl, res) {
		t.Error("SurfacesCoincide disagrees with itself when the operands swap")
	}
	tor := Torus{Center: math.P3(0, 0, 0), AxisDir: math.V3(0, 0, 1).AsUnit(), Ref: math.V3(1, 0, 0).AsUnit(), MajorRadius: 5, MinorRadius: 2}
	torFlip := Torus{Center: math.P3(0, 0, 0), AxisDir: math.V3(0, 0, -1).AsUnit(), Ref: math.V3(1, 0, 0).AsUnit(), MajorRadius: 5, MinorRadius: 2}
	if !SurfacesCoincide(tor, torFlip, res) {
		t.Error("a torus is not coincident with itself about the reversed axis; it is symmetric in its own plane")
	}
}

// TestSurfaceNormalAtReadsTheOutwardSense pins the datum the ON/ON tie-break reads: two coincident
// cylinders wound the same way agree in normal at a shared point, and a plane pair anchored the
// opposite way opposes.
func TestSurfaceNormalAtReadsTheOutwardSense(t *testing.T) {
	t.Parallel()
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	cylSame, _ := NewCylinder(math.P3(0, 0, 9), math.V3(0, 0, 1), 2)
	at := math.P3(2, 0, 1)
	if d := float64(SurfaceNormalAt(cyl, at).Dot(SurfaceNormalAt(cylSame, at))); d <= 0 {
		t.Errorf("two coincident cylinders' normals dot to %v at a shared point; want agreement", d)
	}
	pl, _ := NewPlane(math.P3(0, 0, 2), math.V3(0, 0, 1))
	plFlip, _ := NewPlane(math.P3(0, 0, 2), math.V3(0, 0, -1))
	if d := float64(SurfaceNormalAt(pl, at).Dot(SurfaceNormalAt(plFlip, at))); d >= 0 {
		t.Errorf("two oppositely wound planes' normals dot to %v; want opposition", d)
	}
	if stdmath.Abs(float64(SurfaceNormalAt(cyl, at).Length())-1) > 1e-12 {
		t.Error("SurfaceNormalAt does not return a unit vector")
	}
}
