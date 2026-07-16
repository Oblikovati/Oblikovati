// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The shared test bed for the M5 curved-arm trihedral weld (T5.2–T5.4). It reuses b3CornerArms
// (fillet_curved_corner_solve_test.go) — the oracle-closed B3 corner from
// m5-weld-setback-retrim-derivation.md — and adds the three certified host-tangent points, the
// spherical-triangle vertices every weld rail must connect.

// b3TangentPoints returns the three certified host-tangent points T_W, T_K, T_N (derivation §A.2),
// kept exact via b3CornerCY=−√1500 so the rail endpoints hit them to machine precision:
//   - T_W = radial projection of C onto the wall R=50 (C_xy scaled 50/40), z=90;
//   - T_K = C + r·ẑ (foot on the cap z=100);
//   - T_N = foot of C onto the radial plane x=0.
func b3TangentPoints() (tW, tK, tN math.Point3) {
	tW = math.P3(12.5, 1.25*b3CornerCY, 90) // 1.25·C_xy: C_xy has radius 40, wall has radius 50
	tK = math.P3(10, b3CornerCY, 100)
	tN = math.P3(0, b3CornerCY, 90)
	return tW, tK, tN
}

// railOracle returns the two certified endpoints and the certified subtense of one arm's weld rail,
// discriminating the arms by surface type and axis: torus W∧K (90°), vertical cyl W∧N
// (arccos(−0.25)=104.478°), planar cyl K∧N (90°).
func railOracle(t *testing.T, a armSetback, tW, tK, tN math.Point3) ([2]math.Point3, float64) {
	t.Helper()
	switch s := a.arm.(type) {
	case geom.Torus:
		return [2]math.Point3{tW, tK}, stdmath.Pi / 2
	case geom.Cylinder:
		if stdmath.Abs(float64(s.AxisDir.Z())-1) < 0.5 { // ẑ axis → vertical W∧N arm
			return [2]math.Point3{tW, tN}, stdmath.Acos(-0.25)
		}
		return [2]math.Point3{tK, tN}, stdmath.Pi / 2 // ŷ axis → planar K∧N arm
	default:
		t.Fatalf("unexpected arm surface %T", a.arm)
		return [2]math.Point3{}, 0
	}
}

// TestCurvedSetbackRail_B3 drives the setback-rail constructor for all three B3 arms: each rail must
// be the corner sphere's great-circle arc (centre C, radius r, its plane through C) joining that
// arm's two certified host-tangent points, with the certified subtense (torus/planar 90°, cyl
// 104.478°). This is the T5.2 §A.2 weld rail.
func TestCurvedSetbackRail_B3(t *testing.T) {
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner (want ok)")
	}
	tW, tK, tN := b3TangentPoints()
	tol := res.Weld() * sphere.Radius
	for _, a := range w.arms {
		wantEnds, wantSubtense := railOracle(t, a, tW, tK, tN)
		assertSetbackRail(t, w, a, wantEnds, wantSubtense, tol)
	}
}

// assertSetbackRail builds one arm's rail and asserts it is the certified great-circle arc: a
// supporting circle centred at C with radius r whose plane passes through C, the two endpoints the
// arm's two host-tangent points (either order), and the certified subtense.
func assertSetbackRail(t *testing.T, w cornerWeld, a armSetback, wantEnds [2]math.Point3, wantSubtense, tol float64) {
	t.Helper()
	rail, ok := curvedSetbackRail(w, a)
	if !ok {
		t.Fatalf("curvedSetbackRail declined a certified arm (%T)", a.arm)
	}
	if d := rail.Center.DistanceTo(w.center); d > tol {
		t.Fatalf("rail centre off C by %.3e (want ≤%.1e) — not a great circle", d, tol)
	}
	if e := stdmath.Abs(rail.Radius - w.radius); e > tol {
		t.Fatalf("rail radius = %.9f, want r=%.9f ±%.1e", rail.Radius, w.radius, tol)
	}
	if off := stdmath.Abs(rail.Center.VectorTo(w.center).Dot(rail.Normal.AsVector())); off > tol {
		t.Fatalf("C is %.3e off the rail plane (want ≤%.1e) — plane not through C", off, tol)
	}
	assertRailEnds(t, rail, wantEnds, tol)
	if s := stdmath.Abs(rail.SweepAngle); stdmath.Abs(s-wantSubtense) > 1e-4 {
		t.Fatalf("rail subtense = %.6f rad, want %.6f ±1e-4", s, wantSubtense)
	}
}

// assertRailEnds checks the rail's two endpoints match the certified host-tangent pair (either order).
func assertRailEnds(t *testing.T, rail geom.Arc3d, want [2]math.Point3, tol float64) {
	t.Helper()
	p0, p1 := rail.PointAt(0), rail.PointAt(1)
	forward := p0.DistanceTo(want[0]) <= tol && p1.DistanceTo(want[1]) <= tol
	reverse := p0.DistanceTo(want[1]) <= tol && p1.DistanceTo(want[0]) <= tol
	if !forward && !reverse {
		t.Fatalf("rail endpoints (%v,%v) match neither ordering of (%v,%v) within %.1e", p0, p1, want[0], want[1], tol)
	}
}

// TestCurvedRailG1_B3 certifies the exact-G1 weld: along every arm's rail the arm normal (canal
// identity (P−m)/r) and the sphere normal (P−C)/r must coincide within res.Weld(). The NEGATIVE
// case proves the assertion bites — offsetting the arm surface centre by 0.1·r moves its moving
// ball-centre m off C, so ‖n_arm−n_sphere‖=‖C−m‖/r≈0.1 ≫ res.Weld(), and curvedRailG1 must reject.
func TestCurvedRailG1_B3(t *testing.T) {
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner")
	}
	for _, a := range w.arms {
		rail, ok := curvedSetbackRail(w, a)
		if !ok {
			t.Fatalf("curvedSetbackRail declined a certified arm (%T)", a.arm)
		}
		if !curvedRailG1(a.arm, rail, w.center, w.radius, res) {
			t.Fatalf("curvedRailG1 failed on a certified arm (%T) — normals should coincide exactly", a.arm)
		}
	}
	assertRailG1Bites(t, w, res)
}

// assertRailG1Bites is the mutation: rebuild the vertical cyl arm with its axis offset 0.1·r in x,
// keep the true rail, and require curvedRailG1 to reject. It also reports the observed normal
// mismatch magnitude so the bite is quantified, not just asserted.
func assertRailG1Bites(t *testing.T, w cornerWeld, res Resolution) {
	t.Helper()
	cyl := findVerticalCylArm(t, w)
	rail, ok := curvedSetbackRail(w, cyl)
	if !ok {
		t.Fatalf("curvedSetbackRail declined the cyl arm")
	}
	off := 0.1 * w.radius
	mutated := mustCylinder(t, math.P3(10+off, b3CornerCY, 0), math.V3(0, 0, 1), 10)
	if curvedRailG1(mutated, rail, w.center, w.radius, res) {
		t.Fatalf("curvedRailG1 accepted an arm centre offset 0.1·r (the G1 assertion did not bite)")
	}
	t.Logf("G1 mutation (0.1·r=%.3f offset): observed normal mismatch = %.6f (tol res.Weld()=%.3e)",
		off, observedRailMismatch(mutated, rail, w.center, w.radius), res.Weld())
}

// findVerticalCylArm returns the W∧N vertical cylinder arm (ẑ axis) from the solved corner.
func findVerticalCylArm(t *testing.T, w cornerWeld) armSetback {
	t.Helper()
	for _, a := range w.arms {
		if s, ok := a.arm.(geom.Cylinder); ok && stdmath.Abs(float64(s.AxisDir.Z())-1) < 0.5 {
			return a
		}
	}
	t.Fatalf("no vertical cyl arm in the solved corner")
	return armSetback{}
}

// observedRailMismatch is the max over rail samples of ‖(P−m)/r − (P−C)/r‖ = ‖C−m‖/r, the exact
// G1 error the derivation names — used only to report the mutation's bite magnitude.
func observedRailMismatch(arm geom.Surface, rail geom.Arc3d, center math.Point3, r float64) float64 {
	worst := 0.0
	for i := 0; i < 5; i++ {
		p := rail.PointAt(float64(i) / 4)
		m, ok := armBallCenter(arm, p)
		if !ok {
			continue
		}
		d := m.VectorTo(p).Scale(1 / r).Sub(center.VectorTo(p).Scale(1 / r)).Length()
		worst = stdmath.Max(worst, d)
	}
	return worst
}

// b3TorusArm returns the B3 torus arm surface from the solved corner (the W∧K rolling-ball torus).
func b3TorusArm(t *testing.T, w cornerWeld) geom.Torus {
	t.Helper()
	for _, a := range w.arms {
		if tor, ok := a.arm.(geom.Torus); ok {
			return tor
		}
	}
	t.Fatalf("no torus arm in the solved corner")
	return geom.Torus{}
}

// TestCurvedHostArc_B3 drives the circular host-rail emitter (T5.3 §B.1/B.2): the torus arm carves a
// circle of radius R=50 in the wall (spine plane z=90) and a circle of radius R−r=40 in the cap
// (plane z=100), each sweeping azimuth 0°→−75.522° from the y=0 cut to the sphere-side host-tangent
// point. This is the rail the straight-pull transformFace cannot represent.
func TestCurvedHostArc_B3(t *testing.T) {
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner")
	}
	tor := b3TorusArm(t, w)
	tW, tK, _ := b3TangentPoints()
	wall := mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 50)
	capPl := mustPlane(t, math.P3(0, 0, 100), math.V3(0, 0, 1))
	assertHostArc(t, mustHostArc(t, wall, tor, w, res), math.P3(0, 0, 90), 50, math.P3(50, 0, 90), tW)
	assertHostArc(t, mustHostArc(t, capPl, tor, w, res), math.P3(0, 0, 100), 40, math.P3(40, 0, 100), tK)
}

// mustHostArc builds a host arc or fails the test.
func mustHostArc(t *testing.T, host geom.Surface, tor geom.Torus, w cornerWeld, res Resolution) geom.Arc3d {
	t.Helper()
	arc, ok := curvedHostArc(host, tor, w, res)
	if !ok {
		t.Fatalf("curvedHostArc declined host %T", host)
	}
	return arc
}

// assertHostArc checks a host arc against its certified circle: centre, radius, a plane normal along
// ẑ (the arc lies in a constant-z plane), the far endpoint at azimuth 0, the near endpoint at the
// host-tangent point, and the certified −75.522° sweep.
func assertHostArc(t *testing.T, arc geom.Arc3d, center math.Point3, radius float64, far, near math.Point3) {
	t.Helper()
	if d := arc.Center.DistanceTo(center); float64(d) > 1e-6 {
		t.Fatalf("arc centre %v off %v by %.3e", arc.Center, center, d)
	}
	if e := stdmath.Abs(arc.Radius - radius); e > 1e-6 {
		t.Fatalf("arc radius = %.9f, want %.9f", arc.Radius, radius)
	}
	if z := stdmath.Abs(float64(arc.Normal.Z()) - 1); z > 1e-9 {
		t.Fatalf("arc normal %v not ±ẑ (arc not in a constant-z plane)", arc.Normal)
	}
	if d := arc.PointAt(0).DistanceTo(far); float64(d) > 1e-4 {
		t.Fatalf("arc PointAt(0) = %v, want far end %v (Δ=%.3e)", arc.PointAt(0), far, d)
	}
	if d := arc.PointAt(1).DistanceTo(near); float64(d) > 1e-4 {
		t.Fatalf("arc PointAt(1) = %v, want host-tangent %v (Δ=%.3e)", arc.PointAt(1), near, d)
	}
	if s := stdmath.Abs(arc.SweepAngle) - 75.522*stdmath.Pi/180; stdmath.Abs(s) > 1e-3 {
		t.Fatalf("arc sweep = %.6f rad, want 75.522° (Δ=%.3e)", arc.SweepAngle, s)
	}
}
