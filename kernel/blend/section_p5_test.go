// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestRadiusLaws checks the three OCCT Law_Function analogues: constant, linear (with end clamps),
// and interpolated (sorted, piecewise-linear, end-clamped).
func TestRadiusLaws(t *testing.T) {
	if got := (ConstLaw{R: 2}).At(99); got != 2 {
		t.Errorf("ConstLaw.At = %g, want 2", got)
	}
	lin := LinearLaw{S0: 0, R0: 1, S1: 4, R1: 3}
	for _, tc := range []struct{ w, want float64 }{{-1, 1}, {0, 1}, {2, 2}, {4, 3}, {9, 3}} {
		if got := lin.At(tc.w); stdmath.Abs(got-tc.want) > 1e-12 {
			t.Errorf("LinearLaw.At(%g) = %g, want %g", tc.w, got, tc.want)
		}
	}
	interp := NewInterpLaw([]LawStop{{S: 4, R: 5}, {S: 0, R: 1}, {S: 2, R: 2}}) // deliberately unsorted
	for _, tc := range []struct{ w, want float64 }{{-1, 1}, {0, 1}, {1, 1.5}, {2, 2}, {3, 3.5}, {4, 5}, {5, 5}} {
		if got := interp.At(tc.w); stdmath.Abs(got-tc.want) > 1e-12 {
			t.Errorf("InterpLaw.At(%g) = %g, want %g", tc.w, got, tc.want)
		}
	}
}

// TestSectionFunctionalExtents checks each section family reports the right chamfer flag and the
// governing size the validity bound reads.
func TestSectionFunctionalExtents(t *testing.T) {
	evol := EvolRadiusFillet{Law: LinearLaw{S0: 0, R0: 1, S1: 2, R1: 3}}
	if evol.IsChamfer() || stdmath.Abs(evol.Extent(1)-2) > 1e-12 {
		t.Errorf("EvolRadiusFillet: chamfer=%v extent(1)=%g, want false/2", evol.IsChamfer(), evol.Extent(1))
	}
	two := TwoDistanceChamfer{D1: 0.3, D2: 0.6}
	if !two.IsChamfer() || two.Extent(0) != 0.6 {
		t.Errorf("TwoDistanceChamfer: chamfer=%v extent=%g, want true/0.6", two.IsChamfer(), two.Extent(0))
	}
	da := DistanceAngleChamfer{D1: 1, Angle: stdmath.Pi / 4} // tan45=1 ⇒ D2=1
	if !da.IsChamfer() || stdmath.Abs(da.D2()-1) > 1e-12 || stdmath.Abs(da.Extent(0)-1) > 1e-12 {
		t.Errorf("DistanceAngleChamfer: chamfer=%v D2=%g extent=%g, want true/1/1", da.IsChamfer(), da.D2(), da.Extent(0))
	}
}

// TestChamferSectionChord checks the chamfer chord recedes each setback along its support's inward
// perpendicular and interpolates straight from FootA to FootB.
func TestChamferSectionChord(t *testing.T) {
	sec := chamferSectionAt(math.P3(0, 0, 0), math.V3(1, 0, 0), math.V3(0, 1, 0), 1, 2)
	if d := sec.FootA.DistanceTo(math.P3(1, 0, 0)); float64(d) > 1e-12 {
		t.Errorf("footA = %v, want (1,0,0)", sec.FootA)
	}
	if d := sec.FootB.DistanceTo(math.P3(0, 2, 0)); float64(d) > 1e-12 {
		t.Errorf("footB = %v, want (0,2,0)", sec.FootB)
	}
	if d := sec.PointAt(0.5).DistanceTo(math.P3(0.5, 1, 0)); float64(d) > 1e-12 {
		t.Errorf("chord midpoint = %v, want (0.5,1,0)", sec.PointAt(0.5))
	}
}
