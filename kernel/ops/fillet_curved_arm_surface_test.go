// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// nearlyArm is the tight scalar tolerance for the exact-analytic B3 arm assertions: the OCCT
// oracle values (major 40, minor 10, centre z 90, radius 10) are closed-form, so an arm surface
// that is off by more than this is wrong, not merely imprecise.
func nearlyArm(got, want float64) bool { return stdmath.Abs(got-want) < 1e-6 }

// armOutward wraps a raw vector as a material-outward unit normal for the arm-builder tests (their bare
// test planes are constructed with their normals already pointing out of the material).
func armOutward(x, y, z float64) math.UnitVector3 {
	u, err := math.UnitVector3FromVector(math.V3(x, y, z))
	if err != nil {
		panic(err)
	}
	return u
}

// planeAtZ is the B3 top cap: a plane at height z with its OUTWARD normal +z (the cylinder body
// sits below it, so material is on the −z side — the side torusArmSurface offsets the centre onto).
func planeAtZ(z float64) geom.Plane {
	p, err := geom.NewPlane(math.P3(0, 0, z), math.V3(0, 0, 1))
	if err != nil {
		panic(err)
	}
	return p
}

// b3VerticalWallEdge is the picked Cyl∧Plane LINE edge of B3: the vertical ruling where the radial
// plane x=0 meets the R=50 wall, at (0,50,z). A bare edge (no faces) is enough — cylinderArmSurface
// reads only its endpoints to select which of the two P_r∩C_ρ rulings the fillet sits on.
func b3VerticalWallEdge(t *testing.T) *topo.Edge {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "b3wall", 0))
	bld := topo.NewBuilder(true, lin)
	lo := bld.AddVertex(math.P3(0, 50, 0), lin)
	hi := bld.AddVertex(math.P3(0, 50, 100), lin)
	return bld.AddEdge(geom.NewLineSegment(math.P3(0, 50, 0), math.P3(0, 50, 100)), lo, hi, lin)
}

// TestTorusArmSurface_B3 pins the config-(i) torus arm to the exact OCCT BREP the derivation cites
// (`5 0 0 90 0 0 1 … 40 10`): with NO trimmed host faces the side-selection cannot run its contact-foot
// gate, so it falls back to the BOSS-CAP side — major = R−r = 40, minor = r = 10, centre offset r below
// the cap (z = 100−10 = 90). This is the byte-identity floor every current caller keeps; the
// external-shoulder (R+r = 60) flip is exercised by the face-carrying fixtures below.
func TestTorusArmSurface_B3(t *testing.T) {
	res := testArmResolution()
	tor, ok := torusArmSurface(cylAxis(0, 0, 1, 50), planeAtZ(100), nil, nil, math.P3(50, 0, 100), armOutward(0, 0, 1), 10, res)
	if !ok {
		t.Fatalf("torusArmSurface declined a valid convex B3 rim")
	}
	if !nearlyArm(tor.MajorRadius, 40) || !nearlyArm(tor.MinorRadius, 10) || !nearlyArm(tor.Center.Z, 90) {
		t.Fatalf("B3 torus arm = {major %.6f, minor %.6f, cz %.6f}, want {40,10,90} (OCCT BREP 5 0 0 90 … 40 10)",
			tor.MajorRadius, tor.MinorRadius, tor.Center.Z)
	}
}

// TestTorusArmSurface_Spindle is the existence guard: at r ≥ R the convex tube radius R−r collapses
// onto (or through) the axis — a self-intersecting spindle/horn torus — so the constructor must
// honest-reject rather than emit degenerate geometry (§Numerical pitfalls).
func TestTorusArmSurface_Spindle(t *testing.T) {
	res := testArmResolution()
	if _, ok := torusArmSurface(cylAxis(0, 0, 1, 50), planeAtZ(100), nil, nil, math.P3(50, 0, 100), armOutward(0, 0, 1), 50, res); ok {
		t.Fatalf("torusArmSurface accepted r=R=50 (major R−r=0): a spindle torus must be rejected")
	}
}

// TestCylinderArmSurface_B3 pins the config-(ii) cylinder arm: a rolling-ball cylinder of radius
// r=10 about the selected ruling of P_r∩C_ρ on the vertical wall (OCCT BREP `2 … 10`).
func TestCylinderArmSurface_B3(t *testing.T) {
	cyl, ok := cylinderArmSurface(b3VerticalWallEdge(t), cylAxis(0, 0, 1, 50), planeWithNormal(1, 0, 0), armOutward(1, 0, 0), 10)
	if !ok || !nearlyArm(cyl.Radius, 10) {
		t.Fatalf("B3 cylinder arm radius = %.6f (ok=%v), want 10", cyl.Radius, ok)
	}
	// The selected ruling is parallel to the cylinder axis (the edge is an axis ruling).
	if !nearlyArm(stdmath.Abs(cyl.AxisDir.AsVector().Dot(math.V3(0, 0, 1))), 1) {
		t.Fatalf("B3 cylinder arm axis = %v, want ∥ ẑ", cyl.AxisDir)
	}
}

// TestCylinderArmSurface_Clears is the existence guard: when P_r never reaches C_ρ (the offset plane
// clears the offset cylinder, |m| > ρ) there is no real ruling, so the constructor rejects the edge.
func TestCylinderArmSurface_Clears(t *testing.T) {
	far, err := geom.NewPlane(math.P3(200, 0, 0), math.V3(1, 0, 0)) // 200 ≫ ρ = R−r = 40
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	if _, ok := cylinderArmSurface(b3VerticalWallEdge(t), cylAxis(0, 0, 1, 50), far, armOutward(1, 0, 0), 10); ok {
		t.Fatalf("cylinderArmSurface accepted a plane that clears the offset cylinder")
	}
}

// TestCylinderArmSurface_B3 and its siblings above are the LIVE arm-builder tests (fillet_curved.go
// callers). The config-ii/§D5 cross-section-arc chain (curvedArmSectionArc and friends) was deleted
// with its tests in the M5 Slice A whole-branch fix wave: the great-circle setback rail (T5.2)
// replaced the section-span rail, leaving that chain with zero production callers (the π/2±asin(r/ρ)
// derivation is preserved in m5-arm-section-derivation.md for a future concave slice).

// planeCircleLoopUses builds a closed loop of FOUR quarter-arc edges on the circle of radius R centred
// on the Z axis at height z — a real polygon planeFootOnTrimmedFace can ray-cast (a single closed
// geom.Circle edge samples to a degenerate 2-gon, so the contact-foot gate needs an arc-subdivided rim).
func planeCircleLoopUses(bld *topo.Builder, lin topo.Lineage, radius, z float64) []topo.Use {
	verts := make([]*topo.Vertex, 4)
	for i := 0; i < 4; i++ {
		a := float64(i) * stdmath.Pi / 2
		verts[i] = bld.AddVertex(math.P3(radius*stdmath.Cos(a), radius*stdmath.Sin(a), z), lin)
	}
	uses := make([]topo.Use, 0, 4)
	for i := 0; i < 4; i++ {
		amid := (float64(i) + 0.5) * stdmath.Pi / 2
		mid := math.P3(radius*stdmath.Cos(amid), radius*stdmath.Sin(amid), z)
		arc, err := geom.Arc3dByThreePoints(verts[i].Point(), mid, verts[(i+1)%4].Point())
		if err != nil {
			panic(err)
		}
		uses = append(uses, topo.Fwd(bld.AddEdge(arc, verts[i], verts[(i+1)%4], lin)))
	}
	return uses
}

// bossCapArmFaces synthesises the trimmed hosts of a BOSS-CAP rim (B3-like): an R=50 wall (axis ẑ) and
// a cap DISK (r ≤ 50) at z=100. The cylinder face needs no loop (armRunoutFoot reads the infinite
// surface); only the cap's trimmed loop decides the side — the R−r=40 foot lands ON the disk, the R+r=60
// foot spills OFF it. Returns the two faces and a point on the rim (the arm ball-centre azimuth).
func bossCapArmFaces() (cylFace, planeFace *topo.Face, edgeMid math.Point3) {
	lin := topo.NewLineage(topo.Tok("test", "bosscaparm", 0))
	bld := topo.NewBuilder(true, lin)
	cylFace = bld.AddFace(cylAxis(0, 0, 1, 50), lin)
	planeFace = bld.AddFace(planeAtZ(100), lin, topo.OuterLoop(planeCircleLoopUses(bld, lin, 50, 100)...))
	bld.Build()
	return cylFace, planeFace, math.P3(50, 0, 100)
}

// externalShoulderArmFaces synthesises the trimmed hosts of an EXTERNAL-SHOULDER rim (the convex shaft-
// shelf dual of the boss cap): an R=50 wall (axis ẑ) and a shelf ANNULUS (50 ≤ r ≤ 200) at z=−50. The
// annulus hole is exactly the wall radius, so the R−r=40 foot falls IN the hole (off the trimmed shelf)
// while the R+r=60 foot lands on the annulus — the gate must pick the external-shoulder torus.
func externalShoulderArmFaces() (cylFace, planeFace *topo.Face, edgeMid math.Point3) {
	lin := topo.NewLineage(topo.Tok("test", "extshoulderarm", 0))
	bld := topo.NewBuilder(true, lin)
	cylFace = bld.AddFace(cylAxis(0, 0, 1, 50), lin)
	planeFace = bld.AddFace(planeAtZ(-50), lin,
		topo.OuterLoop(planeCircleLoopUses(bld, lin, 200, -50)...),
		topo.InnerLoop(planeCircleLoopUses(bld, lin, 50, -50)...))
	bld.Build()
	return cylFace, planeFace, math.P3(50, 0, -50)
}

// TestTorusArmSurface_BossCapPicksInner is the H6-root2 regression, boss-cap half: with the REAL cap
// disk in hand the contact-foot gate selects the BOSS-CAP torus (major R−r=40, centre r into the
// material at z=90) — byte-identical to the pre-root2 hardcoded side (B3, OCCT BREP `5 0 0 90 … 40 10`).
func TestTorusArmSurface_BossCapPicksInner(t *testing.T) {
	cylFace, planeFace, mid := bossCapArmFaces()
	tor, ok := torusArmSurface(cylAxis(0, 0, 1, 50), planeAtZ(100), cylFace, planeFace, mid, armOutward(0, 0, 1), 10, testArmResolution())
	if !ok {
		t.Fatalf("torusArmSurface declined a valid boss-cap rim")
	}
	if !nearlyArm(tor.MajorRadius, 40) || !nearlyArm(tor.MinorRadius, 10) || !nearlyArm(tor.Center.Z, 90) {
		t.Fatalf("boss-cap arm = {major %.6f, minor %.6f, cz %.6f}, want {40,10,90} (R−r, ball into the material)",
			tor.MajorRadius, tor.MinorRadius, tor.Center.Z)
	}
}

// TestTorusArmSurface_ExternalShoulderPicksOuter is the H6-root2 regression, external-shoulder half: with
// the REAL shelf annulus (hole = the wall radius) the contact-foot gate selects the EXTERNAL-SHOULDER
// torus (major R+r=60, centre r into the VOID at z=−40) — the side the pre-root2 hardcoded R−r never
// emitted. The R−r=40 cap foot would fall in the annulus hole (off the shelf); only R+r=60 lands on it.
func TestTorusArmSurface_ExternalShoulderPicksOuter(t *testing.T) {
	cylFace, planeFace, mid := externalShoulderArmFaces()
	tor, ok := torusArmSurface(cylAxis(0, 0, 1, 50), planeAtZ(-50), cylFace, planeFace, mid, armOutward(0, 0, 1), 10, testArmResolution())
	if !ok {
		t.Fatalf("torusArmSurface declined a valid external-shoulder rim")
	}
	if !nearlyArm(tor.MajorRadius, 60) || !nearlyArm(tor.MinorRadius, 10) || !nearlyArm(tor.Center.Z, -40) {
		t.Fatalf("external-shoulder arm = {major %.6f, minor %.6f, cz %.6f}, want {60,10,−40} (R+r, ball into the void)",
			tor.MajorRadius, tor.MinorRadius, tor.Center.Z)
	}
}
