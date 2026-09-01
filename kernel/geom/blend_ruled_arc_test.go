// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestRuledArcBlendReproducesArcsExactly pins the exactness claim (#1606): the rational ruled
// blend's v=0 / v=1 isolines ARE the two circular arcs — sampled position error at machine
// precision, not chord error.
func TestRuledArcBlendReproducesArcsExactly(t *testing.T) {
	t.Parallel()
	c0, c1 := math.P3(0, 2, 2), math.P3(5, 3, 3) // centres walking a bisector as r grows
	const r0, r1, sweep = 2.0, 3.0, stdmath.Pi / 2
	x, y := math.V3(0, -1, 0), math.V3(0, 0, -1)
	s, err := NewRuledArcBlend(c0, r0, c1, r1, x, y, sweep)
	if err != nil {
		t.Fatalf("NewRuledArcBlend: %v", err)
	}
	for i := 0; i <= 16; i++ {
		u := float64(i) / 16
		theta := u * sweep
		d := x.Scale(stdmath.Cos(theta)).Add(y.Scale(stdmath.Sin(theta)))
		for _, end := range []struct {
			v float64
			c math.Point3
			r float64
		}{{0, c0, r0}, {1, c1, r1}} {
			// NOTE: the rational quadratic's parameter is not arc length, so compare by
			// DISTANCE-FROM-CENTRE and plane membership, which are parameterization-free.
			p := s.PointAt(u, end.v)
			if got := float64(end.c.DistanceTo(p)); stdmath.Abs(got-end.r) > 1e-12 {
				t.Fatalf("u=%g v=%g: |p-c| = %.15f, want %g (point off the circle)", u, end.v, got, end.r)
			}
			_ = d
			if got := float64(end.c.VectorTo(p).Dot(math.V3(1, 0, 0))); stdmath.Abs(got) > 1e-12 {
				t.Fatalf("u=%g v=%g: point off the profile plane by %g", u, end.v, got)
			}
		}
	}
}

// TestRuledArcBlendRunoutCollapsesToApex: r1=0 collapses the v=1 column to the apex — the exact
// oblique cone of a fillet run-out, still a valid surface.
func TestRuledArcBlendRunoutCollapsesToApex(t *testing.T) {
	t.Parallel()
	apex := math.P3(4, 0, 0)
	s, err := NewRuledArcBlend(math.P3(0, 1, 1), 1.5, apex, 0, math.V3(0, -1, 0), math.V3(0, 0, -1), stdmath.Pi/2)
	if err != nil {
		t.Fatalf("NewRuledArcBlend(runout): %v", err)
	}
	for i := 0; i <= 8; i++ {
		if d := float64(apex.DistanceTo(s.PointAt(float64(i)/8, 1))); d > 1e-12 {
			t.Fatalf("v=1 isoline point %g from the apex, want 0", d)
		}
	}
}

// TestRuledArcBlendIsG1AcrossSegments: a 3/4-turn sweep needs multiple rational segments; the
// surface normal must be continuous across the segment joins (the C0 strip creases this
// replaces were ~11° — assert machine-precision continuity).
func TestRuledArcBlendIsG1AcrossSegments(t *testing.T) {
	t.Parallel()
	const sweep = 3 * stdmath.Pi / 2
	s, err := NewRuledArcBlend(math.P3(0, 0, 0), 2, math.P3(6, 0.5, 0.5), 3, math.V3(0, 1, 0), math.V3(0, 0, 1), sweep)
	if err != nil {
		t.Fatalf("NewRuledArcBlend: %v", err)
	}
	segs := arcNurbsSegments(sweep)
	for k := 1; k < segs; k++ {
		u := float64(k) / float64(segs)
		nl := s.NormalAt(u-1e-9, 0.5)
		nr := s.NormalAt(u+1e-9, 0.5)
		if dot := nl.Dot(nr); dot < 1-1e-9 {
			t.Errorf("normal jump at segment join u=%g: cos=%.12f, want ~1 (G1)", u, dot)
		}
	}
}

// TestRuledSectionBlendMatchesConicBoundary: the blend's v=0 isoline equals the conic section
// curve point-for-point (same parameterization) — so a face boundary built from
// NewConicSectionCurve lies exactly on the surface.
func TestRuledSectionBlendMatchesConicBoundary(t *testing.T) {
	t.Parallel()
	sec0 := [3]math.Point3{math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0)}
	sec1 := [3]math.Point3{math.P3(2, 0, 3), math.P3(2, 2, 3), math.P3(0, 2, 3)}
	const w = 0.75
	s, err := NewRuledSectionBlend(sec0, sec1, w)
	if err != nil {
		t.Fatalf("NewRuledSectionBlend: %v", err)
	}
	c, err := NewConicSectionCurve(sec0[0], sec0[1], sec0[2], w)
	if err != nil {
		t.Fatalf("NewConicSectionCurve: %v", err)
	}
	for i := 0; i <= 10; i++ {
		u := float64(i) / 10
		if d := float64(c.PointAt(u).DistanceTo(s.PointAt(u, 0))); d > 1e-13 {
			t.Fatalf("u=%g: boundary curve %g off the surface isoline", u, d)
		}
	}
	// A degenerate (run-out) column still evaluates: every v=1 point is the apex.
	apex := math.P3(5, 5, 5)
	r, err := NewRuledSectionBlend(sec0, [3]math.Point3{apex, apex, apex}, w)
	if err != nil {
		t.Fatalf("NewRuledSectionBlend(runout): %v", err)
	}
	if d := float64(apex.DistanceTo(r.PointAt(0.3, 1))); d > 1e-13 {
		t.Fatalf("runout column point %g off the apex", d)
	}
}
