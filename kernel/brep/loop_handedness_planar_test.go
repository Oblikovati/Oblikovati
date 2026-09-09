// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// quarterTurnLeftOf is the expression quarterArcLeftOf generalises: a quarter TURN in (u, v), a quarter
// of the segment long. It is correct on a metrically isotropic chart and only there, which is why it was
// replaced — see quarterArcLeftOf. It is kept here, and only here, as the thing the general formula must
// reproduce.
func quarterTurnLeftOf(d math.Vector2) math.Vector2 {
	return math.V2(math.Scalar(-float64(d.Y)/4), math.Scalar(float64(d.X)/4))
}

// TestQuarterArcLeftOfReproducesTheQuarterTurnOnAPlane is the proof the ground rules ask for: a formula
// generalised to a new chart must reproduce the case it replaces BIT FOR BIT, not merely test green.
//
// On a plane whose frame is an exact orthonormal triple of axis vectors the reduction is exact in
// floating point too: E = G = 1 and F = 0 come out of dot products of ±1 and 0 components, |N| = 1
// likewise, so quarterArcLeftOf's divisions are by exactly 1 and its N × T is a signed permutation of
// the components — the same arithmetic the quarter turn does, in a different order.
func TestQuarterArcLeftOfReproducesTheQuarterTurnOnAPlane(t *testing.T) {
	plane, err := geom.NewPlane(math.P3(3, -2, 7), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	for _, d := range quarterTurnProbeDirections() {
		got, ok := quarterArcLeftOf(plane, math.P2(0.25, -0.5), d)
		if !ok {
			t.Fatalf("d=%v: quarterArcLeftOf declined an ordinary planar segment", d)
		}
		if want := quarterTurnLeftOf(d); got != want {
			t.Errorf("d=%v: quarterArcLeftOf = %v, want the quarter turn %v exactly", d, got, want)
		}
	}
}

// TestQuarterArcLeftOfMatchesTheQuarterTurnOnATiltedPlane covers the case the exact reduction cannot: a
// plane whose frame vectors are irrational, so E, G and |N| are 1 only to ROUNDING. It is still the same
// quarter turn — it just arrives through divisions by 1±ulp — so the agreement is bounded in ulps of the
// offset's own length rather than exact, and the bound is measured (the worst case over the directions below is 1.414 of it),
// not assumed.
func TestQuarterArcLeftOfMatchesTheQuarterTurnOnATiltedPlane(t *testing.T) {
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0.3, -0.7, 0.5))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	for _, d := range quarterTurnProbeDirections() {
		got, ok := quarterArcLeftOf(plane, math.P2(1.5, 2.5), d)
		if !ok {
			t.Fatalf("d=%v: quarterArcLeftOf declined an ordinary planar segment", d)
		}
		want := quarterTurnLeftOf(d)
		if gap, bound := offsetGap(got, want), quarterTurnPlanarUlps*ulpOf(offsetLength(want)); gap > bound {
			t.Errorf("d=%v: quarterArcLeftOf = %v, quarter turn %v — apart by %g, want at most %g",
				d, got, want, gap, bound)
		}
	}
}

// quarterTurnPlanarUlps is the measured agreement on a frame that is orthonormal only to rounding. It is
// a floating-point distance in representable steps, not a modelling tolerance: the two expressions are
// the same formula, evaluated in a different order.
const quarterTurnPlanarUlps = 8

// quarterTurnProbeDirections spans the signs and the axis-aligned degenerate cases a boundary segment
// can take.
func quarterTurnProbeDirections() []math.Vector2 {
	return []math.Vector2{
		math.V2(1, 0), math.V2(0, 1), math.V2(-1, 0), math.V2(0, -1),
		math.V2(1, 1), math.V2(-3, 2), math.V2(0.125, -0.0625), math.V2(1e-6, 4e-6),
	}
}

// offsetGap is the distance between two (u, v) offsets.
func offsetGap(a, b math.Vector2) float64 {
	return stdmath.Hypot(float64(a.X)-float64(b.X), float64(a.Y)-float64(b.Y))
}

// offsetLength is a (u, v) offset's own length, the magnitude the ulp bound is taken at.
func offsetLength(v math.Vector2) float64 {
	return stdmath.Hypot(float64(v.X), float64(v.Y))
}

// ulpOf is one representable float64 step at the magnitude m — the unit "a few ulps" counts in. A
// per-component ulp count is the wrong metric here: a quarter turn of an axis-aligned direction has a
// component that is exactly zero, and every nonzero value is astronomically many ulps from zero.
func ulpOf(m float64) float64 {
	return stdmath.Nextafter(m, stdmath.Inf(1)) - m
}
