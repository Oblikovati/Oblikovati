// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// The march's oracle here is the closed-form boss rim: cylinder radius R about z, cap
// plane z = H, ball radius r rolling inside the material. Every station centre must sit on
// the circle radius R−r at height H−r, with feet at (R, ·, H−r) on the wall and (R−r, ·, H)
// on the cap — exact geometry the numeric march must reproduce to station tolerance.

const (
	marchTestR      = 10.0
	marchTestBall   = 2.0
	marchTestH      = 20.0
	marchTestWeld   = 1e-9 * 60 // ResolutionForSize(60).Weld() scale: model-relative, not bare
	marchTestNStats = 33
)

func marchTestHosts(t *testing.T) (CanalMarchHost, CanalMarchHost) {
	t.Helper()
	cyl, err := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), marchTestR)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	pl, err := NewPlane(math.P3(0, 0, marchTestH), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	return CanalMarchHost{Surf: cyl, PeriodU: 2 * stdmath.Pi}, CanalMarchHost{Surf: pl}
}

func marchTestAnchors(n int, a0, a1 float64) []CanalEdgeAnchor {
	anchors := make([]CanalEdgeAnchor, n)
	for k := range n {
		a := a0 + (a1-a0)*float64(k)/float64(n-1)
		anchors[k] = CanalEdgeAnchor{
			P: math.P3(marchTestR*stdmath.Cos(a), marchTestR*stdmath.Sin(a), marchTestH),
			T: math.V3(-stdmath.Sin(a), stdmath.Cos(a), 0),
		}
	}
	return anchors
}

// TestMarchCanalEdgeStationsBossRim checks every marched station against the closed form,
// including a march across the cylinder's u-seam (anchors span 1.5π→2.5π on the cover).
func TestMarchCanalEdgeStationsBossRim(t *testing.T) {
	wall, cap := marchTestHosts(t)
	anchors := marchTestAnchors(marchTestNStats, 1.5*stdmath.Pi, 2.5*stdmath.Pi)
	c0 := math.P3(0, -(marchTestR - marchTestBall), marchTestH-marchTestBall) // closed form at a=1.5π
	sts, err := MarchCanalEdgeStations(wall, cap, marchTestBall, anchors, c0, marchTestWeld)
	if err != nil {
		t.Fatalf("march: %v", err)
	}
	for k, st := range sts {
		assertBossStation(t, k, st)
	}
}

func assertBossStation(t *testing.T, k int, st CanalEdgeStation) {
	t.Helper()
	rc := stdmath.Hypot(float64(st.Center.X), float64(st.Center.Y))
	if stdmath.Abs(rc-(marchTestR-marchTestBall)) > 1e-8 || stdmath.Abs(float64(st.Center.Z)-(marchTestH-marchTestBall)) > 1e-8 {
		t.Fatalf("station %d centre %v off closed form (radial %g, want %g; z want %g)",
			k, st.Center, rc, marchTestR-marchTestBall, marchTestH-marchTestBall)
	}
	rw := stdmath.Hypot(float64(st.FootA.P.X), float64(st.FootA.P.Y))
	if stdmath.Abs(rw-marchTestR) > 1e-8 || stdmath.Abs(float64(st.FootA.P.Z)-(marchTestH-marchTestBall)) > 1e-8 {
		t.Fatalf("station %d wall foot %v off closed form", k, st.FootA.P)
	}
	if stdmath.Abs(float64(st.FootB.P.Z)-marchTestH) > 1e-8 {
		t.Fatalf("station %d cap foot %v off cap plane", k, st.FootB.P)
	}
}

// TestMarchCanalEdgeStationsSeamUnwrap proves the universal-cover lift: a march whose
// anchors cross the seam keeps consecutive wall-foot u values within the station spacing —
// no wrap-induced branch jump.
func TestMarchCanalEdgeStationsSeamUnwrap(t *testing.T) {
	wall, cap := marchTestHosts(t)
	anchors := marchTestAnchors(marchTestNStats, 1.5*stdmath.Pi, 2.5*stdmath.Pi)
	c0 := math.P3(0, -(marchTestR - marchTestBall), marchTestH-marchTestBall)
	sts, err := MarchCanalEdgeStations(wall, cap, marchTestBall, anchors, c0, marchTestWeld)
	if err != nil {
		t.Fatalf("march: %v", err)
	}
	step := stdmath.Pi / float64(marchTestNStats-1)
	for k := 1; k < len(sts); k++ {
		if du := stdmath.Abs(sts[k].FootA.U - sts[k-1].FootA.U); du > 3*step {
			t.Fatalf("station %d wall-foot u jumped %g (lifted step should be ~%g): seam wrap leaked into the march", k, du, step)
		}
	}
}

// TestMarchCanalBallClearanceRefusal proves the evolute guard RED the falsifiable way: a
// ball larger than the bore it rolls in (concave side, r > R) must refuse, never march.
func TestMarchCanalBallClearanceRefusal(t *testing.T) {
	wall, cap := marchTestHosts(t)
	r := marchTestR * 1.5 // ball cannot fit tangent inside the R-bore
	anchors := marchTestAnchors(5, 0, 0.25*stdmath.Pi)
	c0 := math.P3(marchTestR-r, 0, marchTestH-r) // inside past the axis: bore-side seed
	if _, err := MarchCanalEdgeStations(wall, cap, r, anchors, c0, marchTestWeld); err == nil {
		t.Fatal("march accepted a ball of radius 15 rolling inside a radius-10 bore (evolute violation)")
	}
}

// TestSubSpanBSplineCurveExact proves the span extraction is the parent's own geometry:
// sampled positions agree to float noise across the span, including the exact endpoints.
func TestSubSpanBSplineCurveExact(t *testing.T) {
	pts := []math.Point3{
		math.P3(0, 0, 0), math.P3(3, 4, 1), math.P3(7, 2, 5), math.P3(10, 8, 2), math.P3(14, 6, 9),
	}
	parent, err := NewFittedBSplineCurve(pts)
	if err != nil {
		t.Fatalf("fit: %v", err)
	}
	lo, hi := parent.Domain()
	t0, t1 := lo+0.17*(hi-lo), lo+0.83*(hi-lo)
	sub, err := SubSpanBSplineCurve(parent, t0, t1)
	if err != nil {
		t.Fatalf("subspan: %v", err)
	}
	assertSpanMatchesParent(t, parent, sub, t0, t1)
}

func assertSpanMatchesParent(t *testing.T, parent, sub BSplineCurve, t0, t1 float64) {
	t.Helper()
	slo, shi := sub.Domain()
	if stdmath.Abs(slo-t0) > 1e-12 || stdmath.Abs(shi-t1) > 1e-12 {
		t.Fatalf("sub domain [%g, %g], want [%g, %g]", slo, shi, t0, t1)
	}
	for k := 0; k <= 64; k++ {
		u := t0 + (t1-t0)*float64(k)/64
		if d := float64(sub.PointAt(u).DistanceTo(parent.PointAt(u))); d > 1e-10 {
			t.Fatalf("sub deviates %g from parent at %g", d, u)
		}
	}
}

// TestSubSpanBSplineCurveAtKnot splits exactly AT an interior knot (the multiplicity-
// bookkeeping edge case) and at a domain end.
func TestSubSpanBSplineCurveAtKnot(t *testing.T) {
	pts := []math.Point3{
		math.P3(0, 0, 0), math.P3(2, 5, 0), math.P3(5, 1, 3), math.P3(9, 4, 1), math.P3(12, 0, 6), math.P3(15, 3, 2),
	}
	parent, err := NewFittedBSplineCurve(pts)
	if err != nil {
		t.Fatalf("fit: %v", err)
	}
	lo, hi := parent.Domain()
	knot := interiorKnotOf(parent, lo, hi)
	sub, err := SubSpanBSplineCurve(parent, lo, knot)
	if err != nil {
		t.Fatalf("subspan to interior knot: %v", err)
	}
	assertSpanMatchesParent(t, parent, sub, lo, knot)
}

func interiorKnotOf(c BSplineCurve, lo, hi float64) float64 {
	for _, k := range c.Knots {
		if k > lo+1e-9 && k < hi-1e-9 {
			return k
		}
	}
	return (lo + hi) / 2
}
