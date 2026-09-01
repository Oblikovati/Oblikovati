// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestArmSectionArcSpansFilletQuarterCircle unit-tests the fillet cross-section helper against the
// S1 fillet at the +x fillet-cut station: it must run from the plane-A contact (·,-10,4) to the
// plane-B contact (·,-4,10) through the 45° bisector, all at radius 6 from the cylinder axis.
func TestArmSectionArcSpansFilletQuarterCircle(t *testing.T) {
	t.Parallel()
	ef, _ := runoutFixtureCrossingBoss(t)
	planeA, _ := geom.NewPlane(math.P3(0, -10, 0), math.V3(0, 1, 0))
	planeB, _ := geom.NewPlane(math.P3(0, 0, 10), math.V3(0, 0, 1))
	arc, ok := armSectionArc(ef.cyl, planeA, planeB, 16.928203230275503)
	if !ok {
		t.Fatal("armSectionArc declined the +x fillet-cut station")
	}
	assertPointNear(t, "start", arc.PointAt(0), math.P3(6.928203230275509, -10, 4), 1e-6)
	assertPointNear(t, "end", arc.PointAt(1), math.P3(6.928203230275509, -4, 10), 1e-6)
	axisPt := math.P3(6.928203230275503, -4, 4)
	for _, tt := range []float64{0, 0.25, 0.5, 0.75, 1} {
		if d := axisPt.DistanceTo(arc.PointAt(tt)); mabs(d-6) > 1e-6 {
			t.Errorf("arc point t=%v is %v from the axis, want radius 6", tt, d)
		}
	}
}

// TestInternalSeamConnectsFeatureCorners unit-tests the seam helper: a straight segment between a
// seam-bottom and a seam-top, with the endpoints preserved exactly (bit-identical) so the two
// flanking setback patches can share it.
func TestInternalSeamConnectsFeatureCorners(t *testing.T) {
	t.Parallel()
	bottom := math.P3(3.465, -10, 4.898)
	top := math.P3(3.465, -7.211, 10)
	seam := internalSeam(bottom, top)
	if seam.PointAt(0) != bottom || seam.PointAt(1) != top {
		t.Errorf("internalSeam endpoints changed: got %v..%v, want %v..%v",
			seam.PointAt(0), seam.PointAt(1), bottom, top)
	}
}

func mabs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func assertPointNear(t *testing.T, label string, got, want math.Point3, tol float64) {
	t.Helper()
	if d := got.DistanceTo(want); d > tol {
		t.Errorf("%s: got %v, want %v (dist %v > %v)", label, got, want, d, tol)
	}
}
