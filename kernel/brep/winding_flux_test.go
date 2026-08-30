// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// fluxTotal sums a prepared query's signed per-face solid angle at p — the generalized winding number
// times 4π. A test helper so the winding QUANTITY (not just the boolean verdict) can be asserted.
func fluxTotal(q *fluxQuery, p math.Point3) float64 {
	total := 0.0
	for i := range q.faces {
		f := &q.faces[i]
		total += f.sign * integrateFluxCell(f.cf.surface, p, f.polys, f.u0, f.u1, f.v0, f.v1, 0)
	}
	return total
}

// TestFluxWindingNumberOnSphere pins the flux to the generalized winding number: ≈4π for an interior
// point, ≈0 for an exterior one, ≈2π on the surface. The coarse-quadrature tolerance is wide because the
// classifier only needs the sign of (total − 2π), which has a full-π margin either side.
func TestFluxWindingNumberOnSphere(t *testing.T) {
	sph, err := SolidSphere(math.P3(0, 0, 0), 3, "sphere")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	q := newFluxQuery(facesOfAny(sph))
	cases := []struct {
		name   string
		p      math.Point3
		want4π float64
	}{
		{"deep interior", math.P3(0, 0, 0), 4 * stdmath.Pi},
		{"near-surface interior", math.P3(2.5, 0, 0), 4 * stdmath.Pi},
		{"far exterior", math.P3(10, 0, 0), 0},
		{"just outside", math.P3(3.5, 0, 0), 0},
	}
	for _, c := range cases {
		if got := fluxTotal(q, c.p); stdmath.Abs(got-c.want4π) > stdmath.Pi {
			t.Errorf("%s: flux total = %.3f (%.3f·4π), want ≈ %.3f", c.name, got, got/(4*stdmath.Pi), c.want4π)
		}
	}
}

// TestFluxQueryMatchesFreshBuild guards the prepared-query optimisation: repeatedly querying one prepared
// fluxQuery must give the same verdict as building a fresh query per point (the trim polylines are cached,
// nothing else may differ).
func TestFluxQueryMatchesFreshBuild(t *testing.T) {
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	faces := facesOfAny(cyl)
	box := cyl.RangeBox()
	prepared := newFluxQuery(faces)
	for _, p := range []math.Point3{{X: 0, Y: 0, Z: 2}, {X: 1.5, Y: 0, Z: 2}, {X: 3, Y: 0, Z: 2}, {X: 0, Y: 0, Z: 5}} {
		if prepared.inside(p, box) != newFluxQuery(faces).inside(p, box) {
			t.Errorf("prepared query disagrees with fresh build at %v", p)
		}
	}
}

// TestPointInLoops2DWithHole checks the even-odd (u, v) trim test on a square with a square hole: a point
// in the hole is OUTSIDE the trimmed region (two crossings), a point between hole and outer edge is inside.
func TestPointInLoops2DWithHole(t *testing.T) {
	outer := []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 10), math.P2(0, 10)}
	hole := []math.Point2{math.P2(4, 4), math.P2(6, 4), math.P2(6, 6), math.P2(4, 6)}
	polys := [][]math.Point2{outer, hole}
	cases := []struct {
		name string
		q    math.Point2
		want bool
	}{
		{"between hole and edge", math.P2(2, 5), true},
		{"inside the hole", math.P2(5, 5), false},
		{"outside the outer loop", math.P2(15, 5), false},
	}
	for _, c := range cases {
		if got := pointInLoops2D(polys, c.q); got != c.want {
			t.Errorf("%s: pointInLoops2D(%v) = %v, want %v", c.name, c.q, got, c.want)
		}
	}
}
