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
// (`5 0 0 90 0 0 1 … 40 10`): convex ⟹ major = R−r = 40, minor = r = 10, centre offset r below the
// cap (z = 100−10 = 90). A flipped material-side sign would give major R+r = 60 instead.
func TestTorusArmSurface_B3(t *testing.T) {
	res := testArmResolution()
	tor, ok := torusArmSurface(cylAxis(0, 0, 1, 50), planeAtZ(100), 10, res)
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
	if _, ok := torusArmSurface(cylAxis(0, 0, 1, 50), planeAtZ(100), 50, res); ok {
		t.Fatalf("torusArmSurface accepted r=R=50 (major R−r=0): a spindle torus must be rejected")
	}
}

// TestCylinderArmSurface_B3 pins the config-(ii) cylinder arm: a rolling-ball cylinder of radius
// r=10 about the selected ruling of P_r∩C_ρ on the vertical wall (OCCT BREP `2 … 10`).
func TestCylinderArmSurface_B3(t *testing.T) {
	cyl, ok := cylinderArmSurface(b3VerticalWallEdge(t), cylAxis(0, 0, 1, 50), planeWithNormal(1, 0, 0), 10)
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
	if _, ok := cylinderArmSurface(b3VerticalWallEdge(t), cylAxis(0, 0, 1, 50), far, 10); ok {
		t.Fatalf("cylinderArmSurface accepted a plane that clears the offset cylinder")
	}
}

// TestCurvedArmSectionArc_B3 checks the corner cross-section quarter-arc on the torus arm: at any
// u-station it is a radius-r=10 arc running from the cyl-contact (on the R=50 wall) to the
// plane-contact (on the cap z=100) — the [φ_P,φ_C] tube quarter the corner engine consumes (§D5).
func TestCurvedArmSectionArc_B3(t *testing.T) {
	res := testArmResolution()
	tor, ok := torusArmSurface(cylAxis(0, 0, 1, 50), planeAtZ(100), 10, res)
	if !ok {
		t.Fatalf("torusArmSurface declined a valid convex B3 rim")
	}
	sec := curvedArmSectionArc(tor, 0)
	arc, isArc := sec.(geom.Arc3d)
	if !isArc || !nearlyArm(arc.Radius, 10) {
		t.Fatalf("section arc = %#v, want a radius-10 Arc3d", sec)
	}
	start, end := sec.PointAt(0), sec.PointAt(1)
	if !nearlyArm(stdmath.Hypot(start.X, start.Y), 50) { // cyl-contact on the R=50 wall
		t.Fatalf("section start radius = %.6f, want 50 (cyl wall)", stdmath.Hypot(start.X, start.Y))
	}
	if !nearlyArm(end.Z, 100) { // plane-contact on the cap
		t.Fatalf("section end z = %.6f, want 100 (cap plane)", end.Z)
	}
}
