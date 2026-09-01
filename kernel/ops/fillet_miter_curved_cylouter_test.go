// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// m3MiterCorner is a named fixture for OCCT blend/simple M3's concave base-cove miter corner (vertex
// (20,15,100), r=5): the arc arm is a CONCAVE torus C(50,15,105)/major R+r=35/minor 5 about ẑ, the
// line arm an r=5 cylinder about the box edge (axis y=10,z=105 along x). Its torus-outer host is the
// quarter-cyl WALL — a geom.Cylinder R30 coaxial with the torus (the MIRROR of P5's plane-outer host).
// DRAWEXE ground truth: sBot=(20,15,105) on the wall contact circle radius 30 (curved-miter-seam-recon
// §1c). This exercises miterSeamBottom's cylinder-outer branch on raw geometry without importing STEP.
type m3MiterCorner struct {
	arms    curvedMiterArms
	torWall *topo.Face
	vertex  math.Point3
	r       float64
}

// newM3MiterCorner builds the M3 fixture (arms + the cyl-wall torus-outer face) via a topo builder.
func newM3MiterCorner(t *testing.T) m3MiterCorner {
	t.Helper()
	tor, err := geom.NewTorusWithRef(math.P3(50, 15, 105), math.V3(0, 0, 1), math.V3(1, 0, 0), 35, 5)
	if err != nil {
		t.Fatalf("M3 torus arm: %v", err)
	}
	cyl, err := geom.NewCylinderWithRef(math.P3(50, 10, 105), math.V3(-1, 0, 0), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatalf("M3 cylinder arm: %v", err)
	}
	wall, err := geom.NewCylinder(math.P3(50, 15, 105), math.V3(0, 0, 1), 30)
	if err != nil {
		t.Fatalf("M3 cyl-wall outer host: %v", err)
	}
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("m3", "body", 0)))
	face := bld.AddFace(wall, topo.NewLineage(topo.Tok("m3", "wall", 0)))
	return m3MiterCorner{arms: curvedMiterArms{tor: tor, cyl: cyl}, torWall: face, vertex: math.P3(20, 15, 100), r: 5}
}

// TestMiterSeamBottomCylinderOuterHost is the M3/M9 sBot regression: for a CYLINDER torus-outer host,
// miterSeamBottom builds sBot on the torus↔host-cylinder contact circle (radius 30 about (50,15,105)),
// landing on OCCT's (20,15,105) and lying on BOTH arm tubes to ~1e-9 (the equal-r seam endpoint).
func TestMiterSeamBottomCylinderOuterHost(t *testing.T) {
	t.Parallel()
	fx := newM3MiterCorner(t)
	res := ResolutionForPoints([]math.Point3{fx.vertex, fx.arms.tor.Center})
	sBot, ok := miterSeamBottom(fx.arms, fx.torWall, fx.vertex, res)
	if !ok {
		t.Fatal("M3 cylinder-outer sBot declined (seam did not close at the torus-outer wall)")
	}
	if d := sBot.DistanceTo(math.P3(20, 15, 105)); float64(d) > 1e-9 {
		t.Fatalf("M3 sBot: got %v, want (20,15,105) (err %.3g)", sBot, float64(d))
	}
	torErr := stdmath.Abs(torusTubeMembership(fx.arms.tor, fx.r, sBot))
	cylErr := stdmath.Abs(float64(cylinderBallCenter(fx.arms.cyl, sBot).DistanceTo(sBot)) - fx.r)
	if torErr > 1e-9 || cylErr > 1e-9 {
		t.Fatalf("M3 sBot %v off the tubes: torus=%.3g cyl=%.3g (tol 1e-9)", sBot, torErr, cylErr)
	}
}

// TestMiterSeamBottomOnContactCircleRadius pins that M3's sBot lies on the cyl-WALL contact circle
// (radius hostCyl.Radius=30 about the torus centre, in the plane z=105) — the mirror of the plane
// branch's major-circle contact — not on some other torus∩cylinder crossing.
func TestMiterSeamBottomOnContactCircleRadius(t *testing.T) {
	t.Parallel()
	fx := newM3MiterCorner(t)
	res := ResolutionForPoints([]math.Point3{fx.vertex, fx.arms.tor.Center})
	sBot, ok := miterSeamBottom(fx.arms, fx.torWall, fx.vertex, res)
	if !ok {
		t.Fatal("M3 cylinder-outer sBot declined")
	}
	rho := stdmath.Hypot(float64(sBot.X)-50, float64(sBot.Y)-15)
	if stdmath.Abs(rho-30) > 1e-9 {
		t.Fatalf("M3 sBot must lie on the wall contact circle radius 30, got ρ=%.9f", rho)
	}
	if stdmath.Abs(float64(sBot.Z)-105) > 1e-9 {
		t.Fatalf("M3 sBot must lie in the contact plane z=105, got z=%.9f", sBot.Z)
	}
}

// TestMiterSeamBottomPlaneOuterUnchanged is the DO-NO-HARM guard: for a PLANE torus-outer host (P5's
// convex top rim), miterSeamBottom still routes to the untouched plane branch and returns the same
// DRAWEXE sBot=(53.332,5.124,150) TestCurvedMiterSeamBottomOnTorusOuterHost pins — the additive
// cylinder branch does not perturb the convex seam.
func TestMiterSeamBottomPlaneOuterUnchanged(t *testing.T) {
	t.Parallel()
	fx := p5MiterCorner(t)
	plane, err := geom.NewPlane(math.P3(50, 50, 150), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("P5 top-plane outer host: %v", err)
	}
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("p5", "body", 0)))
	top := bld.AddFace(plane, topo.NewLineage(topo.Tok("p5", "top", 0)))
	res := ResolutionForPoints([]math.Point3{math.P3(50, 50, 150), fx.center})
	sBot, ok := miterSeamBottom(fx.arms, top, math.P3(65, 2.303, 150), res)
	if !ok {
		t.Fatal("P5 plane-outer sBot declined (the convex branch must still close)")
	}
	assertXY(t, "P5 sBot (plane branch unchanged)", sBot, 53.332474, 5.123563, 1e-3)
	if stdmath.Abs(float64(sBot.Z)-150) > 1e-9 {
		t.Fatalf("P5 sBot must lie on the top plane z=150, got z=%.9f", sBot.Z)
	}
}
