// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// White-box tests for the far-end trim's ±BRANCH choice (stopFaceAxialSide / nearestHitOnSide,
// fillet_farend_trim.go). The configuration under test is complex/D8's: a fillet band whose stop wall is a
// corner cylinder TANGENT to the filleted face at the terminal vertex, so the wall's axis lies IN the section
// plane and the two crossings of every ruling are EXACTLY equidistant.

// d8CornerWall reproduces complex/D8's stop wall: the r=24 corner cylinder of the rounded-rectangle plate,
// vertical axis, its axis passing through the terminal section plane y = 59.093784332275.
func d8CornerWall(t *testing.T) geom.Cylinder {
	t.Helper()
	cyl, err := geom.NewCylinder(math.P3(223.39418029785, 59.093784332275, -90), math.V3(0, 0, 1), 24)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	return cyl
}

// d8SectionCap reproduces D8's flat terminal section cap: the r=30 quarter arc from the wall contact
// (247.394, ·, −20) to the top-plane contact (217.394, ·, 10), in the plane y = 59.093784332275.
func d8SectionCap(t *testing.T) geom.Arc3d {
	t.Helper()
	const vy, cx, cz, r = 59.093784332275, 217.39418029785, -20.0, 30.0
	mid := math.P3(cx+r*stdmath.Cos(stdmath.Pi/4), vy, cz+r*stdmath.Sin(stdmath.Pi/4))
	arc, err := geom.Arc3dByThreePoints(math.P3(cx+r, vy, cz), mid, math.P3(cx, vy, cz+r))
	if err != nil {
		t.Fatalf("Arc3dByThreePoints: %v", err)
	}
	return arc
}

// TestSlideOntoWallTieIsExactOnASymmetricWall is the MEASUREMENT that falsifies "nearest wins": on a wall
// whose axis lies in the section plane the two crossings are equidistant to the last bit, so the pre-fix
// rule had no information at all and its answer came from the intersector's output order.
func TestSlideOntoWallTieIsExactOnASymmetricWall(t *testing.T) {
	t.Parallel()
	wall := d8CornerWall(t)
	p := math.P3(240, 59.093784332275, -5) // a mid-cap station, 16.6 from the wall axis
	axis := math.V3(0, 1, 0)
	seg := geom.NewLineSegment(p.TranslateBy(axis.Scale(-462.9)), p.TranslateBy(axis.Scale(462.9)))
	hits := geom.IntersectCurveSurface(seg, wall)
	if len(hits) != 2 {
		t.Fatalf("got %d crossings, want the cylinder's 2", len(hits))
	}
	d0, d1 := hits[0].DistanceTo(p), hits[1].DistanceTo(p)
	if stdmath.Abs(d0-d1) > 1e-12 {
		t.Fatalf("crossings are %.17g and %.17g from the station — not a tie, so this is the wrong fixture", d0, d1)
	}
	if sideSignOf(hits[0], p, axis) == sideSignOf(hits[1], p, axis) {
		t.Fatal("both crossings on one side of the section plane: the tie cannot be a branch ambiguity")
	}
}

// sideSignOf is the sign of h's axial offset from p.
func sideSignOf(h, p math.Point3, axis math.Vector3) int {
	if p.VectorTo(h).Dot(axis) > 0 {
		return 1
	}
	return -1
}

// TestSlideSectionOntoWallStaysOnTheStopFaceBranch is the core regression: with the stop face's side supplied,
// every station lands on THAT side, the axial offsets are smooth (no zigzag), and each station sits on the
// analytic branch y = vy − √(R²−(x−cx)²) to machine precision. With side 0 — the pre-fix rule — the list is
// allowed to jump, and this test records that it does.
func TestSlideSectionOntoWallStaysOnTheStopFaceBranch(t *testing.T) {
	t.Parallel()
	const vy, wallX, R = 59.093784332275, 223.39418029785, 24.0
	wall, arc := d8CornerWall(t), d8SectionCap(t)
	slide := axialSlide{axis: math.V3(0, 1, 0), reach: 462.8433495, side: -1, tol: 1e-6}
	pts, ok := slideSectionOntoWall(arc, wall, slide)
	if !ok {
		t.Fatal("slideSectionOntoWall declined")
	}
	prev, flips := 0.0, 0
	for i, p := range pts {
		dy := float64(p.Y) - vy
		if dy > 1e-9 {
			t.Fatalf("station %d landed at dy %+.9g — on the far branch, not the stop face's side", i, dy)
		}
		want := -stdmath.Sqrt(stdmath.Max(R*R-(float64(p.X)-wallX)*(float64(p.X)-wallX), 0))
		if stdmath.Abs(dy-want) > 1e-9 {
			t.Errorf("station %d: dy %.12g, analytic branch %.12g", i, dy, want)
		}
		if i > 0 && stdmath.Abs(dy-prev) > 2.0 { // neighbouring stations of the true branch differ by ≤1.4
			flips++
		}
		prev = dy
	}
	if flips != 0 {
		t.Errorf("%d jumps between stations: the station list is not one smooth branch", flips)
	}
}

// TestSlideSectionOntoWallZigzagsWithoutTheSideRule pins WHY the side rule is needed rather than merely
// asserting the fixed behaviour: with side 0 the same call produces a list that crosses the section plane.
// (It records the defect; it does not endorse it — side 0 remains the honest fallback for a stop face that
// reaches both sides, where "nearest" is the only information available.)
func TestSlideSectionOntoWallZigzagsWithoutTheSideRule(t *testing.T) {
	t.Parallel()
	const vy = 59.093784332275
	wall, arc := d8CornerWall(t), d8SectionCap(t)
	pts, ok := slideSectionOntoWall(arc, wall, axialSlide{axis: math.V3(0, 1, 0), reach: 462.8433495, tol: 1e-6})
	if !ok {
		t.Fatal("slideSectionOntoWall declined")
	}
	crossed := false
	for _, p := range pts {
		if float64(p.Y)-vy > 1e-9 {
			crossed = true
		}
	}
	if !crossed {
		t.Skip("the unguided intersector happens to return one branch on this build; the side rule is what " +
			"makes it deterministic")
	}
}

// TestStopFaceAxialSideReadsTheFacesOwnExtent pins the side predicate on a real *topo.Face: the z=0 face of a
// 10-cube lies wholly IN the plane through the origin normal to Z (side 0, the stop-plane case that keeps the
// whole corpus byte-identical), reaches +X from the origin (side +1), and reaches −X from the far corner
// (side −1).
func TestStopFaceAxialSideReadsTheFacesOwnExtent(t *testing.T) {
	t.Parallel()
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	bottom := faceAtZ(t, box, 0)
	cases := []struct {
		name   string
		vertex math.Point3
		axis   math.Vector3
		want   int
	}{
		{"face lies in the section plane", math.P3(0, 0, 0), math.V3(0, 0, 1), 0},
		{"face reaches +X of the vertex", math.P3(0, 0, 0), math.V3(1, 0, 0), 1},
		{"face reaches -X of the vertex", math.P3(10, 0, 0), math.V3(1, 0, 0), -1},
		{"vertex inside the face's span", math.P3(5, 0, 0), math.V3(1, 0, 0), 0},
	}
	for _, tc := range cases {
		if got := stopFaceAxialSide(bottom, tc.vertex, tc.axis, 1e-9); got != tc.want {
			t.Errorf("%s: stopFaceAxialSide = %d, want %d", tc.name, got, tc.want)
		}
	}
}

// faceAtZ returns the block's face whose plane sits at the given z with a Z normal.
func faceAtZ(t *testing.T, body *topo.Body, z float64) *topo.Face {
	t.Helper()
	for _, f := range body.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if ok && stdmath.Abs(float64(pl.Origin.Z)-z) < 1e-12 && stdmath.Abs(pl.Normal().Dot(math.V3(0, 0, 1))) > 0.99 {
			return f
		}
	}
	t.Fatalf("no planar face at z=%g", z)
	return nil
}
