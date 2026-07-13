// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

func TestObstacleProviderName(t *testing.T) {
	var p bsplineObstacleProvider
	if got := p.Name(); got != BlendKindBSpline {
		t.Errorf("obstacle provider Name() = %q, want %q", got, BlendKindBSpline)
	}
}

func TestObstacleRequestNilByDefault(t *testing.T) {
	req := CornerBlendRequest{}
	if req.ObstacleFeature != nil {
		t.Errorf("a default CornerBlendRequest must carry no ObstacleFeature (junction request unchanged)")
	}
}

func TestObstacleRailsRejectNilWing(t *testing.T) {
	of := &ObstacleFeature{ /* WingStart/WingEnd left nil */ }
	if _, _, _, _, ok := obstacleRails(of); ok {
		t.Errorf("obstacleRails must reject a nil wing pointer (regression-crack defense)")
	}
}

func TestObstacleSidesContinuityOrders(t *testing.T) {
	sides := obstacleSides(&ObstacleFeature{}, geom.BSplineSurface{}, geom.BSplineSurface{}, geom.BSplineSurface{})
	// side order: c0(v=0)=wall, c1(v=1)=rim, d0(u=0)=wingL, d1(u=1)=wingR
	want := [4]int{1, 0, 1, 1}
	for i, s := range sides {
		if s.Order != want[i] {
			t.Errorf("side %d Order = %d, want %d (rim must be G0)", i, s.Order, want[i])
		}
	}
}

// newT6Obstacle builds a real T6-shaped ObstacleFeature (mid-span obstacle on a box with a
// vertical wall + two quarter-circle wing arcs + an elliptical base rim): WingStart/WingEnd are
// the exact section arcs of the abutting cylinder wings at the Nodes (radius 6, quarter-circle),
// WallLine is the wall-tangent seam A->D, and RimArcPts are 13 samples of the base ellipse
// (a=15, b=10) lower arc from P- through its bottom (0,-10,0) to P+ — the shape task-3-brief
// specifies for exercising the real rail converters end to end. Reused by Task 4's provider tests.
func newT6Obstacle(t *testing.T) *ObstacleFeature {
	t.Helper()
	a := math.P3(-10.712142, -13, -6)
	d := math.P3(10.712142, -13, -6)
	pMinus := math.P3(-10.712142, -7, 0)
	pPlus := math.P3(10.712142, -7, 0)
	return &ObstacleFeature{
		Nodes:     [2]math.Point3{pMinus, pPlus},
		WingStart: quarterWingArc(t, a, pMinus),
		WingEnd:   quarterWingArc(t, d, pPlus),
		WallLine:  geom.NewLineSegment(a, d),
		RimArcPts: ellipseLowerArcSamples(pMinus, pPlus, 15, 10, 13),
	}
}

// quarterWingArc builds the exact quarter-circle (radius 6) cylinder-wing section arc from wall
// point wallPt to node nodePt (T6's wing cylinders are axis-vertical, X constant across both
// points): the center sits at (wallPt.X, wallPt.Y, nodePt.Z), so with refDir = center->wallPt and
// a +90 deg sweep about the X normal, PointAt(0)=wallPt and PointAt(1)=nodePt exactly.
func quarterWingArc(t *testing.T, wallPt, nodePt math.Point3) geom.Arc3d {
	t.Helper()
	center := math.P3(wallPt.X, wallPt.Y, nodePt.Z)
	normal := math.V3(1, 0, 0)
	refDir := center.VectorTo(wallPt)
	arc, err := geom.NewArc3d(center, normal, refDir, 6, 0, stdmath.Pi/2)
	if err != nil {
		t.Fatalf("quarterWingArc: NewArc3d: %v", err)
	}
	return arc
}

// ellipseLowerArcSamples returns n ordered points from p0 through the ellipse's bottom to p1,
// tracing the (a, b) ellipse's lower half (x = a*cos(theta), y = -b*sin(theta)) — the base rim's
// dip-side arc that RimArcPts (task 6) is documented to hold. The first and last samples are PINNED
// to the exact p0/p1 the caller passes: in the real pipeline the detector computes the Nodes AND
// the rim samples from the SAME crossing, so Nodes[i] == RimArcPts[endpoint] to machine precision —
// the model-relative corner weld (ADR-0042) relies on that self-consistency, which a trig
// round-trip of a truncated node literal would otherwise break by ~4e-7.
func ellipseLowerArcSamples(p0, p1 math.Point3, a, b float64, n int) []math.Point3 {
	theta0 := stdmath.Acos(p0.X / a)
	theta1 := stdmath.Acos(p1.X / a)
	pts := make([]math.Point3, n)
	for i := 0; i < n; i++ {
		f := float64(i) / float64(n-1)
		theta := theta0 + f*(theta1-theta0)
		pts[i] = math.P3(a*stdmath.Cos(theta), -b*stdmath.Sin(theta), 0)
	}
	pts[0], pts[n-1] = p0, p1
	return pts
}

// TestObstacleRailsBuildT6 exercises the real converters (asBSplineCurve on the wing Arc3ds and the
// wall LineSegment, obstacleRimArc's fit) on a genuine T6 obstacle, not just the nil-check path —
// the corner-exactness contract is only proven by handing the four rails to the real CoonsFill
// precondition (1e-9 corner match), the exact check FillSurface itself performs.
func TestObstacleRailsBuildT6(t *testing.T) {
	of := newT6Obstacle(t)
	c0, c1, d0, d1, ok := obstacleRails(of)
	if !ok {
		t.Fatalf("obstacleRails(T6) ok = false, want true")
	}
	for name, c := range map[string]geom.BSplineCurve{"c0": c0, "c1": c1, "d0": d0, "d1": d1} {
		if len(c.Ctrl) == 0 {
			t.Errorf("rail %s has no control points", name)
		}
	}
	if _, err := geom.CoonsFill(c0, c1, d0, d1); err != nil {
		t.Errorf("CoonsFill(rails) = %v, want no error (corners must meet within 1e-9)", err)
	}
}

// TestReverseBSplineCurve exercises reverseBSplineCurve's INTERIOR directly — the reversed control
// points, weights, and reflected knots — WITHOUT the endpoint pinning (pinEnds) that would mask an
// interior bug in TestObstacleRailsBuildT6 (which only checks corners, and pinning fixes those
// regardless). Uses a genuinely asymmetric degree-2 RATIONAL curve (distinct control points,
// non-uniform weights) so a wrong reversal cannot pass by coincidence.
func TestReverseBSplineCurve(t *testing.T) {
	orig, err := geom.NewBSplineCurve(2,
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 2, 0), math.P3(3, 1, 0)},
		[]float64{1, 2, 1},
		[]float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("build original: %v", err)
	}
	rev, ok := reverseBSplineCurve(orig)
	if !ok {
		t.Fatalf("reverseBSplineCurve rejected a valid rational curve")
	}
	assertReversedTraces(t, orig, rev)
	assertReversedNet(t, orig, rev)
}

// assertReversedTraces checks the geometric reversal: reversed.PointAt(t) == original.PointAt(1-t)
// at several interior parameters (the rational blend, weights included, must match end to end).
func assertReversedTraces(t *testing.T, orig, rev geom.BSplineCurve) {
	t.Helper()
	for _, u := range []float64{0.1, 0.25, 0.5, 0.75, 0.9} {
		want := orig.PointAt(1 - u)
		got := rev.PointAt(u)
		if !got.IsEqualTo(want, 1e-9) {
			t.Errorf("rev.PointAt(%.2f) = %v, want orig.PointAt(%.2f) = %v", u, got, 1-u, want)
		}
	}
}

// assertReversedNet checks the representation: control points and weights order-reversed, knots
// reflected about the domain (lo+hi-k).
func assertReversedNet(t *testing.T, orig, rev geom.BSplineCurve) {
	t.Helper()
	n := len(orig.Ctrl)
	for i := 0; i < n; i++ {
		if !rev.Ctrl[i].IsEqualTo(orig.Ctrl[n-1-i], 1e-12) {
			t.Errorf("rev.Ctrl[%d] = %v, want %v", i, rev.Ctrl[i], orig.Ctrl[n-1-i])
		}
		if stdmath.Abs(rev.Weights[i]-orig.Weights[n-1-i]) > 1e-12 {
			t.Errorf("rev.Weights[%d] = %v, want %v", i, rev.Weights[i], orig.Weights[n-1-i])
		}
	}
	lo, hi := orig.Knots[0], orig.Knots[len(orig.Knots)-1]
	m := len(orig.Knots)
	for i, k := range orig.Knots {
		if stdmath.Abs(rev.Knots[m-1-i]-(lo+hi-k)) > 1e-12 {
			t.Errorf("rev.Knots[%d] = %v, want %v", m-1-i, rev.Knots[m-1-i], lo+hi-k)
		}
	}
}
