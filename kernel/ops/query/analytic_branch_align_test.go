// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"
	"testing"
)

// Unit guards for putting a face's loops on one branch of the covering space before asking which
// contains which (M48/C3, Oblikovati/Oblikovati#3489). TestDrilledWallAreaSubtractsItsWindows covers
// the same code through a real drilled body; these pin the arithmetic directly, including the cases
// that body does not reach.

func TestWholePeriodOffsetClosesAGapByWholePeriods(t *testing.T) {
	t.Parallel()
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

func TestPolygonRangeAlongOfAnEmptyPolygonIsRefused(t *testing.T) {
	t.Parallel()
	if _, ok := polygonRangeAlong(nil, bandAxis{}); ok {
		t.Error("an empty polygon has no extent; a degenerate loop keeps its slot and must not report one")
	}
}

func TestPolygonRangeAlongSpansEverySample(t *testing.T) {
	t.Parallel()
	poly := []arcSample{{u: 2, v: 30}, {u: 1, v: 10}, {u: 3, v: 20}}
	got, ok := polygonRangeAlong(poly, bandAxis{})
	if !ok || got != (paramRange{1, 3}) {
		t.Errorf("u range = %+v (ok=%v), want {1 3}", got, ok)
	}
	if got, _ := polygonRangeAlong(poly, bandAxis{alongV: true}); got != (paramRange{10, 30}) {
		t.Errorf("v range = %+v, want {10 30}", got)
	}
}

// TestALoopHalfAPeriodAwayIsNotMoved is the sphere-zone regression, in miniature. Two rims of a zone
// sit exactly half a period apart in their chart, which is the TIE POINT of "shift to the nearest
// branch": rounding sent one of them a whole period away, out of the region entirely, and the belt
// went on to name the caps and report four times the area. A shift is applied only when it lands the
// loop INSIDE the outer one, and no shift does here.
func TestALoopHalfAPeriodAwayIsNotMoved(t *testing.T) {
	t.Parallel()
	const tau = 2 * stdmath.Pi
	first := []arcSample{{u: 4.07, v: -0.64}, {u: 5.36, v: 0.64}}
	second := []arcSample{{u: 0.93, v: -0.64}, {u: 2.21, v: 0.64}}

	got := alignPolygonBranches([][]arcSample{first, second}, tau, 0)

	if got[1][0].u != 0.93 {
		t.Errorf("a loop half a period away moved to u=%g, want 0.93 unchanged: no whole period puts it inside the first loop, so none is justified", got[1][0].u)
	}
}

// TestParamRangeHolds pins the containment test the shift is justified by.
func TestParamRangeHolds(t *testing.T) {
	t.Parallel()
	outer := paramRange{-6.28, 0}
	if !outer.holds(paramRange{-3.78, -2.78}) {
		t.Error("an interval strictly within the outer one must read as held")
	}
	if outer.holds(paramRange{-1, 1}) {
		t.Error("an interval running past the outer one's end must not read as held")
	}
	if got := (paramRange{2, 4}).mid(); got != 3 {
		t.Errorf("mid = %g, want 3", got)
	}
	if got := (paramRange{2, 4}).shifted(-6); got != (paramRange{-4, -2}) {
		t.Errorf("shifted = %+v, want {-4 -2}", got)
	}
}

// TestAlignPolygonBranchesBringsAHoleOntoItsOuterLoopsBranch is the bug itself, in miniature: a hole
// unwrapped onto the NEXT branch up must come back beside the loop that contains it, or the nesting
// test reads it as outside and the region ADDS it.
func TestAlignPolygonBranchesBringsAHoleOntoItsOuterLoopsBranch(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	only := []arcSample{{u: 99, v: 1}}
	got := alignPolygonBranches([][]arcSample{only}, 2*stdmath.Pi, 0)
	if len(got) != 1 || got[0][0].u != 99 {
		t.Errorf("a lone polygon was moved to %v; it defines the branch and cannot be off it", got)
	}
}

// TestAlignPolygonBranchesKeepsPolygonsAlreadyTogether guards the false-positive direction: loops that
// already share a branch must come back untouched, not nudged by a spurious whole period.
func TestAlignPolygonBranchesKeepsPolygonsAlreadyTogether(t *testing.T) {
	t.Parallel()
	const tau = 2 * stdmath.Pi
	outer := []arcSample{{u: 0, v: 0}, {u: tau, v: 0}, {u: tau, v: 4}, {u: 0, v: 4}}
	hole := []arcSample{{u: 3, v: 1}, {u: 4, v: 1}, {u: 4, v: 2}, {u: 3, v: 2}}
	got := alignPolygonBranches([][]arcSample{outer, hole}, tau, 0)
	if got[1][0].u != 3 {
		t.Errorf("a hole already on the outer loop's branch moved to u=%g, want 3", got[1][0].u)
	}
}

// TestLoopsWrapASeamAsksThePeriodNotZero pins the difference between a loop that WRAPS and a loop
// that merely fails to close to the last bit. Net uv travel is accumulated from ParamAt round trips,
// so a loop that closes perfectly still reports a residue — measured, 3.0e-8 on a torus section and
// 1.8e-7 on its planar cap. Judging that against a bare absolute epsilon called both of them
// seam-wrapping and sent ordinary bounded faces down the band path.
func TestLoopsWrapASeamAsksThePeriodNotZero(t *testing.T) {
	t.Parallel()
	const tau = 2 * stdmath.Pi
	closed := []arcSample{{u: 0, v: 0}, {u: 1, v: 0}, {u: 1, v: 1}}
	periodic := func(net float64) []faceLoop {
		return []faceLoop{{netU: net, edges: []loopEdge{{samples: closed, uPeriod: tau}}}}
	}
	if loopsWrapASeam(periodic(2.98e-8)) {
		t.Error("a round-trip residue of 3e-8 is not a wrap; a wrap is a whole period")
	}
	if !loopsWrapASeam(periodic(tau)) {
		t.Error("travelling one whole period IS a wrap")
	}
	if !loopsWrapASeam(periodic(-tau)) {
		t.Error("travelling one period backwards is equally a wrap")
	}
	flat := []faceLoop{{netV: 1.788e-7, edges: []loopEdge{{samples: closed}}}}
	if loopsWrapASeam(flat) {
		t.Error("a parameter with no period cannot wrap, whatever residue it carries")
	}
}
