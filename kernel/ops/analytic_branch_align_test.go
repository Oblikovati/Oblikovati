// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"
)

// Unit guards for putting a face's loops on one branch of the covering space before asking which
// contains which (M48/C3, Oblikovati/Oblikovati#3489). TestDrilledWallAreaSubtractsItsWindows covers
// the same code through a real drilled body; these pin the arithmetic directly, including the cases
// that body does not reach.

func TestWholePeriodOffsetClosesAGapByWholePeriods(t *testing.T) {
	const tau = 2 * stdmath.Pi
	cases := []struct {
		name        string
		gap, period float64
		want        float64
	}{
		{"a gap of one period is closed by exactly one", tau, tau, tau},
		{"a gap of two periods is closed by two", -2 * tau, tau, -2 * tau},
		{"a gap under half a period is left alone", 0.4 * tau, tau, 0},
		{"a gap just over half a period rounds to one", 0.6 * tau, tau, tau},
		{"a non-periodic parameter never shifts", 500, 0, 0},
		{"a negative period is treated as non-periodic", 500, -1, 0},
		{"a period other than 2pi is respected", 7, 3, 6},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := wholePeriodOffset(c.gap, c.period); stdmath.Abs(got-c.want) > 1e-12 {
				t.Errorf("wholePeriodOffset(%g, %g) = %g, want %g", c.gap, c.period, got, c.want)
			}
		})
	}
}

func TestPolygonMeanOfAnEmptyPolygonIsTheOrigin(t *testing.T) {
	u, v := polygonMean(nil)
	if u != 0 || v != 0 {
		t.Errorf("polygonMean(nil) = (%g, %g), want the origin; a degenerate loop keeps its slot and must not divide by zero", u, v)
	}
}

func TestPolygonMeanAveragesEverySample(t *testing.T) {
	u, v := polygonMean([]arcSample{{u: 1, v: 10}, {u: 2, v: 20}, {u: 3, v: 30}})
	if stdmath.Abs(u-2) > 1e-12 || stdmath.Abs(v-20) > 1e-12 {
		t.Errorf("polygonMean = (%g, %g), want (2, 20)", u, v)
	}
}

// TestAlignPolygonBranchesBringsAHoleOntoItsOuterLoopsBranch is the bug itself, in miniature: a hole
// unwrapped onto the NEXT branch up must come back beside the loop that contains it, or the nesting
// test reads it as outside and the region ADDS it.
func TestAlignPolygonBranchesBringsAHoleOntoItsOuterLoopsBranch(t *testing.T) {
	const tau = 2 * stdmath.Pi
	outer := []arcSample{{u: -tau, v: 0}, {u: 0, v: 0}, {u: 0, v: 12}, {u: -tau, v: 12}}
	hole := []arcSample{{u: 2.5, v: 5}, {u: 3.5, v: 5}, {u: 3.5, v: 7}, {u: 2.5, v: 7}} // one branch up

	got := alignPolygonBranches([][]arcSample{outer, hole}, tau, 0)

	uMin, uMax := got[1][0].u, got[1][0].u
	for _, sp := range got[1] {
		uMin, uMax = stdmath.Min(uMin, sp.u), stdmath.Max(uMax, sp.u)
	}
	if uMin < -tau || uMax > 0 {
		t.Errorf("the hole spans u ∈ [%g, %g] after alignment, which is outside its outer loop's branch [%g, 0]", uMin, uMax, -tau)
	}
	if w := uMax - uMin; stdmath.Abs(w-1) > 1e-12 {
		t.Errorf("the hole is %g wide after alignment, want 1: a whole-period shift must preserve its shape", w)
	}
	if got[1][0].v != 5 {
		t.Errorf("v = %g, want 5 unchanged: a non-periodic parameter must not be shifted", got[1][0].v)
	}
}

// TestAlignPolygonBranchesLeavesASinglePolygonAlone: with nothing to agree with, there is no branch
// to move to — and a face with one loop is the common case, so this path must stay free.
func TestAlignPolygonBranchesLeavesASinglePolygonAlone(t *testing.T) {
	only := []arcSample{{u: 99, v: 1}}
	got := alignPolygonBranches([][]arcSample{only}, 2*stdmath.Pi, 0)
	if len(got) != 1 || got[0][0].u != 99 {
		t.Errorf("a lone polygon was moved to %v; it defines the branch and cannot be off it", got)
	}
}

// TestAlignPolygonBranchesKeepsPolygonsAlreadyTogether guards the false-positive direction: loops that
// already share a branch must come back untouched, not nudged by a spurious whole period.
func TestAlignPolygonBranchesKeepsPolygonsAlreadyTogether(t *testing.T) {
	const tau = 2 * stdmath.Pi
	outer := []arcSample{{u: 0, v: 0}, {u: tau, v: 0}, {u: tau, v: 4}, {u: 0, v: 4}}
	hole := []arcSample{{u: 3, v: 1}, {u: 4, v: 1}, {u: 4, v: 2}, {u: 3, v: 2}}
	got := alignPolygonBranches([][]arcSample{outer, hole}, tau, 0)
	if got[1][0].u != 3 {
		t.Errorf("a hole already on the outer loop's branch moved to u=%g, want 3", got[1][0].u)
	}
}
