// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// White-box tests for the GATE-FREE core of the curved-rim carry (exactRetainedSpanOnParent,
// fillet_survivor_rim.go), which the rim rebuild reuses as its re-aimed host seam
// (wallSeamCurve, fillet_rim_build.go).
//
// The rim rebuild recedes a curved host and re-aims its axial SEAM edge at the new contact vertex. That
// seam bounds the host, so it must stay ON the host — and the curve that does so is the host's own seam
// curve, sub-spanned. Before this, the rebuild always rebuilt the seam as a straight line, which is the
// meridian only on a ruled host: J2's SPHERE shipped a 90.38-long chord 28.44 off itself, and J4's TORUS
// a 61.24-long chord 10.43 off itself (rimhost-carry-report.md).

// j2SphereMeridian is a stand-in for J2's host seam: the sphere's meridian great circle in the x/z plane,
// radius 50 about the origin, swept the quarter that the psphere -90..45 zone's seam covers.
func j2SphereMeridian() geom.Arc3d {
	return geom.Arc3d{
		Center: math.P3(0, 0, 0), Normal: math.V3(0, -1, 0).AsUnit(), RefDir: math.V3(0, 0, -1).AsUnit(),
		Radius: 50, StartAngle: 0, SweepAngle: 3 * stdmath.Pi / 4,
	}
}

// TestExactRetainedSpanCarriesAnArcMeridian pins the circular arm: a span whose endpoints lie ON the parent
// is returned as the parent's OWN sub-arc, running from→to in the caller's order, with every interior
// station on the parent circle — and NOT on the chord, which is the whole point.
func TestExactRetainedSpanCarriesAnArcMeridian(t *testing.T) {
	t.Parallel()
	parent := j2SphereMeridian()
	from, to := parent.PointAt(0.1), parent.PointAt(0.9)
	sub := exactRetainedSpanOnParent(parent, from, to)
	arc, ok := sub.(geom.Arc3d)
	if !ok {
		t.Fatalf("carried curve is %T, want geom.Arc3d (an on-parent span must not degrade to a chord)", sub)
	}
	lo, hi := arc.Domain()
	if d := float64(arc.PointAt(lo).DistanceTo(from)); d > 1e-12 {
		t.Errorf("sub-arc starts %.6g from the span's own start point, want ≤1e-12 (it must run from→to)", d)
	}
	if d := float64(arc.PointAt(hi).DistanceTo(to)); d > 1e-12 {
		t.Errorf("sub-arc ends %.6g from the span's own end point, want ≤1e-12", d)
	}
	chord := geom.NewLineSegment(from, to)
	offChord := 0.0
	for i := 0; i <= 16; i++ {
		p := arc.PointAt(lo + (hi-lo)*float64(i)/16)
		if d := stdmath.Abs(float64(parent.Center.DistanceTo(p)) - parent.Radius); d > 1e-9 {
			t.Errorf("station %d sits %.6g off the parent circle, want ≤1e-9", i, d)
		}
		if d := float64(p.DistanceTo(chord.PointAt(float64(i) / 16))); d > offChord {
			offChord = d
		}
	}
	if offChord < 1 {
		t.Errorf("the carried span is within %.6g of its own chord — it is not actually curved", offChord)
	}
}

// TestExactRetainedSpanDeclinesAnOffParentEndpoint pins the exactness gate: an endpoint pulled OFF the
// parent means the parent's sub-span is NOT the boundary, so nil (the caller's own fallback) is returned
// rather than a guess.
func TestExactRetainedSpanDeclinesAnOffParentEndpoint(t *testing.T) {
	t.Parallel()
	parent := j2SphereMeridian()
	from := parent.PointAt(0.1)
	off := parent.PointAt(0.9).TranslateBy(math.V3(0, 0, 5)) // 5 off the meridian circle
	if got := exactRetainedSpanOnParent(parent, from, off); got != nil {
		t.Errorf("carried %T for an OFF-parent endpoint, want nil", got)
	}
}

// TestExactRetainedSpanDeclinesAStraightParent is the byte-identity half of the host-seam fix: a cylinder,
// cone or elliptical-cylinder host's seam parent IS a straight ruling, so the carry must decline and leave
// the rebuild shipping the very LineSegment it always shipped. Every rim-fillet green but J2/J4 depends on
// this (I9 J1 K1 R8 U6 W6 W8 W9 Z1 J6 J8, bfuseblend A1 A2 A7 B1).
func TestExactRetainedSpanDeclinesAStraightParent(t *testing.T) {
	t.Parallel()
	line := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(0, 0, 100))
	if got := exactRetainedSpanOnParent(line, math.P3(0, 0, 10), math.P3(0, 0, 90)); got != nil {
		t.Errorf("carried %T for a straight ruling parent, want nil (the ruling is already on the host)", got)
	}
}

// TestExactRetainedSpanCarriesAnEllipticParent pins the elliptic arm's reuse: the ellipse's own
// exactness-gated sub-span is returned, so the one choke point covers both conic families.
func TestExactRetainedSpanCarriesAnEllipticParent(t *testing.T) {
	t.Parallel()
	ea := f6CapRim(t)
	sub := exactRetainedSpanOnParent(ea, ea.PointAt(0.05), ea.PointAt(0.95))
	if _, ok := sub.(geom.EllipticalArc); !ok {
		t.Fatalf("carried curve is %T, want geom.EllipticalArc", sub)
	}
}
