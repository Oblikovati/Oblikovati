// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// twoCylCornerExactTol pins the synthetic-fixture centre to its hand-solved value tightly — sibling
// of coneCornerExactTol.
const twoCylCornerExactTol = 1e-9

// twoCylSyntheticFixture builds a hand-solvable two-parallel-cylinder + perpendicular-plane corner:
// CylA R=18 at origin, CylB R=20 at (14,0,0), both axis ẑ (boss, material inside each — un-reversed
// faces); a z=25 cap plane (boss, material below). r=5 gives offset radii ρA=13, ρB=15; the two
// offset circles' centre-to-centre distance is 14, so the classic 5-12-13 / 9-12-15 right triangles
// place the intersection at (5,±12) — verified independently below by direct circle-circle algebra,
// not by calling intersectCoplanarCircles.
func twoCylSyntheticFixture(t *testing.T) (*topo.Vertex, []*topo.Face, float64) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "twocyl-corner", 0))
	bld := topo.NewBuilder(true, lin)
	const r = 5.0
	vertex := math.P3(5, 12, 20) // the expected centre itself — nearer-vertex picks it unambiguously
	v := bld.AddVertex(vertex, lin)
	cylA, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 18)
	if err != nil {
		t.Fatalf("cylA: %v", err)
	}
	cylB, err := geom.NewCylinder(math.P3(14, 0, 0), math.V3(0, 0, 1), 20)
	if err != nil {
		t.Fatalf("cylB: %v", err)
	}
	faceA := bld.AddFace(cylA, lin)
	faceB := bld.AddFace(cylB, lin)
	plane := bld.AddFace(planeOn(t, math.P3(0, 0, 25), math.V3(0, 0, 1)), lin)
	return v, []*topo.Face{faceA, plane, faceB}, r
}

// TestTwoCylinderHostCorner_Recognizes checks twoParallelCylinderHostCorner accepts exactly the
// {2 cylinder, 1 plane} host set and returns the cylinder faces + plane face.
func TestTwoCylinderHostCorner_Recognizes(t *testing.T) {
	t.Parallel()
	_, faces, _ := twoCylSyntheticFixture(t)
	cylFaces, planeFace, ok := twoParallelCylinderHostCorner(faces)
	if !ok {
		t.Fatalf("twoParallelCylinderHostCorner declined a genuine {2 cyl, 1 plane} host set")
	}
	if planeFace == nil {
		t.Fatalf("planeFace is nil")
	}
	if _, isCyl := cylFaces[0].Geometry().(geom.Cylinder); !isCyl {
		t.Fatalf("cylFaces[0] is not a cylinder")
	}
	if _, isCyl := cylFaces[1].Geometry().(geom.Cylinder); !isCyl {
		t.Fatalf("cylFaces[1] is not a cylinder")
	}
}

// TestTwoCylinderHostCorner_ExactSyntheticCentre solves the hand-built fixture and checks the
// solved centre against an INDEPENDENT direct circle-circle-intersection computation (not
// intersectCoplanarCircles): a=(d²+ρA²−ρB²)/(2d), h=√(ρA²−a²), centre=(a,±h,axialC) with d=14,
// ρA=13, ρB=15, axialC=20 — the classic 5-12-13/9-12-15 integer right-triangle pair.
func TestTwoCylinderHostCorner_ExactSyntheticCentre(t *testing.T) {
	t.Parallel()
	v, faces, r := twoCylSyntheticFixture(t)
	cb, err := solveBlend(nil, v, faces, r)
	if err != nil {
		t.Fatalf("synthetic two-cylinder corner declined: %v", err)
	}
	const d, rhoA, rhoB = 14.0, 13.0, 15.0
	a := (d*d + rhoA*rhoA - rhoB*rhoB) / (2 * d)
	h := stdmath.Sqrt(rhoA*rhoA - a*a)
	want := math.P3(a, h, 20)
	if got := cb.center.DistanceTo(want); float64(got) > twoCylCornerExactTol {
		t.Fatalf("centre %v, want %v (independent circle-circle solve), Δ=%.3e", cb.center, want, got)
	}
	if cb.sphere.Radius != r {
		t.Fatalf("sphere radius %g, want %g", cb.sphere.Radius, r)
	}
}

// TestTwoCylinderHostCorner_CrossingAxisDeclines is the do-no-harm regression for simple/O4's class:
// a plane containing (not perpendicular to) the shared axis direction, OR non-parallel cylinder
// axes, must decline with the exact historical string — the crossing-axis corner is out of scope
// (a follow-on, per the R4 wave report).
func TestTwoCylinderHostCorner_CrossingAxisDeclines(t *testing.T) {
	t.Parallel()
	lin := topo.NewLineage(topo.Tok("test", "twocyl-corner-crossing", 0))
	bld := topo.NewBuilder(true, lin)
	v := bld.AddVertex(math.P3(0, 0, 0), lin)
	cylA, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 18)
	cylB, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 1, 0), 20) // crossing (perpendicular) axis
	faces := []*topo.Face{
		bld.AddFace(cylA, lin),
		bld.AddFace(planeOn(t, math.P3(0, 0, 25), math.V3(1, 0, 0)), lin),
		bld.AddFace(cylB, lin),
	}
	if _, err := solveBlend(nil, v, faces, 5); err == nil || err.Error() != "fillet: corner face must be planar" {
		t.Fatalf("crossing-axis two-cylinder corner: got err %v, want the historical decline string", err)
	}
}

// TestTwoCylCornerCenter_NonPerpendicularPlaneDeclines is the mutation witness for the plane⊥axis
// gate: tilting the cap plane off the shared axis (but keeping the cylinders parallel) must decline
// — proving the |n̂·â|≈1 check is load-bearing, not a no-op.
func TestTwoCylCornerCenter_NonPerpendicularPlaneDeclines(t *testing.T) {
	t.Parallel()
	lin := topo.NewLineage(topo.Tok("test", "twocyl-corner-tilted", 0))
	bld := topo.NewBuilder(true, lin)
	v := bld.AddVertex(math.P3(5, 12, 20), lin)
	cylA, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 18)
	cylB, _ := geom.NewCylinder(math.P3(14, 0, 0), math.V3(0, 0, 1), 20)
	tilted, _ := math.UnitVector3FromVector(math.V3(0.1, 0, 1)) // NOT parallel to ẑ
	faces := []*topo.Face{
		bld.AddFace(cylA, lin),
		bld.AddFace(planeOn(t, math.P3(0, 0, 25), tilted.AsVector()), lin),
		bld.AddFace(cylB, lin),
	}
	if _, err := solveBlend(nil, v, faces, 5); err == nil || err.Error() != "fillet: corner face must be planar" {
		t.Fatalf("tilted-plane two-cylinder corner: got err %v, want decline", err)
	}
}

// TestTwoCylCornerConsistent_RejectsPerturbedCentre is the certificate regression fence, sibling of
// TestConeCornerConsistent_RejectsPerturbedCentre.
func TestTwoCylCornerConsistent_RejectsPerturbedCentre(t *testing.T) {
	t.Parallel()
	v, faces, r := twoCylSyntheticFixture(t)
	cylFaces, planeFace, ok := twoParallelCylinderHostCorner(faces)
	if !ok {
		t.Fatalf("recognizer declined the synthetic fixture")
	}
	cylA := cylFaces[0].Geometry().(geom.Cylinder)
	cylB := cylFaces[1].Geometry().(geom.Cylinder)
	pl := planeFace.Geometry().(geom.Plane)
	res := twoCylCornerResolution(v, cylA, cylB, pl)
	center := math.P3(5, 12, 20)
	if !twoCylCornerConsistent(center, cylA, cylB, planeFace, pl, r, res) {
		t.Fatalf("certified centre %v rejected; want accept", center)
	}
	offPlane := center.TranslateBy(math.V3(0, 0, 1).Scale(1e-3))
	off := center.TranslateBy(math.V3(1, 0, 0).Scale(1e-2))
	if twoCylCornerConsistent(offPlane, cylA, cylB, planeFace, pl, r, res) {
		t.Fatalf("centre nudged off the plane accepted; want reject")
	}
	if twoCylCornerConsistent(off, cylA, cylB, planeFace, pl, r, res) {
		t.Fatalf("centre nudged off both cylinder axes accepted; want reject")
	}
}

// twoCylImportedCorner drives the real imported corpus fixture through solveBlend and checks the
// centre with an INDEPENDENT circle-circle recomputation (not intersectCoplanarCircles): both
// cylinder axes must be genuinely parallel to the plane's own axis-perpendicular reduction to hold,
// which the test verifies directly rather than assuming.
func twoCylImportedCorner(t *testing.T, rel string, near math.Point3, r float64) *cornerBlend {
	t.Helper()
	body := corpusFixture(t, rel)
	v := vertexNearest(t, body, near)
	faces := facesAtVertex(v)
	cylFaces, planeFace, ok := twoParallelCylinderHostCorner(faces)
	if !ok {
		t.Fatalf("%s corner at %v is not a [cylinder,plane,cylinder] host set", rel, near)
	}
	cylA := cylFaces[0].Geometry().(geom.Cylinder)
	cylB := cylFaces[1].Geometry().(geom.Cylinder)
	if stdmath.Abs(stdmath.Abs(float64(cylA.AxisDir.Dot(cylB.AxisDir)))-1) > 1e-9 {
		t.Fatalf("%s: cylinder axes are not parallel (dot=%v) — this fixture is NOT this corner's class", rel, cylA.AxisDir.Dot(cylB.AxisDir))
	}
	pl := planeFace.Geometry().(geom.Plane)
	if stdmath.Abs(stdmath.Abs(float64(cylA.AxisDir.AsVector().Dot(pl.Normal())))-1) > 1e-9 {
		t.Fatalf("%s: cap plane is not perpendicular to the shared axis", rel)
	}
	cb, err := solveBlend(nil, v, faces, r)
	if err != nil {
		t.Fatalf("%s: two-cylinder corner declined: %v", rel, err)
	}
	assertTwoCylCentreIndependent(t, cb.center, cylA, cylB, planeFace, pl, r)
	return cb
}

// assertTwoCylCentreIndependent recomputes distance-to-each-axis and plane distance directly (the
// SAME two identities twoCylCornerConsistent checks, but hand-written here rather than reusing it)
// — an independent certificate of the imported-fixture solve.
func assertTwoCylCentreIndependent(t *testing.T, c math.Point3, cylA, cylB geom.Cylinder, planeFace *topo.Face, pl geom.Plane, r float64) {
	t.Helper()
	n := outwardPlaneNormal(planeFace, pl)
	if d := stdmath.Abs(float64(pl.Origin.VectorTo(c).Dot(n))); stdmath.Abs(d-r) > 1e-6 {
		t.Fatalf("centre %v is %.9f from the cap plane, want %g", c, d, r)
	}
	for _, cyl := range []geom.Cylinder{cylA, cylB} {
		a := cyl.AxisDir.AsVector()
		w := cyl.Origin.VectorTo(c)
		dist := float64(w.Sub(a.Scale(w.Dot(a))).Length())
		if stdmath.Abs(dist-(cyl.Radius-r)) > 1e-6 && stdmath.Abs(dist-(cyl.Radius+r)) > 1e-6 {
			t.Fatalf("centre %v is %.9f from a cylinder axis (R=%g), want R∓%g", c, dist, cyl.Radius, r)
		}
	}
}

// TestTwoCylinderHostCorner_O9Imported certifies simple/O9's real corner (R4 wave): the corner PATCH
// now solves; the case still declines end-to-end in the corpus (fillet_miter_curved.go's "curved
// miter arms unsupported" gate at a companion 2-edge vertex is outside this file's ownership — see
// the R4 wave report), so this test certifies the corner math alone, matching the sibling files'
// "this file solves the SPHERE, the weld is a follow-on" precedent.
func TestTwoCylinderHostCorner_O9Imported(t *testing.T) {
	t.Parallel()
	twoCylImportedCorner(t, "simple/O9.step", math.P3(65, 2.303039929, 70), 5)
}

// TestTwoCylinderHostCorner_P7Imported is O9's sibling certification for simple/P7.
func TestTwoCylinderHostCorner_P7Imported(t *testing.T) {
	t.Parallel()
	twoCylImportedCorner(t, "simple/P7.step", math.P3(75, 6.698729811, 150), 5)
}

// TestTwoCylinderHostCorner_O4CrossingAxisImportedDeclines certifies simple/O4's REAL corner has
// crossing (non-parallel) cylinder axes — confirming the do-no-harm decline is the right call for
// this specific corpus fixture, not just the synthetic crossing-axis case above.
func TestTwoCylinderHostCorner_O4CrossingAxisImportedDeclines(t *testing.T) {
	t.Parallel()
	body := corpusFixture(t, "simple/O4.step")
	v := vertexNearest(t, body, math.P3(30, 4.174243050442, 90))
	faces := facesAtVertex(v)
	cylFaces, _, ok := twoParallelCylinderHostCorner(faces)
	if !ok {
		t.Fatalf("O4 corner is not a [cylinder,plane,cylinder] host set")
	}
	cylA := cylFaces[0].Geometry().(geom.Cylinder)
	cylB := cylFaces[1].Geometry().(geom.Cylinder)
	if dot := stdmath.Abs(float64(cylA.AxisDir.Dot(cylB.AxisDir))); dot > 1-1e-6 {
		t.Fatalf("O4's cylinder axes are parallel (dot=%v) — expected a genuine crossing-axis fixture", dot)
	}
	if _, err := solveBlend(nil, v, faces, 5); err == nil || err.Error() != "fillet: corner face must be planar" {
		t.Fatalf("O4 crossing-axis corner: got err %v, want the historical decline string", err)
	}
}
