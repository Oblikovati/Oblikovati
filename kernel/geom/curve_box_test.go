// SPDX-License-Identifier: GPL-2.0-only

package geom_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A whole circle's box is the disc's box, not the seam point its two ends share.
func TestCurveBoxWholeCircle(t *testing.T) {
	c, err := geom.NewCircle(math.P3(1, 0, 5), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	box, ok := geom.CurveBox(c, 0, 1)
	if !ok {
		t.Fatalf("CurveBox declined a circle")
	}
	want := math.NewBox(math.P3(-1, -2, 5), math.P3(3, 2, 5))
	if box.Min.DistanceTo(want.Min) > 1e-9 || box.Max.DistanceTo(want.Max) > 1e-9 {
		t.Fatalf("circle box = %v, want %v", box, want)
	}
}

// A quarter arc reaches only its own quadrant: the box is the ends' box, with no full-circle bulge.
func TestCurveBoxQuarterArc(t *testing.T) {
	a, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 2, 0, stdmath.Pi/2)
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	box, ok := geom.CurveBox(a, 0, 1)
	if !ok {
		t.Fatalf("CurveBox declined an arc")
	}
	want := math.NewBox(math.P3(0, 0, 0), math.P3(2, 2, 0))
	if box.Min.DistanceTo(want.Min) > 1e-9 || box.Max.DistanceTo(want.Max) > 1e-9 {
		t.Fatalf("quarter-arc box = %v, want %v", box, want)
	}
}

// A half arc bulges past both ends: the box carries the interior stationary point.
func TestCurveBoxHalfArcBulge(t *testing.T) {
	a, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 2, 0, stdmath.Pi)
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	box, ok := geom.CurveBox(a, 0, 1)
	if !ok {
		t.Fatalf("CurveBox declined an arc")
	}
	if stdmath.Abs(float64(box.Max.Y)-2) > 1e-9 {
		t.Fatalf("half-arc box top = %v, want y=2 (the arc's crown)", box.Max)
	}
}

// A line segment's box is its two ends.
func TestCurveBoxLineSegment(t *testing.T) {
	s := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 2, 3))
	box, ok := geom.CurveBox(s, 0, 1)
	if !ok {
		t.Fatalf("CurveBox declined a segment")
	}
	want := math.NewBox(math.P3(0, 0, 0), math.P3(1, 2, 3))
	if box.Min.DistanceTo(want.Min) > 1e-9 || box.Max.DistanceTo(want.Max) > 1e-9 {
		t.Fatalf("segment box = %v, want %v", box, want)
	}
}

// obliqueFigureEightLobe is one lobe of a tilted torus cut by z=1 — the section at the saddle, whose
// curve kind (a spiric) has no closed-form axial extent, so CurveBox declines it.
func obliqueFigureEightLobe(t *testing.T) geom.Curve3 {
	t.Helper()
	ring, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0.6, 0.8), 5, 2)
	if err != nil {
		t.Fatalf("NewTorus: %v", err)
	}
	lid, err := geom.NewPlane(math.P3(0, 0, 1), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	curves, ok := geom.IntersectSurfacesAnalytic(ring, lid, geom.ResolutionForSize(20))
	if !ok || len(curves) == 0 {
		t.Fatalf("torus∩plane at the saddle: ok=%v n=%d", ok, len(curves))
	}
	return curves[0]
}

// CurveSpanBox bounds a curve whose kind has no closed-form extent, where CurveBox declines. A face
// bounded by ONE such closed curve otherwise measured its own scale as a point and took the model-size
// floor, which put its stitch weld grid at 1e-15 — below the rounding of its own coordinates
// (CI run 34280554924 macos-latest).
//
// Every probe is taken OFF the walk's own stations: the box is built at i/curveSpanSamples, so a probe
// on that grid is a point the box was extended with and cannot fail. offGridParams walks a prime count
// of half-offset parameters instead, which shares no station with the walk.
func TestCurveSpanBoxBoundsACurveWithNoClosedFormExtent(t *testing.T) {
	t.Parallel()
	lobe := obliqueFigureEightLobe(t)
	lo, hi := lobe.Domain()
	if _, ok := geom.CurveBox(lobe, lo, hi); ok {
		t.Skipf("%T now has a closed-form extent; this row needs another curve kind", lobe)
	}
	box := geom.CurveSpanBox(lobe, lo, hi)
	if float64(box.Diagonal().Length()) < 1 {
		t.Fatalf("%T span box = %v (diagonal %g): a lobe of a torus of major radius 5 is units across",
			lobe, box, float64(box.Diagonal().Length()))
	}
	for _, t01 := range offGridParams() {
		if p := lobe.PointAt(lo + (hi-lo)*t01); !box.Contains(p) {
			t.Fatalf("%T span box %v misses its own point %v at t=%g", lobe, box, p, t01)
		}
	}
}

// TestCurveSpanBoxHoldsAWigglyCurveItsWalkUnderBounds is the property the walk alone does NOT have.
// A helix of 6.4 turns is sampled 5 times a turn by curveSpanSamples stations, so the hull of those
// stations is an inscribed pentagon that misses the tube. Both of those numbers are load-bearing: a
// WHOLE turn count makes the stations divide the turn evenly and land on the axes, and a start on the
// reference direction puts the first station at +r itself, either of which makes the hull exact on an
// axis and the row vacuous. Hence 6.4 turns from a reference direction at 45°. The row asserts BOTH halves — that the bare hull really does miss, and that
// the returned box (the hull grown by the step reach the curve's own speed bounds) holds every
// off-grid point anyway.
func TestCurveSpanBoxHoldsAWigglyCurveItsWalkUnderBounds(t *testing.T) {
	t.Parallel()
	const radius = 3.0
	coil, err := geom.NewHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 1, 0), radius, 1, 0, 6.4, false)
	if err != nil {
		t.Fatalf("NewHelix3d: %v", err)
	}
	lo, hi := coil.Domain()
	if _, ok := geom.CurveBox(coil, lo, hi); ok {
		t.Skipf("%T now has a closed-form extent; this row needs a curve the walk under-bounds", coil)
	}
	hull, box := stationHull(coil, lo, hi), geom.CurveSpanBox(coil, lo, hi)
	missed, params := 0, offGridParams()
	for _, t01 := range params {
		p := coil.PointAt(lo + (hi-lo)*t01)
		if !hull.Contains(p) {
			missed++
		}
		if !box.Contains(p) {
			t.Errorf("%T span box %v misses its own point %v at t=%g", coil, box, p, t01)
		}
	}
	if missed == 0 { // never pass vacuously: without a real under-bound the grown box proves nothing
		t.Fatalf("the walk's bare hull %v already holds all %d off-grid points; this row needs a curve "+
			"whose walk under-bounds it", hull, len(params))
	}
}

// stationHull is the hull of the walk CurveSpanBox builds its box from, with no growth — the
// under-bound the grown box has to improve on.
func stationHull(c geom.Curve3, t0, t1 float64) math.Box {
	box := math.EmptyBox()
	for i := 0; i <= geom.CurveSpanSamplesForTest; i++ {
		box = box.ExtendPoint(c.PointAt(t0 + (t1-t0)*float64(i)/geom.CurveSpanSamplesForTest))
	}
	return box
}

// offGridParams returns parameters in (0, 1) that share no value with the walk's stations: a prime
// count of them, each offset half a step of its own spacing.
func offGridParams() []float64 {
	const probes = 97 // prime, so k/97 lands on i/32 only at the ends, which the half-offset removes
	out := make([]float64, probes)
	for k := range out {
		out[k] = (float64(k) + 0.5) / probes
	}
	return out
}

// CurveSpanBox is CurveBox where the closed form applies: same box, so nothing a conic bounds moves.
func TestCurveSpanBoxIsCurveBoxWhereTheClosedFormApplies(t *testing.T) {
	t.Parallel()
	c, err := geom.NewCircle(math.P3(1, 0, 5), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	want, ok := geom.CurveBox(c, 0, 1)
	if !ok {
		t.Fatalf("CurveBox declined a circle")
	}
	if got := geom.CurveSpanBox(c, 0, 1); got != want {
		t.Fatalf("CurveSpanBox = %v, want CurveBox's %v", got, want)
	}
}
