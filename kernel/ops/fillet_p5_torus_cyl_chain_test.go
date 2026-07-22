// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The torus∩cylinder far-runout CHAIN (piece A), pinned to the DRAWEXE dump of OCCT blend/simple P5
// (torus-cyl-springs-feet-derivation.md §0): two equal-radius R=50 vertical cylinders, the top rim of s1
// filleted r=5 into a torus arm (centre (50,50,145), axis ẑ, R′=45, r=5) hosted on the s1 wall (cylinder
// R=50) and the s1 top cap (plane z=150), capping on the s2 wall (cylinder R=50 through (80,50)). These
// tests pin all three links: the cylinder-host spring (Link 1, Circle R=50=R_h + the HOST-vs-CAPPING guard),
// the cylinder-capping feet (Link 2, the two 13-digit DRAWEXE feet), and that P5's torus arm now FLOWS
// armSprings → springCapFoot → torusCylinderTrim to a geom.TorusCylinderArc instead of declining.

// p5Cyl is P5's cylinder-host-spring foot (65, 97.696960070847, 145) — the DRAWEXE start of edge result_3_1.
var p5CylFoot = math.P3(65, 97.696960070847, 145)

// p5PlaneFoot is P5's plane-host-spring foot (57.083333333333, 94.439018766045, 150) — DRAWEXE's end of it.
var p5PlaneFoot = math.P3(57.083333333333, 94.439018766045, 150)

// p5TorusArm reconstructs P5's torus arm and its three surfaces (arm torus; cylinder host s1; plane host the
// s1 top cap; capping cylinder s2), as measured from the DRAWEXE mksurface dump.
func p5TorusArm(t *testing.T) (tor geom.Torus, cylHost geom.Cylinder, planeHost geom.Plane, capping geom.Cylinder) {
	t.Helper()
	tor, err := geom.NewTorusWithRef(math.P3(50, 50, 145), math.V3(0, 0, 1), math.V3(1, 0, 0), 45, 5)
	if err != nil {
		t.Fatalf("P5 arm torus: %v", err)
	}
	cylHost, err = geom.NewCylinder(math.P3(50, 50, 0), math.V3(0, 0, 1), 50)
	if err != nil {
		t.Fatalf("P5 host cylinder s1: %v", err)
	}
	capping, err = geom.NewCylinder(math.P3(80, 50, 60), math.V3(0, 0, 1), 50)
	if err != nil {
		t.Fatalf("P5 capping cylinder s2: %v", err)
	}
	planeHost = planeOn(t, math.P3(50, 50, 150), math.V3(0, 0, 1))
	return tor, cylHost, planeHost, capping
}

// p5EdgeFillet wraps P5's torus arm as an edgeFillet with real topo host faces (cylinder s1 = ef.a, top-cap
// plane = ef.b) — the shape armRunoutFeet/intersectArmCapping consume. Only the faces' Geometry() is read.
func p5EdgeFillet(t *testing.T, tor geom.Torus, cylHost geom.Cylinder, planeHost geom.Plane) edgeFillet {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "p5-chain", 0))
	bld := topo.NewBuilder(true, lin)
	return edgeFillet{a: stubFace(bld, lin, cylHost), b: stubFace(bld, lin, planeHost), armSurface: tor}
}

// TestP5CylinderHostSpring_IsEquatorCircle (Link 1): the cylinder-host spring is the tube-equator latitude
// circle — centre = torus centre, radius = R_h = 50, axis = torus axis — exactly OCCT's Circle R=50 rail.
func TestP5CylinderHostSpring_IsEquatorCircle(t *testing.T) {
	tor, cylHost, _, _ := p5TorusArm(t)
	tol := ResolutionForSize(200).Weld()
	spring, ok := torusCylinderSpring(tor, cylHost, tol)
	if !ok {
		t.Fatal("torusCylinderSpring declined P5's coaxial cylinder host")
	}
	if d := float64(spring.Center.DistanceTo(math.P3(50, 50, 145))); d > 1e-12 {
		t.Fatalf("spring centre %v off the torus centre by %.3e", spring.Center, d)
	}
	if stdmath.Abs(spring.Radius-50) > 1e-12 {
		t.Fatalf("spring radius %.15g != R_h=50 (the DRAWEXE Circle R=50 rail)", spring.Radius)
	}
}

// TestTorusCylinderSpring_HostVsCappingGuard (Link 1, load-bearing): a coaxial cylinder that CUTS the tube
// (|cos v_C|<1) is a transversal capping, not a tangent host — decline; and an OFF-AXIS cylinder (P5's own
// capping s2, whose axis misses the torus centre) is not a host either. Only the genuine tangent host is
// admitted.
func TestTorusCylinderSpring_HostVsCappingGuard(t *testing.T) {
	tor, _, _, capping := p5TorusArm(t)
	tol := ResolutionForSize(200).Weld()
	// A coaxial cylinder R=47 cuts the tube (cos v_C=(47−45)/5=0.4): transversal capping, decline.
	transversal, err := geom.NewCylinder(math.P3(50, 50, 0), math.V3(0, 0, 1), 47)
	if err != nil {
		t.Fatalf("transversal cylinder: %v", err)
	}
	if _, ok := torusCylinderSpring(tor, transversal, tol); ok {
		t.Fatal("a |cos v_C|<1 coaxial cylinder is a transversal CAPPING — must decline, not emit a mid-tube circle")
	}
	// P5's own capping s2 is off the torus axis (A⊥=30): not a host.
	if _, ok := torusCylinderSpring(tor, capping, tol); ok {
		t.Fatal("an off-axis cylinder (P5's capping s2) has no latitude-circle spring — must decline")
	}
}

// TestP5CylinderCappingFeet (Link 2): both host springs cross the capping cylinder s2 at the 13-digit
// DRAWEXE feet — the parallel-axis linear-trig branch (P5's axes are all ẑ).
func TestP5CylinderCappingFeet(t *testing.T) {
	tor, cylHost, planeHost, capping := p5TorusArm(t)
	tol := ResolutionForSize(200).Weld()
	res := ResolutionForSize(200)
	cylSpring, ok := torusCylinderSpring(tor, cylHost, tol)
	if !ok {
		t.Fatal("cylinder-host spring declined")
	}
	planeSpring, ok := torusPlaneSpring(tor, planeHost, tol)
	if !ok {
		t.Fatal("plane-host spring declined")
	}
	cf, ok0 := circleCylinderFoot(cylSpring, capping, math.P3(65, 97.7, 145), res)
	pf, ok1 := circleCylinderFoot(planeSpring, capping, math.P3(57, 94.4, 150), res)
	if !ok0 || !ok1 {
		t.Fatalf("circleCylinderFoot declined: cyl=%v plane=%v", ok0, ok1)
	}
	if d := float64(cf.DistanceTo(p5CylFoot)); d > 1e-7 {
		t.Fatalf("cylinder-host foot %v off DRAWEXE %v by %.3e", cf, p5CylFoot, d)
	}
	if d := float64(pf.DistanceTo(p5PlaneFoot)); d > 1e-7 {
		t.Fatalf("plane-host foot %v off DRAWEXE %v by %.3e", pf, p5PlaneFoot, d)
	}
}

// TestCircleCylinderFoot_SkewQuarticMachineEps (Link 2, general skew): the off-coaxial circle∩cylinder
// solver finds BOTH real roots of the degree-4 trig section, each exact on the cylinder to machine ε — the
// self-cert fixture from the derivation (circle a=20 ⊥ẑ ∩ tilted cylinder O₂=(15,0,0), â₂∝(0.3,0,0.954),
// R₂=12; g=0.3≠0).
func TestCircleCylinderFoot_SkewQuarticMachineEps(t *testing.T) {
	circle := skewFixtureCircle(t)
	cyl, err := geom.NewCylinder(math.P3(15, 0, 0), math.V3(0.3, 0, 0.954), 12)
	if err != nil {
		t.Fatalf("tilted cylinder: %v", err)
	}
	e1 := circle.RefDir.AsVector()
	e2 := circle.Normal.Cross(circle.RefDir)
	roots := bracketedTrigRoots(circleCylCoef(circle, cyl, e1, e2))
	if len(roots) != 2 {
		t.Fatalf("skew circle∩cylinder found %d real roots, want 2", len(roots))
	}
	a2 := cyl.AxisDir.AsVector()
	for i, tt := range roots {
		p := circle.PointAt(tt / (2 * stdmath.Pi))
		e := cyl.Origin.VectorTo(p)
		ea := float64(e.Dot(a2))
		f := float64(e.LengthSquared()) - ea*ea - cyl.Radius*cyl.Radius // cylinder implicit
		if stdmath.Abs(f) > 1e-9 {
			t.Fatalf("skew root %d: cylinder implicit residual %.3e (not machine ε)", i, f)
		}
		if d := stdmath.Abs(float64(circle.Center.VectorTo(p).Length()) - circle.Radius); d > 1e-12 {
			t.Fatalf("skew root %d: off the circle by %.3e", i, d)
		}
	}
}

// TestP5TorusArmFlowsToTorusCylinderArc is the NON-LATENT consumer gate: P5's torus arm, which used to
// decline at armSprings, now flows armSprings(cyl-spring Circle R=50) → springCapFoot(the two DRAWEXE feet)
// → torusCylinderTrim(a geom.TorusCylinderArc). The capping gets its first live consumer.
func TestP5TorusArmFlowsToTorusCylinderArc(t *testing.T) {
	tor, cylHost, planeHost, capping := p5TorusArm(t)
	ef := p5EdgeFillet(t, tor, cylHost, planeHost)
	res := ResolutionForSize(200)
	feet, ok, reason := armRunoutFeet(ef, capping, math.P3(65, 97.7, 145), math.P3(57, 94.4, 150), 5, res)
	if !ok {
		t.Fatalf("P5 torus arm declined at armRunoutFeet (blocker did NOT move to piece B): %s", reason)
	}
	if d := float64(feet[0].DistanceTo(p5CylFoot)); d > 1e-7 {
		t.Fatalf("feet[0] (ef.a=cyl host) %v off DRAWEXE %v by %.3e", feet[0], p5CylFoot, d)
	}
	if d := float64(feet[1].DistanceTo(p5PlaneFoot)); d > 1e-7 {
		t.Fatalf("feet[1] (ef.b=plane host) %v off DRAWEXE %v by %.3e", feet[1], p5PlaneFoot, d)
	}
	section, ok := intersectArmCapping(ef, capping, feet, 5, res)
	if !ok {
		t.Fatal("intersectArmCapping declined the torus∩cylinder trim (Link 3 did not fire)")
	}
	arc, isTCA := section.(geom.TorusCylinderArc)
	if !isTCA {
		t.Fatalf("section is %T, want geom.TorusCylinderArc (the far-runout capping trim)", section)
	}
	if d := float64(arc.PointAt(0).DistanceTo(feet[0])); d > float64(res.Weld()*5) {
		t.Fatalf("TorusCylinderArc PointAt(0) off foot0 by %.3e", d)
	}
	if d := float64(arc.PointAt(1).DistanceTo(feet[1])); d > float64(res.Weld()*5) {
		t.Fatalf("TorusCylinderArc PointAt(1) off foot1 by %.3e", d)
	}
}

// skewFixtureCircle is the a=20 circle ⊥ẑ centred at the origin, RefDir x̂ (the skew self-cert fixture).
func skewFixtureCircle(t *testing.T) geom.Circle {
	t.Helper()
	normal, err := math.NewUnitVector3(0, 0, 1)
	if err != nil {
		t.Fatalf("circle normal: %v", err)
	}
	ref, err := math.NewUnitVector3(1, 0, 0)
	if err != nil {
		t.Fatalf("circle ref: %v", err)
	}
	return geom.Circle{Center: math.P3(0, 0, 0), Normal: normal, RefDir: ref, Radius: 20}
}
