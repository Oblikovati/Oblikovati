// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

func TestObstacleProviderName(t *testing.T) {
	t.Parallel()
	var p bsplineObstacleProvider
	if got := p.Name(); got != BlendKindBSpline {
		t.Errorf("obstacle provider Name() = %q, want %q", got, BlendKindBSpline)
	}
}

func TestObstacleRequestNilByDefault(t *testing.T) {
	t.Parallel()
	req := CornerBlendRequest{}
	if req.ObstacleFeature != nil {
		t.Errorf("a default CornerBlendRequest must carry no ObstacleFeature (junction request unchanged)")
	}
}

func TestObstacleRailsRejectNilWing(t *testing.T) {
	t.Parallel()
	of := &ObstacleFeature{ /* WingStart/WingEnd left nil */ }
	if _, _, _, _, ok := obstacleRails(of); ok {
		t.Errorf("obstacleRails must reject a nil wing pointer (regression-crack defense)")
	}
}

func TestObstacleSidesContinuityOrders(t *testing.T) {
	t.Parallel()
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
		Radius:    6,                 // rolling-ball blend radius → the G1-ribbon length
		BlendAxis: math.V3(1, 0, 0),  // fillet-cylinder axis: the wings extrude along ±X
		WallInto:  math.V3(0, 0, -1), // in the wall plane, ⟂ the bottom rail, pointing down into the wall
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
	for i := range n {
		f := float64(i) / float64(n-1)
		theta := theta0 + f*(theta1-theta0)
		pts[i] = math.P3(a*stdmath.Cos(theta), -b*stdmath.Sin(theta), 0)
	}
	pts[0], pts[n-1] = p0, p1
	return pts
}

// newFoldingObstacle returns a T6 obstacle whose WallInto is set PARALLEL to the bottom rail (c0 runs
// along +X, so WallInto=+X instead of the required in-plane ⟂ direction). This is a real geometric
// degeneracy, not a random perturbation: the wall's G1 tangent ribbon (extrudeRibbon(c0, WallInto·len))
// then extrudes ALONG the seam, collapsing to a zero-area strip whose normal vanishes. FillSurface
// still produces a surface (Build ok=true), but matching to that degenerate ribbon drives the fill's
// S_u and S_v parallel at the wall seam, so |S_u×S_v|→0 there — a genuine fold the anti-fold gate must
// reject. It exercises the CERTIFICATE (NoFold), not just the rail nil-checks.
func newFoldingObstacle(t *testing.T) *ObstacleFeature {
	of := newT6Obstacle(t)
	of.WallInto = math.V3(1, 0, 0) // parallel to c0 (the bottom rail runs along +X) ⇒ degenerate wall ribbon
	return of
}

// TestObstacleCertificateDeclinesPathological proves ADR-3 honest-reject at the CERTIFICATE level: a
// patch whose wall tangent ribbon is degenerate (folding the fill) is DECLINED — Build produces the
// surface, but cert.NoFold is false and cert.Valid is false, so resolveCornerBlend passes this tier
// over. This is the safety net for a malformed detector output (a wrong WallInto direction), the whole
// reason the certificate exists rather than trusting "the code ran".
func TestObstacleCertificateDeclinesPathological(t *testing.T) {
	t.Parallel()
	of := newFoldingObstacle(t)
	req := CornerBlendRequest{ObstacleFeature: of, Setback: tol.ForSize(50)}
	var p bsplineObstacleProvider
	_, cert, ok := p.Build(req)
	if !ok {
		t.Fatal("Build should still PRODUCE a patch (the decline is the certificate's job, not Build's)")
	}
	if cert.NoFold {
		t.Errorf("degenerate wall ribbon folds the fill; cert.NoFold must be false, got cert=%+v", cert)
	}
	if cert.Valid(req.Setback) {
		t.Errorf("a folded patch must not certify Valid (honest-reject); got cert=%+v", cert)
	}
}

// TestObstacleRailsBuildT6 exercises the real converters (asBSplineCurve on the wing Arc3ds and the
// wall LineSegment, obstacleRimArc's fit) on a genuine T6 obstacle, not just the nil-check path —
// the corner-exactness contract is only proven by handing the four rails to the real CoonsFill
// precondition (1e-9 corner match), the exact check FillSurface itself performs.
func TestObstacleRailsBuildT6(t *testing.T) {
	t.Parallel()
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

// TestObstacleBuildT6AreaAndCert is the crux test (spec §3): the obstacle provider Fits an obstacle
// request, Builds a 4-sided FillSurface with G1 wall+wing ribbons, and the certificate is Valid — in
// particular NoFold true (the fillet normal sweeps ~90°, |S_u×S_v| stays ~137..212, never 0). The
// patch area is asserted only within a GENEROUS band of the OCCT oracle 156.364 (the binding 1% gate
// is body-level in Task 7; G1 ribbons shift the area off the advisor's bilinear ~178 estimate).
func TestObstacleBuildT6AreaAndCert(t *testing.T) {
	t.Parallel()
	of := newT6Obstacle(t)
	req := CornerBlendRequest{ObstacleFeature: of, Setback: tol.ForSize(50)}
	var p bsplineObstacleProvider
	if !p.Fits(req) {
		t.Fatal("provider must Fit an obstacle request")
	}
	patch, cert, ok := p.Build(req)
	if !ok {
		t.Fatal("Build failed on a valid T6 obstacle")
	}
	if !cert.NoFold {
		t.Errorf("T6 patch must not fold (|S_u×S_v| stays ~137..212, never 0); cert=%+v", cert)
	}
	if !cert.Valid(req.Setback) {
		t.Errorf("T6 patch certificate must be Valid, got %+v", cert)
	}
	bs, isBS := patch.Surface.(geom.BSplineSurface)
	if !isBS {
		t.Fatalf("patch surface must be a BSplineSurface, got %T", patch.Surface)
	}
	area := surfaceArea(bs)
	rel := stdmath.Abs(area-156.364) / 156.364
	t.Logf("T6 obstacle patch area = %.4f (oracle 156.364, rel %.2f%%); cert=%+v", area, rel*100, cert)
	if stdmath.IsInf(area, 0) || stdmath.IsNaN(area) || rel > 0.20 {
		t.Errorf("patch area %.4f vs oracle 156.364 (rel %.4f > 0.20)", area, rel)
	}
}

// surfaceArea is a fine-midpoint (40×40) quadrature of ∬|S_u×S_v| du dv over the surface domain — a
// test-only integrator for the patch-area parity check (NOT model code; kept in the test file).
func surfaceArea(s geom.BSplineSurface) float64 {
	u0, u1 := s.UDomain()
	v0, v1 := s.VDomain()
	const n = 40
	du, dv := (u1-u0)/n, (v1-v0)/n
	area := 0.0
	for i := range n {
		for j := range n {
			u := u0 + (float64(i)+0.5)*du
			v := v0 + (float64(j)+0.5)*dv
			pu, pv := s.DerivativesAt(u, v)
			area += pu.Cross(pv).Length() * du * dv
		}
	}
	return area
}

// TestObstacleT6RibbonNonFolding proves the sign-corrected obstacle patch passes the F2 probe.
// Before the fix this FAILS on the wall seam — that failure is the regression witness the report
// predicts (f2-reconciliation-report.md §C, "before the flip this assertion is expected to FAIL").
func TestObstacleT6RibbonNonFolding(t *testing.T) {
	t.Parallel()
	of := newT6Obstacle(t)
	g, ok := obstaclePatchNeighbours(of)
	if !ok {
		t.Fatal("obstaclePatchNeighbours declined T6")
	}
	sides := obstacleSides(of, g.wingL, g.wingR, g.wall)
	rails := [4]geom.BSplineCurve{g.c0, g.c1, g.d0, g.d1}
	fill, err := geom.FillSurface(g.c0, g.c1, g.d0, g.d1, sides)
	if err != nil {
		t.Fatalf("FillSurface: %v", err)
	}
	fill, err = pinFillBoundary(fill, g.c0, g.c1, g.d0, g.d1)
	if err != nil {
		t.Fatalf("pinFillBoundary: %v", err)
	}
	if !ribbonSeamNonFolding(fill, rails, sides, blendScale()) {
		t.Fatal("sign-corrected obstacle T6 patch still folds")
	}
}

// TestReverseBSplineCurve exercises reverseBSplineCurve's INTERIOR directly — the reversed control
// points, weights, and reflected knots — WITHOUT the endpoint pinning (pinEnds) that would mask an
// interior bug in TestObstacleRailsBuildT6 (which only checks corners, and pinning fixes those
// regardless). Uses a genuinely asymmetric degree-2 RATIONAL curve (distinct control points AND
// NON-PALINDROMIC weights {1,2,3}) so a wrong reversal cannot pass by coincidence — palindromic
// weights would leave a "forgot to reverse Weights" bug undetected (the rational blend is unchanged
// by reversing a symmetric weight vector).
func TestReverseBSplineCurve(t *testing.T) {
	t.Parallel()
	orig, err := geom.NewBSplineCurve(2,
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 2, 0), math.P3(3, 1, 0)},
		[]float64{1, 2, 3},
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
	for i := range n {
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
