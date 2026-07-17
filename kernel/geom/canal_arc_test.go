// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/math"
)

// crossSectionArc ground truth (blend-sweep-spike-report.md / result5-poles.txt): the N7 corner's
// v=0 cross-section (rail E1) is the radius-5 quarter-circle at center C″=(55, 5.27864, 5) between
// the wall foot V0=(55.5556, 0.30960, 5) and the s_10 foot V1=(55, 5.27864, 10). Its shoulder is the
// poles' row-2 col-1 control (55.5556, 0.30960, 10), weight cos 45° = 0.70711.
func TestCrossSectionArcN7IsRadius5QuarterCircle(t *testing.T) {
	wall, s10 := n7Hosts()
	cPP := n7Ends()[0] // C″, the v=0 spine station
	_, _, fa := ClosestPointOnSurface(wall, cPP)
	_, _, fb := ClosestPointOnSurface(s10, cPP)
	res := n7Resolution(n7Ends())
	tol := res.Weld() * 5 // res.Weld·r — the plan's arc radius tolerance

	arc, err := crossSectionArc(cPP, fa, fb, 5)
	if err != nil {
		t.Fatalf("crossSectionArc: unexpected error: %v", err)
	}
	assertArcRadius(t, arc, cPP, 5, tol)
	assertArcEndpoints(t, arc, fa, fb, tol)
}

// The emitted shoulder (rational-quadratic middle control) and weight must be OCCT's row-2 col-1
// pole exactly — the parametrization-free proof the arc is the SAME conic OCCT built, not merely a
// radius-5 arc that happens to share the endpoints.
func TestCrossSectionArcShoulderMatchesOCCTPole(t *testing.T) {
	wall, s10 := n7Hosts()
	cPP := n7Ends()[0]
	_, _, fa := ClosestPointOnSurface(wall, cPP)
	_, _, fb := ClosestPointOnSurface(s10, cPP)

	shoulder, weight, err := arcControls(cPP, fa, fb, 5)
	if err != nil {
		t.Fatalf("arcControls: %v", err)
	}
	wantShoulder := math.P3(55.5555555555556, 0.309600500004678, 10)
	if d := float64(shoulder.DistanceTo(wantShoulder)); d > 1e-6 {
		t.Errorf("shoulder %v != OCCT pole %v (dist %g)", shoulder, wantShoulder, d)
	}
	if stdmath.Abs(weight-0.707106781186548) > 1e-9 {
		t.Errorf("weight %g != cos45° pole weight 0.70711", weight)
	}
}

// Collinear feet+center (here antipodal: fa, fb on opposite sides of m) have no radius arc plane —
// crossSectionArc must reject, and the error must carry the half-angle it measured.
func TestCrossSectionArcRejectsCollinearFeet(t *testing.T) {
	m := math.P3(0, 0, 0)
	fa := math.P3(5, 0, 0)
	fb := math.P3(-5, 0, 0) // antipodal → half-angle π/2, weight → 0
	_, err := crossSectionArc(m, fa, fb, 5)
	if err == nil {
		t.Fatal("crossSectionArc accepted antipodal (collinear) feet; want reject")
	}
	if !strings.Contains(err.Error(), "collinear") {
		t.Errorf("collinear error should name the condition, got: %v", err)
	}
}

// assertArcRadius samples the arc and asserts every point is `radius` from center to tol.
func assertArcRadius(t *testing.T, arc Curve3, center math.Point3, radius, tol float64) {
	t.Helper()
	lo, hi := arc.Domain()
	maxResid := 0.0
	const samples = 64
	for i := 0; i <= samples; i++ {
		p := arc.PointAt(lo + (hi-lo)*float64(i)/samples)
		resid := stdmath.Abs(float64(p.DistanceTo(center)) - radius)
		if resid > maxResid {
			maxResid = resid
		}
		if resid > tol {
			t.Errorf("arc point %v: |dist(center)=%g - %g| = %g > tol %g",
				p, float64(p.DistanceTo(center)), radius, resid, tol)
		}
	}
	t.Logf("crossSectionArc max radius residual = %g (tol %g)", maxResid, tol)
}

// assertArcEndpoints asserts the arc runs fa (u=0) → fb (u=1).
func assertArcEndpoints(t *testing.T, arc Curve3, fa, fb math.Point3, tol float64) {
	t.Helper()
	lo, hi := arc.Domain()
	if d := float64(arc.PointAt(lo).DistanceTo(fa)); d > tol {
		t.Errorf("arc start %v != fa %v (dist %g > tol %g)", arc.PointAt(lo), fa, d, tol)
	}
	if d := float64(arc.PointAt(hi).DistanceTo(fb)); d > tol {
		t.Errorf("arc end %v != fb %v (dist %g > tol %g)", arc.PointAt(hi), fb, d, tol)
	}
}
