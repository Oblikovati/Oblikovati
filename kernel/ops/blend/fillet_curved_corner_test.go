// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// nearly reports whether two scalars agree to a generous absolute tolerance (the fixtures are exact;
// the slack only absorbs the OCCT oracle's 2-decimal BREP rounding and floating-point round-off).
func nearly(got, want float64) bool { return stdmath.Abs(got-want) < 1e-2 }

// nearlyPt reports whether two points agree component-wise under nearly.
func nearlyPt(got, want math.Point3) bool {
	return nearly(float64(got.X), float64(want.X)) &&
		nearly(float64(got.Y), float64(want.Y)) &&
		nearly(float64(got.Z), float64(want.Z))
}

// TestSolveBlend_B3CurvedCorner drives the curved-host corner solve on the REAL corpus fixture: B3's
// three picked edges (top-rim arc, vertical wall line, top radial segment, all r=10) meet at the
// [Cylinder,Plane,Plane] vertex (0,−50,100). computeCorners must now return ONE cornerBlend carrying an
// analytic geom.Sphere of radius 10 tangent to the R=50 wall (centre 40 = R−r from the z-axis) and 10
// inside both planes — centre (10, −38.73, 90) — instead of the old "corner face must be planar" reject.
//
// The asserted centre is the committed fixture's frame: the derivation's DRAWEXE run reported
// (38.73, 10, 90), which is THIS point rotated −90° about z (a trotate-frame difference). Both are the
// same OCCT corner KPart (BREP surface code 4) — radius r, centre 40 from the axis, z=90.
func TestSolveBlend_B3CurvedCorner(t *testing.T) {
	t.Parallel()
	body := importCorpusSolid(t, "simple/B3")
	picks := cornerEdgePicks(t, body, math.P3(0, -50, 100), 10)
	blends, _, err := computeCorners(body, picks)
	if err != nil {
		t.Fatalf("computeCorners on B3 curved corner errored (still rejecting curved host): %v", err)
	}
	cb := singleBlend(t, blends)
	if !nearly(cb.sphere.Radius, 10) || !nearlyPt(cb.center, math.P3(10, -stdmath.Sqrt(1500), 90)) {
		t.Fatalf("B3 corner sphere = {center %v r %.6f}, want center (10,-38.7298,90) r10 (OCCT KPart code 4)",
			cb.center, cb.sphere.Radius)
	}
	assertOnCylinderWall(t, cb.center, body)
}

// n7CornerInputs is the named fake for N7's tangent-degenerate trihedral corner, taken straight from
// the committed OCCT fixture (not hand-authored coords — the real solve inputs, so the fake cannot
// model impossible topology). Wall cylinder R=50 axis ẑ at (50,50); the x=50 diametral plane (through
// the axis) and the z=10 plane; V=(50,0,10); r=5. curvedCornerCenter picks the WRONG reflected root
// (centre z=5, wall-tangent z=5) today; the correct root is z=15 (oracle corner face at z∈[5,15]).
func n7CornerInputs(t *testing.T) (geom.Cylinder, [2]*topo.Face, *topo.Vertex, float64) {
	t.Helper()
	return cornerHostInputs(t, "simple/N7", math.P3(50, 0, 10), 5)
}

// cleanOctantInputs is the named fake for B3's clean (non-tangent) trihedral corner from the fixture:
// wall R=50 axis ẑ at origin, cap z=100 and radial x=0 planes, V=(0,−50,100), r=10. Its in-domain
// root IS the nearer-vertex root the legacy solve returns (centre (10,−√1500,90)) — the reduction case.
func cleanOctantInputs(t *testing.T) (geom.Cylinder, [2]*topo.Face, *topo.Vertex, float64) {
	t.Helper()
	return cornerHostInputs(t, "simple/B3", math.P3(0, -50, 100), 10)
}

// cornerHostInputs imports a corpus fixture and returns the curved trihedral corner at point p: the
// wall cylinder, its two planar hosts, the corner vertex, and radius r — exactly what solveCurvedBlend
// receives (facesAtVertex → cylinderHostCorner), so the tests exercise the real call path.
func cornerHostInputs(t *testing.T, rel string, p math.Point3, r float64) (geom.Cylinder, [2]*topo.Face, *topo.Vertex, float64) {
	t.Helper()
	v := vertexNear(t, importCorpusSolid(t, rel), p)
	cyl, planes, ok := cylinderHostCorner(facesAtVertex(v))
	if !ok {
		t.Fatalf("%s corner at %v is not [cylinder,plane,plane]", rel, p)
	}
	return cyl, planes, v, r
}

// TestCurvedCornerCenter_PicksInDomainRootAtTangentDihedron pins the N7 fix: the ball must root at the
// z=15 reflected root (wall-tangent z=15), NOT the nearer-vertex z=5 root the legacy tiebreak returns
// (which yields corner area 42 vs the oracle 90.19). See n7-runout-rederivation.md §tangent-dihedron.
func TestCurvedCornerCenter_PicksInDomainRootAtTangentDihedron(t *testing.T) {
	t.Parallel()
	cyl, planes, v, r := n7CornerInputs(t)
	res := curvedCornerResolution(v, cyl, planes)
	c, ok := curvedCornerCenter(cyl, planes, r, 1, v, res)
	if !ok {
		t.Fatalf("curvedCornerCenter declined the N7 tangent-dihedron corner")
	}
	if got := float64(cylinderWallPoint(cyl, c).Z); stdmath.Abs(got-15) > res.Weld()*r {
		t.Fatalf("ball mis-rooted: wall-tangent z=%.6f; want z=15 (reflected z=5 root gives corner area 42 vs oracle 90.19)", got)
	}
}

// TestCurvedCornerCenter_CleanOctantUnchanged is the B3 reduction: on a clean (non-tangent) corner the
// in-domain root IS the legacy nearer-vertex root, so the returned centre is UNCHANGED — (10,−√1500,90),
// wall-tangent z=90. This guards byte-faithfulness: the new root selection must not perturb B3.
func TestCurvedCornerCenter_CleanOctantUnchanged(t *testing.T) {
	t.Parallel()
	cyl, planes, v, r := cleanOctantInputs(t)
	res := curvedCornerResolution(v, cyl, planes)
	c, ok := curvedCornerCenter(cyl, planes, r, 1, v, res)
	if !ok {
		t.Fatalf("curvedCornerCenter declined the clean B3 octant")
	}
	if !nearlyPt(c, math.P3(10, -stdmath.Sqrt(1500), 90)) {
		t.Fatalf("clean octant centre = %v, want (10,−38.7298,90) (legacy nearer-vertex root, unchanged)", c)
	}
}

// assertOnCylinderWall confirms the corner centre sits at distance R−r (=40) from B3's boss axis — the
// convex/concave discrimination: an R+r=60 (concave) or mis-signed centre fails here, not just the
// nominal coordinate check.
func assertOnCylinderWall(t *testing.T, c math.Point3, body *topo.Body) {
	t.Helper()
	cyl := b3BossWall(t, body)
	w := cyl.Origin.VectorTo(c)
	perp := w.Sub(cyl.AxisDir.AsVector().Scale(w.Dot(cyl.AxisDir.AsVector())))
	if d := float64(perp.Length()); !nearly(d, 40) {
		t.Fatalf("corner centre is %.4f from the boss axis, want R−r = 40 (convex)", d)
	}
}

// b3BossWall returns B3's R=50 boss-wall cylinder geometry.
func b3BossWall(t *testing.T, body *topo.Body) geom.Cylinder {
	t.Helper()
	for _, f := range body.Faces() {
		if c, ok := f.Geometry().(geom.Cylinder); ok {
			return c
		}
	}
	t.Fatalf("B3 boss-wall cylinder face not found")
	return geom.Cylinder{}
}
