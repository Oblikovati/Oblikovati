// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// concaveBoreFixture wires a synthetic CONCAVE bore edge: the R=30 cylinder wall (axis +z) meets the
// plane y=0 along the vertical ruling at (30,0,z), with the cylinder face REVERSED so its material-
// outward normal points toward the axis (−r̂) — a BORE (material outside the wall, void inside). Bare
// two-face topology (each face carries the shared edge in a degenerate loop) is enough:
// concaveCylinderArmSurface reads only the edge endpoints, the cyl face's outward normal (ε), and the
// two geometries. planeN is the plane's material-outward (into-void) normal +ŷ.
func concaveBoreFixture(t *testing.T) (*topo.Edge, geom.Cylinder, geom.Plane, math.UnitVector3) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "concave-bore", 0))
	bld := topo.NewBuilder(true, lin)
	lo := bld.AddVertex(math.P3(30, 0, 0), lin)
	hi := bld.AddVertex(math.P3(30, 0, 50), lin)
	e := bld.AddEdge(geom.NewLineSegment(math.P3(30, 0, 0), math.P3(30, 0, 50)), lo, hi, lin)
	cyl := cylAxis(0, 0, 1, 30)
	pl := planeWithNormal(0, 1, 0)
	bld.AddReversedFace(cyl, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e))) // reversed ⇒ outward −r̂ (bore)
	bld.AddFace(pl, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e)))
	bld.Build()
	return e, cyl, pl, armOutward(0, 1, 0)
}

// TestConcaveCylinderArmSurface_VoidSide pins the config-(ii) CONCAVE cylinder arm on the bore fixture:
// radius r, axis ∥ ẑ, the ball centre at ρ = R−r = 20 from the axis (the derivation's data-driven radial
// offset for ε=−1) and offset +r onto the plane's VOID side (signed plane distance +r, not the convex
// −r). A convex-side mirror root would sit at −r; a flipped radial sign at ρ=R+r=40.
func TestConcaveCylinderArmSurface_VoidSide(t *testing.T) {
	e, cyl, pl, planeN := concaveBoreFixture(t)
	arm, ok := concaveCylinderArmSurface(e, cyl, pl, planeN, 10, testArmResolution())
	if !ok {
		t.Fatalf("concaveCylinderArmSurface declined a valid bore edge")
	}
	if !nearlyArm(arm.Radius, 10) || !nearlyArm(stdmath.Abs(arm.AxisDir.AsVector().Dot(math.V3(0, 0, 1))), 1) {
		t.Fatalf("bore concave arm = {radius %.6f, axis %v}, want {10, ∥ẑ}", arm.Radius, arm.AxisDir)
	}
	rho := stdmath.Hypot(arm.Origin.X, arm.Origin.Y) // dist(centre, axis) in the z=0 plane
	if !nearlyArm(rho, 20) {
		t.Fatalf("bore concave arm centre radial distance %.6f, want ρ = R−r = 20", rho)
	}
	if planeOff := arm.Origin.Y; !nearlyArm(planeOff, 10) { // signed dist to plane y=0 = +r (void side)
		t.Fatalf("bore concave arm plane offset %.6f, want +r = 10 (void side, not the convex −10)", planeOff)
	}
}

// TestConcaveCylinderArmSurface_Boss pins the boss branch (ε=+1): an unreversed cylinder face (material
// inside the wall, +r̂ outward) offsets the ball to ρ = R+r = 40 — the derivation's data-driven radial
// sign, verified distinct from the bore's R−r.
func TestConcaveCylinderArmSurface_Boss(t *testing.T) {
	e, cyl, pl, planeN := concaveBossFixture(t)
	arm, ok := concaveCylinderArmSurface(e, cyl, pl, planeN, 10, testArmResolution())
	if !ok {
		t.Fatalf("concaveCylinderArmSurface declined a valid boss edge")
	}
	if rho := stdmath.Hypot(arm.Origin.X, arm.Origin.Y); !nearlyArm(rho, 40) {
		t.Fatalf("boss concave arm centre radial distance %.6f, want ρ = R+r = 40", rho)
	}
}

// concaveBossFixture is concaveBoreFixture's boss dual: the cylinder face is NOT reversed, so its
// material-outward normal is +r̂ (material inside the wall) — ε=+1, ρ=R+r.
func concaveBossFixture(t *testing.T) (*topo.Edge, geom.Cylinder, geom.Plane, math.UnitVector3) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "concave-boss", 0))
	bld := topo.NewBuilder(true, lin)
	lo := bld.AddVertex(math.P3(30, 0, 0), lin)
	hi := bld.AddVertex(math.P3(30, 0, 50), lin)
	e := bld.AddEdge(geom.NewLineSegment(math.P3(30, 0, 0), math.P3(30, 0, 50)), lo, hi, lin)
	cyl := cylAxis(0, 0, 1, 30)
	pl := planeWithNormal(0, 1, 0)
	bld.AddFace(cyl, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e))) // unreversed ⇒ outward +r̂ (boss)
	bld.AddFace(pl, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e)))
	bld.Build()
	return e, cyl, pl, armOutward(0, 1, 0)
}

// TestConcaveCylinderArmSurface_Spindle is the bore existence guard: at r ≥ R the offset cylinder radius
// ρ = R−r collapses onto (or through) the axis, so the constructor honest-rejects (derivation §4).
func TestConcaveCylinderArmSurface_Spindle(t *testing.T) {
	e, cyl, pl, planeN := concaveBoreFixture(t)
	if _, ok := concaveCylinderArmSurface(e, cyl, pl, planeN, 30, testArmResolution()); ok {
		t.Fatalf("concaveCylinderArmSurface accepted r=R=30 (bore spindle ρ=R−r=0) — must reject")
	}
}

// TestConcaveArmRulingBase_Clears is the P_r∩C_ρ existence guard: when the offset plane clears the offset
// cylinder (|m| > ρ, disc ≤ 0) there is no real ruling and the base solve declines (derivation §4).
func TestConcaveArmRulingBase_Clears(t *testing.T) {
	e, cyl, _, planeN := concaveBoreFixture(t)
	far := planeAtY(200) // 200 ≫ ρ = 20 ⇒ m = 200, disc = ρ²−m² < 0
	if _, ok := concaveArmRulingBase(e, cyl, far, planeN, 20, 10); ok {
		t.Fatalf("concaveArmRulingBase accepted an offset plane that clears the offset cylinder")
	}
}

// planeAtY is a plane at y = d with outward normal +ŷ (the clearance-guard test plane).
func planeAtY(d float64) geom.Plane {
	p, err := geom.NewPlane(math.P3(0, d, 0), math.V3(0, 1, 0))
	if err != nil {
		panic(err)
	}
	return p
}
