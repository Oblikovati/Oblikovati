// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestTangentBoundIsUpperBound is the #1608 certification guard for the analytic surfaces: each
// closed-form tangentBoundOverBox must dominate the ACTUAL partial magnitudes everywhere in the box,
// or the SSI prune could discard a cell that holds a crossing. It samples DerivativesAt on a lattice
// and fails if any sample exceeds the claimed bound.
func TestTangentBoundIsUpperBound(t *testing.T) {
	t.Parallel()
	plane, _ := NewPlane(math.P3(1, 2, 3), math.V3(0, 0, 1))
	cyl, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 4)
	cone, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.6)
	sphere, _ := NewSphere(math.P3(0, 0, 0), 5)
	torus, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 10, 3)
	cases := []struct {
		name           string
		s              boxTangentBounder
		u0, u1, v0, v1 float64
	}{
		{"plane", plane, -3, 3, -3, 3},
		{"cylinder", cyl, 0, 2 * stdmath.Pi, -5, 5},
		{"cone", cone, 0, 2 * stdmath.Pi, 0, 8},
		{"sphere", sphere, 0, 2 * stdmath.Pi, -stdmath.Pi / 2, stdmath.Pi / 2},
		{"torus", torus, 0, 2 * stdmath.Pi, 0, 2 * stdmath.Pi},
	}
	for _, c := range cases {
		assertTangentBound(t, c.name, c.s, c.u0, c.u1, c.v0, c.v1)
	}
}

// assertTangentBound samples the surface's partials over the box and checks they never exceed its bound.
func assertTangentBound(t *testing.T, name string, b boxTangentBounder, u0, u1, v0, v1 float64) {
	t.Helper()
	su, sv, ok := b.tangentBoundOverBox(u0, u1, v0, v1)
	if !ok {
		t.Fatalf("%s: analytic surface must offer a certified bound", name)
	}
	s := b.(Surface)
	for i := 0; i <= 20; i++ {
		for j := 0; j <= 20; j++ {
			u := u0 + (u1-u0)*float64(i)/20
			v := v0 + (v1-v0)*float64(j)/20
			du, dv := s.DerivativesAt(u, v)
			if float64(du.Length()) > su+1e-9 || float64(dv.Length()) > sv+1e-9 {
				t.Fatalf("%s: bound (%.4f,%.4f) below tangent (%.4f,%.4f) at (%.3f,%.3f)", name, su, sv, du.Length(), dv.Length(), u, v)
			}
		}
	}
}

// TestRationalSurfaceDeclinesHodograph guards issue #1608 point 5: a rational NURBS has no simple
// control-net hodograph, so it must DECLINE (ok=false) — the SSI seeder then records the fallback
// rather than pruning on a bound it cannot certify.
func TestRationalSurfaceDeclinesHodograph(t *testing.T) {
	t.Parallel()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 1, 0)},
		{math.P3(1, 0, 0), math.P3(1, 1, 1)},
	}
	weights := [][]float64{{1, 2}, {1, 1}} // a non-unit weight ⇒ rational
	knots := []float64{0, 0, 1, 1}
	s, err := NewBSplineSurface(1, 1, ctrl, weights, knots, knots)
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	if _, _, ok := s.tangentBoundOverBox(0, 1, 0, 1); ok {
		t.Error("rational surface reported a certified hodograph bound; want decline")
	}
}
