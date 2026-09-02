// SPDX-License-Identifier: GPL-2.0-only

package geom_test

import (
	"math"
	"testing"

	"oblikovati.org/kernel/geom"
	m "oblikovati.org/math"
)

// circleParams2d is the unit-frame parametric form of a circle of radius r about c.
func circleParams2d(c m.Point2, r float64) geom.EllipticalParams2d {
	return geom.EllipticalParams2d{Center: c, U: m.V2(1, 0), V: m.V2(0, 1), A: r, B: r}
}

// circleImplicit2d is the same circle as a quadratic form: x²+y²−2cx·x−2cy·y+(c²−r²).
func circleImplicit2d(t *testing.T, c m.Point2, r float64) geom.Conic2dImplicit {
	t.Helper()
	f, ok := geom.ImplicitConic2dOf(c, m.V2(1, 0), m.V2(0, 1), r, r, false)
	if !ok {
		t.Fatalf("ImplicitConic2dOf(circle r=%g) declined", r)
	}
	return f
}

// residualsOn reports the largest |q(P(t))| over the returned parameters — zero when every root
// really is a crossing. It is the check that matters: a root that does not lie on BOTH conics is not
// an intersection however plausible its value.
func residualsOn(p geom.EllipticalParams2d, q geom.Conic2dImplicit, ts []float64) float64 {
	worst := 0.0
	for _, t := range ts {
		pt := p.PointAt(t)
		worst = math.Max(worst, math.Abs(q.Value(float64(pt.X), float64(pt.Y))))
	}
	return worst
}

// TestTwoCirclesMeetTwice is the simplest conic pair with a known closed form: circles of radius 5
// centred 6 apart meet at x=3, y=±4.
func TestTwoCirclesMeetTwice(t *testing.T) {
	t.Parallel()
	p := circleParams2d(m.P2(0, 0), 5)
	q := circleImplicit2d(t, m.P2(6, 0), 5)
	ts, inf := geom.IntersectConic2d(p, q)
	if inf {
		t.Fatal("two distinct circles are not the same curve")
	}
	if len(ts) != 2 {
		t.Fatalf("got %d crossings %v, want 2", len(ts), ts)
	}
	for _, want := range []m.Point2{m.P2(3, 4), m.P2(3, -4)} {
		found := false
		for _, tt := range ts {
			if p.PointAt(tt).DistanceTo(want) < 1e-9 {
				found = true
			}
		}
		if !found {
			t.Errorf("crossing %v missing from %v", want, geom.ConicPointsAt(p, ts))
		}
	}
}

// TestTangentCirclesMeetOnce keeps a tangency ONE contact rather than two crossings a hair apart —
// the distinction a caller walking the roots as interval boundaries depends on.
func TestTangentCirclesMeetOnce(t *testing.T) {
	t.Parallel()
	p := circleParams2d(m.P2(0, 0), 5)
	q := circleImplicit2d(t, m.P2(10, 0), 5)
	ts, _ := geom.IntersectConic2d(p, q)
	if len(ts) != 1 {
		t.Fatalf("got %d contacts %v, want 1 (externally tangent circles touch once)", len(ts), ts)
	}
	if got := p.PointAt(ts[0]); got.DistanceTo(m.P2(5, 0)) > 1e-7 {
		t.Errorf("tangency at %v, want (5, 0)", got)
	}
}

// TestSeparateCirclesMeetNever pins the empty answer, which must be empty rather than a pair of
// near-complex roots leaking through the real filter.
func TestSeparateCirclesMeetNever(t *testing.T) {
	t.Parallel()
	p := circleParams2d(m.P2(0, 0), 1)
	q := circleImplicit2d(t, m.P2(10, 0), 1)
	if ts, _ := geom.IntersectConic2d(p, q); len(ts) != 0 {
		t.Errorf("disjoint circles reported %d crossings %v", len(ts), ts)
	}
}

// TestEllipseMeetsRotatedEllipse is the case no type-pair table would carry: two ellipses at an
// angle, four crossings, checked by residual on both forms rather than against hand-computed points.
func TestEllipseMeetsRotatedEllipse(t *testing.T) {
	t.Parallel()
	p := geom.EllipticalParams2d{Center: m.P2(0, 0), U: m.V2(1, 0), V: m.V2(0, 1), A: 4, B: 2}
	s, c := math.Sqrt2/2, math.Sqrt2/2
	q, ok := geom.ImplicitConic2dOf(m.P2(0, 0), m.V2(m.Scalar(c), m.Scalar(s)), m.V2(m.Scalar(-s), m.Scalar(c)), 4, 2, false)
	if !ok {
		t.Fatal("ImplicitConic2dOf declined a rotated ellipse")
	}
	ts, inf := geom.IntersectConic2d(p, q)
	if inf {
		t.Fatal("two ellipses at 45° are not the same curve")
	}
	if len(ts) != 4 {
		t.Fatalf("got %d crossings %v, want 4", len(ts), geom.ConicPointsAt(p, ts))
	}
	if r := residualsOn(p, q, ts); r > 1e-9 {
		t.Errorf("worst residual %g: a root must lie on BOTH conics", r)
	}
}

// TestLineMeetsEllipse checks the degenerate conic goes through the same substitution, which is what
// keeps a line from needing its own intersector.
func TestLineMeetsEllipse(t *testing.T) {
	t.Parallel()
	p := geom.EllipticalParams2d{Center: m.P2(0, 0), U: m.V2(1, 0), V: m.V2(0, 1), A: 4, B: 2}
	q, ok := geom.ImplicitLine2dOf(m.P2(0, 0), m.V2(0, 1)) // the y axis
	if !ok {
		t.Fatal("ImplicitLine2dOf declined a unit direction")
	}
	ts, _ := geom.IntersectConic2d(p, q)
	if len(ts) != 2 {
		t.Fatalf("got %d crossings %v, want 2", len(ts), geom.ConicPointsAt(p, ts))
	}
	for _, tt := range ts {
		if x := math.Abs(float64(p.PointAt(tt).X)); x > 1e-9 {
			t.Errorf("crossing at x=%g, want 0 (the y axis)", x)
		}
	}
}

// TestCoincidentConicsReportInfinite pins the answer no finite root set can carry. A caller that read
// it as "no crossings" would treat two identical curves as disjoint.
func TestCoincidentConicsReportInfinite(t *testing.T) {
	t.Parallel()
	p := circleParams2d(m.P2(1, 2), 3)
	q := circleImplicit2d(t, m.P2(1, 2), 3)
	if _, inf := geom.IntersectConic2d(p, q); !inf {
		t.Error("a circle against itself must report infinite solutions, not a root list")
	}
}

// TestHyperbolaBranchMeetsCircle exercises the exponential substitution: the branch x²−y²=1 (x>0)
// crossing a circle of radius 2 about the origin, at x=√2.5, y=±√1.5.
func TestHyperbolaBranchMeetsCircle(t *testing.T) {
	t.Parallel()
	p := geom.EllipticalParams2d{
		Center: m.P2(0, 0), U: m.V2(1, 0), V: m.V2(0, 1), A: 1, B: 1, Hyperbolic: true,
	}
	q := circleImplicit2d(t, m.P2(0, 0), 2)
	ts, _ := geom.IntersectConic2d(p, q)
	if len(ts) != 2 {
		t.Fatalf("got %d crossings %v, want 2", len(ts), geom.ConicPointsAt(p, ts))
	}
	if r := residualsOn(p, q, ts); r > 1e-9 {
		t.Errorf("worst residual %g on the hyperbola branch", r)
	}
	for _, tt := range ts {
		if x := float64(p.PointAt(tt).X); math.Abs(x-math.Sqrt(2.5)) > 1e-9 {
			t.Errorf("crossing at x=%g, want √2.5", x)
		}
	}
}
