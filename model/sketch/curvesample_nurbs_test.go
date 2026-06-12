// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
)

// TestClosedFitSplineAreaApproachesCircle: a closed fit spline through n
// points of a circle must enclose nearly the circle's area — the
// interpolating NURBS bellies *outward* between points, where the inscribed
// Catmull–Rom polygon it replaced systematically undershot (M06-F12, #627).
func TestClosedFitSplineAreaApproachesCircle(t *testing.T) {
	const n, r = 8, 2.0
	pts := make([]gmath.Point2, n)
	for i := range pts {
		a := 2 * math.Pi * float64(i) / n
		pts[i] = gmath.P2(gmath.Scalar(r*math.Cos(a)), gmath.Scalar(r*math.Sin(a)))
	}
	s := NewSketches().Add(XYPlane())
	sp := s.Splines().AddByPoints(pts, true)

	area := math.Abs(polygonArea(sampleSplineEntity(sp)))
	circle := math.Pi * r * r
	if rel := math.Abs(area-circle) / circle; rel > 0.01 {
		t.Errorf("sampled area = %.5f, want within 1%% of πr² = %.5f (rel %.4f)", area, circle, rel)
	}
	inscribed := float64(n) / 2 * r * r * math.Sin(2*math.Pi/n)
	if area <= inscribed {
		t.Errorf("sampled area %.5f did not improve on the inscribed polygon %.5f", area, inscribed)
	}
}

// TestSplineSamplePassesThroughFitPoints: every defining point appears
// exactly in the sampled polyline (region detection treats them as vertices).
func TestSplineSamplePassesThroughFitPoints(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	defining := []gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 2), gmath.P2(3, 1), gmath.P2(4, -1)}
	sp := s.Splines().AddByPoints(defining, false)
	sampled := sampleSplineEntity(sp)
	for _, want := range defining {
		found := false
		for _, p := range sampled {
			if float64(p.DistanceTo(want)) < 1e-12 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("defining point %v missing from the sampled polyline", want)
		}
	}
}

// TestFitMethodChangesSampledCurve: the three public fit methods produce
// different curves between unevenly spaced points — no dead setting.
func TestFitMethodChangesSampledCurve(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	defining := []gmath.Point2{gmath.P2(0, 0), gmath.P2(0.2, 1), gmath.P2(3, 1.2), gmath.P2(4, 0)}
	sp := s.Splines().AddByPoints(defining, false)

	smooth := append([]gmath.Point2(nil), sampleSplineEntity(sp)...)
	sp.FitMethod = types.SplineFitChord
	chord := sampleSplineEntity(sp)
	maxGap := 0.0
	for i := range smooth {
		if d := float64(smooth[i].DistanceTo(chord[i])); d > maxGap {
			maxGap = d
		}
	}
	if maxGap < 1e-6 {
		t.Errorf("smooth and chord fits coincide (max gap %g); the fit method is a dead setting", maxGap)
	}
}

// TestControlSplineStaysInHull: a control-point spline approximates — it must
// not pass through its middle control point but must stay inside the hull.
func TestControlSplineStaysInHull(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	sp := s.Splines().AddByControlPoints([]gmath.Point2{
		gmath.P2(0, 0), gmath.P2(1, 2), gmath.P2(2, 0),
	}, false)
	sampled := sampleSplineEntity(sp)
	if first, last := sampled[0], sampled[len(sampled)-1]; float64(first.DistanceTo(gmath.P2(0, 0))) > 1e-9 ||
		float64(last.DistanceTo(gmath.P2(2, 0))) > 1e-9 {
		t.Errorf("control spline ends = %v→%v, want clamped to the outer control points", first, last)
	}
	for _, p := range sampled {
		if float64(p.Y) > 2+1e-9 || float64(p.Y) < -1e-9 {
			t.Errorf("sample %v escapes the control hull", p)
		}
		if float64(p.DistanceTo(gmath.P2(1, 2))) < 1e-6 {
			t.Errorf("control spline passes through its middle control point %v", p)
		}
	}
}

// TestPathPointsFollowSplineRail: a path containing a spline samples the
// curve into the rail polyline instead of collapsing it to a chord
// (M06-F12, #627; registry row "path.go" rail fidelity).
func TestPathPointsFollowSplineRail(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.Splines().AddByPoints([]gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 1), gmath.P2(2, 0)}, false)
	paths := s.Paths()
	if len(paths) != 1 {
		t.Fatalf("paths = %d, want the single spline chain", len(paths))
	}
	pts := paths[0].Points()
	if len(pts) <= 3 {
		t.Fatalf("rail polyline has %d points — the spline collapsed to a chord", len(pts))
	}
	apexSeen := false
	for _, p := range pts {
		if float64(p.DistanceTo(gmath.P2(1, 1))) < 1e-9 {
			apexSeen = true
		}
	}
	if !apexSeen {
		t.Error("rail polyline misses the spline's apex fit point")
	}
}
