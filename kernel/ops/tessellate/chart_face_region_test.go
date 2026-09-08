// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// splitBandChart is the shape a region that WRAPS a period arrives in: the ring bored by a coaxial
// shaft records its torus face as two disjoint positive rectangles, v ∈ [3.9827, 2π] and v ∈ [0, 2.3005],
// which are one band through the v seam (measured, ADR-0061).
func splitBandChart() [][]math.Point2 {
	rect := func(v0, v1 float64) []math.Point2 {
		return []math.Point2{
			math.P2(stdmath.Pi, v0), math.P2(3*stdmath.Pi, v0),
			math.P2(3*stdmath.Pi, v1), math.P2(stdmath.Pi, v1),
		}
	}
	return [][]math.Point2{rect(3.9827, 2*stdmath.Pi), rect(0, 2.3005)}
}

// doublyPeriodicRegion is that chart read as a torus face's region.
func doublyPeriodicRegion() chartRegion {
	r := chartRegion{contours: splitBandChart(), uPer: true, vPer: true}
	r.uLo, r.uHi = branchWindow(stdmath.Pi, 3*stdmath.Pi, true)
	r.vLo, r.vHi = branchWindow(0, 2*stdmath.Pi, true)
	return r
}

// TestASplitChartReadsAsOneBand: even-odd over ALL contours together makes the two rectangles the one
// band they are. Reading the first as an outer and the second as its hole would invert the face.
func TestASplitChartReadsAsOneBand(t *testing.T) {
	t.Parallel()
	r := doublyPeriodicRegion()
	for _, c := range []struct {
		v    float64
		want bool
	}{{5.0, true}, {1.0, true}, {3.0, false}, {2.30051, false}, {0.001, true}} {
		if got := r.covers(4.0, c.v); got != c.want {
			t.Errorf("covers(4, %g) = %v, want %v", c.v, got, c.want)
		}
	}
}

// TestAQueryIsCarriedOntoTheChartsBranch: the chart lives on [π, 3π] in u, so a query a whole turn away
// is the same point and must read the same. A raw even-odd would call it outside.
func TestAQueryIsCarriedOntoTheChartsBranch(t *testing.T) {
	t.Parallel()
	r := doublyPeriodicRegion()
	for _, u := range []float64{4.0, 4.0 - 2*stdmath.Pi, 4.0 + 4*stdmath.Pi} {
		if !r.covers(u, 5.0) {
			t.Errorf("covers(%g, 5) = false; the query must fold onto the chart's branch", u)
		}
	}
	if fu, fv := r.fold(4.0-2*stdmath.Pi, 5.0-2*stdmath.Pi); stdmath.Abs(fu-4.0) > 1e-9 || stdmath.Abs(fv-5.0) > 1e-9 {
		t.Errorf("fold = (%g, %g), want (4, 5)", fu, fv) // tol:numeric — the fold's own rounding
	}
}

// TestTheBranchWindowKeepsExactlyOneReplica: the window is half-open, so of a point and its whole-period
// translates exactly one is in it. That is what de-duplicates a seam-spanning triangle.
func TestTheBranchWindowKeepsExactlyOneReplica(t *testing.T) {
	t.Parallel()
	r := doublyPeriodicRegion()
	for _, u := range []float64{stdmath.Pi, 4.0, 3*stdmath.Pi - 1e-9} {
		in := 0
		for k := -2; k <= 2; k++ {
			if r.inWindow(u+float64(k)*2*stdmath.Pi, 1.0) {
				in++
			}
		}
		if in != 1 {
			t.Errorf("u=%g: %d of its five replicas are in the window, want exactly 1", u, in)
		}
	}
	if r.inWindow(3*stdmath.Pi, 1.0) {
		t.Error("the window's upper end is closed; it must be half-open or the seam triangle is meshed twice")
	}
}

// TestABoundedAxisHasNoReplicas: a sphere's latitude does not wrap, so the covering replicates only in
// longitude — nine copies on a torus, three on a sphere or a wall, one on neither.
func TestABoundedAxisHasNoReplicas(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		uPer, vPer bool
		want       int
	}{{true, true, 9}, {true, false, 3}, {false, true, 3}, {false, false, 1}} {
		r := chartRegion{uPer: c.uPer, vPer: c.vPer}
		if got := len(r.shifts()); got != c.want {
			t.Errorf("shifts for (uPer=%v, vPer=%v) = %d, want %d", c.uPer, c.vPer, got, c.want)
		}
	}
}

// TestBranchWindowSpansOnePeriodOnlyWhereTheAxisWraps.
func TestBranchWindowSpansOnePeriodOnlyWhereTheAxisWraps(t *testing.T) {
	t.Parallel()
	if lo, hi := branchWindow(1.0, 1.5, true); hi-lo != 2*stdmath.Pi || lo != 1.0 {
		t.Errorf("branchWindow wrapping = [%g,%g], want one period from the lower bound", lo, hi)
	}
	if lo, hi := branchWindow(-1.5708, 1.5708, false); lo != -1.5708 || hi != 1.5708 {
		t.Errorf("branchWindow bounded = [%g,%g], want the contours' own extent", lo, hi)
	}
}

// TestChartBoundsRejectsAContourThatBoundsNoArea: a chart collapsed onto a line has no region to mesh.
func TestChartBoundsRejectsAContourThatBoundsNoArea(t *testing.T) {
	t.Parallel()
	flat := [][]math.Point2{{math.P2(0, 1), math.P2(1, 1), math.P2(2, 1)}}
	if _, _, _, _, ok := chartBounds(flat); ok {
		t.Error("chartBounds accepted a contour of zero v extent")
	}
	u0, u1, v0, v1, ok := chartBounds(splitBandChart())
	if !ok || u0 != stdmath.Pi || u1 != 3*stdmath.Pi || v0 != 0 || v1 != 2*stdmath.Pi {
		t.Errorf("chartBounds = [%g,%g]x[%g,%g] ok=%v, want the union box of both contours", u0, u1, v0, v1, ok)
	}
}

// TestBranchOffsetCarriesAWholeTraceAtOnce: one shift per axis, chosen by the trace's mean, because a
// continuous trace differs from the chart's branch by a whole number of periods and nothing else.
func TestBranchOffsetCarriesAWholeTraceAtOnce(t *testing.T) {
	t.Parallel()
	r := doublyPeriodicRegion()
	trace := []float64{0.1, 1.0, 2.0, 3.0} // mean 1.525, a period below the chart's [π, 3π]
	du, dv := r.branchOffset(trace, []float64{1, 2, 3})
	if du != 2*stdmath.Pi {
		t.Errorf("branchOffset u = %g, want one period up onto [π, 3π]", du)
	}
	if dv != 0 {
		t.Errorf("branchOffset v = %g, want none; the trace is already on [0, 2π]", dv)
	}
	if got := meanOfParams(trace); stdmath.Abs(got-1.525) > 1e-12 { // tol:numeric — the mean's own rounding
		t.Errorf("meanOfParams = %g, want 1.525", got)
	}
	if got := meanOfParams(nil); got != 0 {
		t.Errorf("meanOfParams(nil) = %g, want 0", got)
	}
}
