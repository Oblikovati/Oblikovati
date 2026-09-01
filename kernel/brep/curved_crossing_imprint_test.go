// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Crossing-cylinder imprint (M2 Phase 2, Oblikovati/Oblikovati#1335). The imprint stage must trace the
// surface-surface intersection of two crossing cylinders as closed loops lying on BOTH surfaces — the
// boundary the split/stitch slices will build the watertight result on.

// onBothCylinders returns the largest distance any loop vertex sits off either cylinder surface.
func onBothCylinders(loop geom.Curve3, a, b geom.Cylinder) float64 {
	worst := 0.0
	for _, p := range imprintLoopPoints(loop) {
		ea := stdmath.Abs(float64(geom.SignedDistanceToSurface(a, p)))
		eb := stdmath.Abs(float64(geom.SignedDistanceToSurface(b, p)))
		worst = stdmath.Max(worst, stdmath.Max(ea, eb))
	}
	return worst
}

// TestCrossingCylinderImprintThinThroughFat traces a thin cylinder crossing a fat one perpendicularly:
// the rod's entry and exit through the fat wall give two clean closed loops, each on both surfaces.
func TestCrossingCylinderImprintThinThroughFat(t *testing.T) {
	t.Parallel()
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)    // axis z, R=3
	thin, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12) // axis x, r=1.5, through the centre

	loops, ok := crossingCylinderImprint(fat, thin, nil)
	if !ok || len(loops) != 2 {
		t.Fatalf("thin-through-fat imprint: ok=%v loops=%d, want 2 closed loops", ok, len(loops))
	}
	ca, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	cb, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 1.5)
	for i, lp := range loops {
		if !samePoint(lp.PointAt(0), lp.PointAt(1), geom.ResolutionForSize(1)) {
			t.Errorf("loop %d is not closed: %v vs %v", i, lp.PointAt(0), lp.PointAt(1))
		}
		if err := onBothCylinders(lp, ca, cb); err > 1e-5 {
			t.Errorf("loop %d sits %.2e off a cylinder surface, want it on both", i, err)
		}
	}
}

// TestClosedTraceLoopsRecordsUnclosedChain: a traced chain that does not close (more than a single
// tangency marker, endpoints apart) must be DROPPED from the watertight imprint but NOT silently — it
// raises a CodeImprintUnclosedChain diagnostic so the degradation is visible (Oblikovati#1404).
func TestClosedTraceLoopsRecordsUnclosedChain(t *testing.T) {
	t.Parallel()
	res := geom.ResolutionForSize(10)
	open := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0), math.P3(5, 0, 0)} // ends 5 apart
	rec := &diag.Recorder{}
	loops := closedTraceLoops(geom.SurfaceIntersection{Curves: [][]math.Point3{open}}, res, rec)
	if len(loops) != 0 {
		t.Fatalf("an unclosed chain yielded %d loops, want 0 (it must be dropped)", len(loops))
	}
	if !rec.Has(CodeImprintUnclosedChain) {
		t.Errorf("dropping an unclosed chain recorded no %q diagnostic; got %v", CodeImprintUnclosedChain, rec.Records())
	}
	if rec.Count(diag.Defect) != 1 {
		t.Errorf("recorded %d defects, want exactly 1 for the one dropped chain", rec.Count(diag.Defect))
	}
}

// TestClosedTraceLoopsQuietOnCleanTrace: a closed loop is kept and a single-point tangency marker is
// skipped, neither raising a diagnostic — only a genuine unclosed chain is a tracked defect (#1404).
func TestClosedTraceLoopsQuietOnCleanTrace(t *testing.T) {
	t.Parallel()
	res := geom.ResolutionForSize(10)
	square := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0), math.P3(0, 0, 0)}
	marker := []math.Point3{math.P3(9, 9, 9)} // an isolated tangential contact, not a chain
	rec := &diag.Recorder{}
	loops := closedTraceLoops(geom.SurfaceIntersection{Curves: [][]math.Point3{square, marker}}, res, rec)
	if len(loops) != 1 {
		t.Fatalf("got %d loops, want 1 (the closed square; the marker is not a loop)", len(loops))
	}
	if rec.Count(diag.Defect) != 0 {
		t.Errorf("a clean trace recorded %d defects, want 0; got %v", rec.Count(diag.Defect), rec.Records())
	}
}

// TestCrossingCylinderImprintEqualRadiusPinch: the equal-radius Steinmetz case — two ellipses crossing at
// two pinch points — used to break the tracer (it stopped at the first pinch), which is why an analytic
// Steinmetz constructor exists. The through-pinch tracer (Oblikovati#1404) now returns both closed ellipse
// loops here too: this pins the prerequisite the general curved∩curved pipeline (#1403) needs to one day
// fold curved_steinmetz.go onto the traced path. Each loop must be a planar ellipse of minor radius R and
// major radius R√2 (the Steinmetz ellipse), lying on both cylinders.
func TestCrossingCylinderImprintEqualRadiusPinch(t *testing.T) {
	t.Parallel()
	const r = 3.0
	a, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), r, 12) // axis x
	b, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), r, 12) // axis z
	loops, ok := crossingCylinderImprint(a, b, nil)
	if !ok || len(loops) != 2 {
		t.Fatalf("equal-radius imprint: ok=%v loops=%d, want 2 closed Steinmetz ellipses", ok, len(loops))
	}
	ca, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), r)
	cb, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	for i, lp := range loops {
		if !samePoint(lp.PointAt(0), lp.PointAt(1), geom.ResolutionForSize(1)) {
			t.Errorf("loop %d is not closed: %v vs %v", i, lp.PointAt(0), lp.PointAt(1))
		}
		if err := onBothCylinders(lp, ca, cb); err > 1e-4 {
			t.Errorf("loop %d sits %.2e off a cylinder surface", i, err)
		}
		near, far := loopRadialExtent(lp)
		if stdmath.Abs(near-r) > 1e-3 || stdmath.Abs(far-stdmath.Sqrt2*r) > 1e-3 {
			t.Errorf("loop %d radial extent [%.4f,%.4f], want [R=%.4f, R√2=%.4f] (a Steinmetz ellipse)", i, near, far, r, stdmath.Sqrt2*r)
		}
	}
}

// loopRadialExtent returns the min and max distance of a loop's vertices from the origin — for a Steinmetz
// ellipse (centred at the axis crossing) this is its minor (R) and major (R√2) semi-axis length.
func loopRadialExtent(lp geom.Curve3) (near, far float64) {
	near, far = stdmath.Inf(1), 0
	for _, p := range imprintLoopPoints(lp) {
		d := float64(p.AsVector().Length())
		near, far = stdmath.Min(near, d), stdmath.Max(far, d)
	}
	return near, far
}

// TestCrossingCylinderImprintNonCylinderDefers: the imprint only handles bare cylinders.
func TestCrossingCylinderImprintNonCylinderDefers(t *testing.T) {
	t.Parallel()
	block, _ := SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "b")
	cyl, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1, 4)
	if _, ok := crossingCylinderImprint(block, cyl, nil); ok {
		t.Error("imprint of a block and a cylinder should defer (ok=false)")
	}
}

// TestCrossingCylinderImprintDisjointHasNoLoops: cylinders that do not meet trace no loop.
func TestCrossingCylinderImprintDisjointHasNoLoops(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1, 4)
	b, _ := SolidCylinder(math.P3(20, 0, 0), math.V3(1, 0, 0), 1, 4) // far away
	if _, ok := crossingCylinderImprint(a, b, nil); ok {
		t.Error("disjoint cylinders should trace no imprint loop (ok=false)")
	}
}

// TestCrossingCylinderImprintSquatFat: squat proportions (R ≫ h) used to size the SSI march from the
// axial height alone — the corner-to-corner extent estimate maps the periodic angular corners to the
// same generator line (#1597) — starving the trace of step and tolerance. The rod's entry and exit
// through the squat fat wall must still give two clean closed loops, with no recorded degradation.
func TestCrossingCylinderImprintSquatFat(t *testing.T) {
	t.Parallel()
	fat, _ := SolidCylinder(math.P3(0, 0, -2), math.V3(0, 0, 1), 50, 4)     // squat: R=50, h=4
	rod, _ := SolidCylinder(math.P3(-60, 0, 0), math.V3(1, 0, 0), 1.5, 120) // axis x, through both walls
	rec := &diag.Recorder{}
	loops, ok := crossingCylinderImprint(fat, rod, rec)
	if !ok || len(loops) != 2 {
		t.Fatalf("squat-fat imprint: ok=%v loops=%d, want 2 closed loops", ok, len(loops))
	}
	if rec.Count(diag.Defect) != 0 {
		t.Errorf("squat-fat imprint recorded %d defects, want 0; got %v", rec.Count(diag.Defect), rec.Records())
	}
	ca, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 50)
	cb, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 1.5)
	for i, lp := range loops {
		if !samePoint(lp.PointAt(0), lp.PointAt(1), geom.ResolutionForSize(1)) {
			t.Errorf("loop %d is not closed: %v vs %v", i, lp.PointAt(0), lp.PointAt(1))
		}
		if err := onBothCylinders(lp, ca, cb); err > 1e-4 {
			t.Errorf("loop %d sits %.2e off a cylinder surface, want it on both", i, err)
		}
	}
}

// TestCrossingCylinderImprintSnapBandSilent: radii closer than the snap ceiling decline the rod-band imprint
// (so dispatch falls through to the exact Steinmetz constructor, which snaps them — #1780) WITHOUT recording a
// degradation. The snap is honest, not a fallback, so it must raise no near-pinch defect.
func TestCrossingCylinderImprintSnapBandSilent(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	base, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	ceil := geom.ResolutionForBox(a.RangeBox().Union(base.RangeBox())).Stitch()
	b, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3+0.5*ceil, 12)
	rec := &diag.Recorder{}
	if _, ok := crossingCylinderImprint(a, b, rec); ok {
		t.Fatal("snap-band radii must decline the rod-band imprint (Steinmetz snaps instead)")
	}
	if rec.Count(diag.Defect) != 0 || rec.Has(CodeImprintNearPinchDeclined) {
		t.Errorf("snap-band decline must be SILENT (no degradation to record); got %v", rec.Records())
	}
}

// TestCrossingCylinderImprintResidualBandRecords: radii ABOVE the snap ceiling but inside the near-pinch band
// decline AND record exactly one CodeImprintNearPinchDeclined defect — the genuine, non-silent fallback the
// residual band still takes until #1780 Direction 2 folds it onto the analytic path.
func TestCrossingCylinderImprintResidualBandRecords(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	base, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	ceil := geom.ResolutionForBox(a.RangeBox().Union(base.RangeBox())).Stitch()
	b, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3+4*ceil, 12)
	rec := &diag.Recorder{}
	if _, ok := crossingCylinderImprint(a, b, rec); ok {
		t.Fatal("residual-band radii must decline the rod-band imprint")
	}
	if !rec.Has(CodeImprintNearPinchDeclined) || rec.Count(diag.Defect) != 1 {
		t.Errorf("residual-band decline must record exactly one near-pinch defect; got %v", rec.Records())
	}
}

// TestKeepImprintLoopsRecordsFallbackContour: curves whose provenance is the marching-squares fallback
// must raise CodeImprintFallbackContour — the imprint proceeds on contour-quality loops, but the
// degradation is recorded instead of silent (#1597).
func TestKeepImprintLoopsRecordsFallbackContour(t *testing.T) {
	t.Parallel()
	res := geom.ResolutionForSize(10)
	square := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0), math.P3(0, 0, 0)}
	rec := &diag.Recorder{}
	loops := keepImprintLoops(geom.SurfaceIntersection{Curves: [][]math.Point3{square}, ViaFallback: true}, res, rec)
	if len(loops) != 1 {
		t.Fatalf("got %d loops, want 1 (fallback curves are kept, only flagged)", len(loops))
	}
	if !rec.Has(CodeImprintFallbackContour) {
		t.Errorf("fallback-supplied curves recorded no %q diagnostic; got %v", CodeImprintFallbackContour, rec.Records())
	}
}
