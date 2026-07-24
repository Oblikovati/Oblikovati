// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The concave analytic miter-arm regression: the ClassifyEdgeConvexity-gated side selection must build
// the convex R−r arm AND the concave R+r arm for each miter arm shape (torus, Cylinder∧Plane cylinder,
// equal-parallel-cyl valley), each ball on the correct side. It pins the DISCRIMINANT (convex ⟹ R−r,
// concave ⟹ R+r — e.g. O2's R55) so a future edit that flips a sign or drops the gate fails loud. The
// convex assertions duplicate the shipped R−r values (byte-identity anchors); the concave ones pin the
// new R+r sibling.

// TestMiterTorusArmSideSelection pins the torus arm's convex/concave side selection on one shared
// fixture (a R=50 cylinder ⊥ a cap plane at z=100, r=5): the convex builder gives major R−r=45 with the
// ball centre r INTO the material (z=95); the concave builder gives major R+r=55 (O2's Radii 55 5) with
// the ball centre r into the VOID (z=105). A flipped sign would swap 45↔55 and the centre side.
func TestMiterTorusArmSideSelection(t *testing.T) {
	res := testArmResolution()
	cyl, pl, n := cylAxis(0, 0, 1, 50), planeAtZ(100), armOutward(0, 0, 1)
	convex, ok := torusArmSurface(cyl, pl, n, 5, 1, res)
	if !ok {
		t.Fatalf("torusArmSurface (convex) declined a valid rim")
	}
	if !nearlyArm(convex.MajorRadius, 45) || !nearlyArm(convex.Center.Z, 95) {
		t.Fatalf("convex torus arm major=%.6f cz=%.6f, want R−r=45 and cz=95 (ball INTO material)",
			convex.MajorRadius, convex.Center.Z)
	}
	concave, ok := concaveTorusArmSurface(cyl, pl, n, 5, res)
	if !ok {
		t.Fatalf("concaveTorusArmSurface declined a valid reentrant rim")
	}
	if !nearlyArm(concave.MajorRadius, 55) || !nearlyArm(concave.Center.Z, 105) {
		t.Fatalf("concave torus arm major=%.6f cz=%.6f, want R+r=55 (O2) and cz=105 (ball into VOID)",
			concave.MajorRadius, concave.Center.Z)
	}
}

// TestMiterCylinderArmSideSelection pins the Cylinder∧Plane cylinder arm's convex/concave side selection
// on the shipped boss fixture (R=30 wall, r=10): the convex builder rides the offset cylinder at ρ=R−r=20
// (ball centre r INTO the material, plane offset −10); the concave builder rides ρ=R+r=40 (ε=+1 boss,
// plane offset +10 into the VOID). The radial distance of the arm centre from the host axis is the
// R−r vs R+r discriminant.
func TestMiterCylinderArmSideSelection(t *testing.T) {
	res := testArmResolution()
	e, cyl, pl, n := concaveBossFixture(t)
	convex, ok := cylinderArmSurface(e, cyl, pl, n, 10, 1)
	if !ok {
		t.Fatalf("cylinderArmSurface (convex) declined a valid boss edge")
	}
	if rho := stdmath.Hypot(float64(convex.Origin.X), float64(convex.Origin.Y)); !nearlyArm(rho, 20) {
		t.Fatalf("convex cylinder arm axis radial distance %.6f, want ρ=R−r=20 (ball INTO material)", rho)
	}
	concave, ok := concaveCylinderArmSurface(e, cyl, pl, n, 10, res)
	if !ok {
		t.Fatalf("concaveCylinderArmSurface declined a valid boss edge")
	}
	if rho := stdmath.Hypot(float64(concave.Origin.X), float64(concave.Origin.Y)); !nearlyArm(rho, 40) {
		t.Fatalf("concave cylinder arm axis radial distance %.6f, want ρ=R+r=40 (ball into VOID)", rho)
	}
	if !nearlyArm(float64(concave.Origin.Y), 10) {
		t.Fatalf("concave cylinder arm plane offset %.6f, want +r=10 (VOID side, not the convex −10)", concave.Origin.Y)
	}
}

// TestMiterEqualParallelValleySideSelection pins the equal-parallel Cylinder∧Cylinder arm's convex/concave
// side selection on a two-boss valley fixture (two R=50 axis-∥ cylinders 70 apart, r=5): the CONVEX arm
// rides each wall's offset at R−r=45 (ball INSIDE the walls); the CONCAVE fused-boss valley arm rides each
// wall at R+r=55 (ball OUTSIDE both walls, in the void valley — P2/P3). The arm axis' distance to EACH
// host axis is the discriminant; 55>R=50 (void) vs 45<R (material) shows the side.
func TestMiterEqualParallelValleySideSelection(t *testing.T) {
	res := testArmResolution()
	e, cA, cB := valleyBossFixture(t)
	for _, tc := range []struct {
		name string
		conv EdgeConvexity
		want float64
	}{
		{"convex R−r", EdgeConvex, 45},
		{"concave valley R+r", EdgeConcave, 55},
	} {
		s, ok := equalParallelCylMiterArm(e, 5, res, tc.conv)
		if !ok {
			t.Fatalf("%s: equalParallelCylMiterArm declined a valid equal-parallel edge", tc.name)
		}
		arm := s.(geom.Cylinder)
		dA := axisRadialDistance(arm.Origin, cA)
		dB := axisRadialDistance(arm.Origin, cB)
		if !nearlyArm(dA, tc.want) || !nearlyArm(dB, tc.want) {
			t.Fatalf("%s: arm axis is %.6f/%.6f from the two host axes, want %.1f from each", tc.name, dA, dB, tc.want)
		}
	}
}

// axisRadialDistance is the perpendicular distance from point p to cylinder cyl's axis line.
func axisRadialDistance(p math.Point3, cyl geom.Cylinder) float64 {
	w := cyl.Origin.VectorTo(p)
	a := cyl.AxisDir.AsVector()
	perp := w.Sub(a.Scale(math.Scalar(float64(w.Dot(a)))))
	return float64(perp.Length())
}

// valleyBossFixture wires two equal-radius (R=50) axis-∥ boss cylinders 70 apart (axes at the origin and
// (70,0,·)) sharing their intersection LINE edge at (35,−√1275,·) — the fused-boss valley (P2/P3's
// concave Cyl∧Cyl sibling of P5's boss∩bore). Both cylinder faces are UNREVERSED, so cylinderHostRadialSign
// reads ε=+1 (boss) for each: the concave arm then rides R+r=55 from each axis, the convex R−r=45.
func valleyBossFixture(t *testing.T) (*topo.Edge, geom.Cylinder, geom.Cylinder) {
	t.Helper()
	y := -stdmath.Sqrt(1275) // 50²−35² ⇒ the shared edge sits at (35, −35.707, z), on both R=50 walls
	lin := topo.NewLineage(topo.Tok("test", "valley-boss", 0))
	bld := topo.NewBuilder(true, lin)
	lo := bld.AddVertex(math.P3(35, y, 0), lin)
	hi := bld.AddVertex(math.P3(35, y, 50), lin)
	e := bld.AddEdge(geom.NewLineSegment(math.P3(35, y, 0), math.P3(35, y, 50)), lo, hi, lin)
	cA, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 50)
	if err != nil {
		t.Fatalf("valley cylinder A: %v", err)
	}
	cB, err := geom.NewCylinder(math.P3(70, 0, 0), math.V3(0, 0, 1), 50)
	if err != nil {
		t.Fatalf("valley cylinder B: %v", err)
	}
	bld.AddFace(cA, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e))) // unreversed ⇒ outward +r̂ (boss, ε=+1)
	bld.AddFace(cB, lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e)))
	bld.Build()
	return e, cA, cB
}
