// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// b5Wall is B5's stop wall: the r=50 cylinder about Z the 270°-sector host is cut from. Every expectation
// below is the closed form DRAWEXE agrees with — e.g. the cap-side contact slides to √(50²−10²)=48.9898.
func b5Wall(t *testing.T) geom.Cylinder {
	t.Helper()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 50)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	return cyl
}

// TestSlideOntoWallHitsTheClosedFormCrossing pins the core of the trim: sliding a section point along the
// band's own ruling lands it exactly where the wall's implicit equation says, for each analytic wall family.
func TestSlideOntoWallHitsTheClosedFormCrossing(t *testing.T) {
	axisX := math.V3(1, 0, 0)
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), 150)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	// C4's host: r=90 at z=0 shrinking to r=40 at z=150, i.e. apex (0,0,270), half-angle atan(1/3),
	// so R(z) = (270−z)/3.
	cone, err := geom.NewCone(math.P3(0, 0, 270), math.V3(0, 0, -1), stdmath.Atan2(1, 3))
	if err != nil {
		t.Fatalf("NewCone: %v", err)
	}
	cases := []struct {
		name string
		from math.Point3
		axis math.Vector3
		wall geom.Surface
		want math.Point3
	}{
		{"cylinder cap contact slides to sqrt(2500-100)", math.P3(50, 10, 100), axisX, b5Wall(t),
			math.P3(stdmath.Sqrt(2400), 10, 100)},
		{"cylinder flank contact already on the wall", math.P3(50, 0, 90), axisX, b5Wall(t),
			math.P3(50, 0, 90)},
		{"sphere r=150 at y=10,z=129.9038 slides to sqrt(22500-100-z^2)", math.P3(75, 10, 129.903810567666), axisX, sphere,
			math.P3(stdmath.Sqrt(22500-100-129.903810567666*129.903810567666), 10, 129.903810567666)},
		{"cone R(z)=(270-z)/3 at y=10,z=140 slides to sqrt(R(140)^2-100)", math.P3(40, 10, 140), axisX, cone,
			math.P3(stdmath.Sqrt((270-140.0)/3*((270-140.0)/3)-100), 10, 140)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := slideOntoWall(tc.from, tc.axis, tc.wall, 200)
			if !ok {
				t.Fatalf("slideOntoWall(%v) declined", tc.from)
			}
			if d := got.DistanceTo(tc.want); d > 1e-9 {
				t.Errorf("slid to %v, want %v (off by %.3g)", got, tc.want, d)
			}
		})
	}
}

// TestSlideOntoWallDeclinesWhenTheRulingMissesTheWall pins the honest decline: a ruling parallel to the wall
// (or one whose crossing lies outside the band's own axial span) must leave the section alone.
func TestSlideOntoWallDeclinesWhenTheRulingMissesTheWall(t *testing.T) {
	wall := b5Wall(t)
	if _, ok := slideOntoWall(math.P3(0, 0, 0), math.V3(0, 0, 1), wall, 200); ok {
		t.Error("a ruling ALONG the cylinder axis has no crossing, want decline")
	}
	if _, ok := slideOntoWall(math.P3(50, 10, 100), math.V3(1, 0, 0), wall, 0.5); ok {
		t.Error("the crossing is 1.01 away but the band's span is 0.5, want decline")
	}
}

// TestSectionLeavesWallGatesOnlyOffWallCaps is the byte-identity gate: a flat section cap that already lies
// on its stop face must NOT be trimmed (that is what keeps every correctly-built band unchanged), while B5's
// run-on cap must be.
func TestSectionLeavesWallGatesOnlyOffWallCaps(t *testing.T) {
	onWall, err := geom.Arc3dByThreePoints(math.P3(50, 0, 90), math.P3(35.35533905932738, 35.35533905932738, 90), math.P3(0, 50, 90))
	if err != nil {
		t.Fatalf("Arc3dByThreePoints: %v", err)
	}
	if sectionLeavesWall(onWall, b5Wall(t), 1e-7) {
		t.Error("a section already on the wall reads as leaving it — the trim would perturb a correct band")
	}
	runOn, err := geom.Arc3dByThreePoints(math.P3(50, 0, 90), math.P3(50, 2.9289321881345254, 97.07106781186548), math.P3(50, 10, 100))
	if err != nil {
		t.Fatalf("Arc3dByThreePoints: %v", err)
	}
	if !sectionLeavesWall(runOn, b5Wall(t), 1e-7) {
		t.Error("B5's run-on cap (0.990195 off the r=50 cylinder) reads as on the wall")
	}
}

// TestExtendableWallAcceptsOnlyImplicitWalls pins which stop faces the slide may use: the analytic families,
// whose implicit form extends past the face's own trim. A fitted patch must be refused — ClosestPointOnSurface
// clamps to its parametric box, so a slide off the patch would converge onto the patch BOUNDARY.
func TestExtendableWallAcceptsOnlyImplicitWalls(t *testing.T) {
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	for _, s := range []geom.Surface{pl, b5Wall(t)} {
		if !extendableWall(s) {
			t.Errorf("%T rejected, want accepted", s)
		}
	}
	if extendableWall(geom.BSplineSurface{}) {
		t.Error("a fitted BSpline patch accepted as an extendable wall")
	}
}

// TestBandAxialSpanIsTheCentreSeparationAlongTheAxis pins the slide cap.
func TestBandAxialSpanIsTheCentreSeparationAlongTheAxis(t *testing.T) {
	c0 := corner{cen: math.P3(10, 10, 90)}
	c1 := corner{cen: math.P3(50, 10, 90)}
	if got := bandAxialSpan(c0, c1, math.V3(1, 0, 0)); stdmath.Abs(got-40) > 1e-12 {
		t.Errorf("bandAxialSpan = %.12g, want 40", got)
	}
	if got := bandAxialSpan(c0, c1, math.V3(0, 0, 1)); got != 0 {
		t.Errorf("bandAxialSpan across the axis = %.12g, want 0", got)
	}
}

// TestPlainWallStopExcludesEveryNonWallEnd pins the dispatch: only a plain end-face round is trimmed here.
// A blend / miter / run-out end has no stop face, and a variable fillet's chorded or conic section is emitted
// by the ruled-strip path with its own end geometry.
func TestPlainWallStopExcludesEveryNonWallEnd(t *testing.T) {
	face := someFaceOf(t)
	cases := []struct {
		name string
		c    corner
		want bool
	}{
		{"plain wall stop", corner{endFace: face}, true},
		{"no end face", corner{}, false},
		{"blend corner", corner{endFace: face, blend: true}, false},
		{"miter corner", corner{endFace: face, miter: true}, false},
		{"run-out apex", corner{endFace: face, runout: true}, false},
		{"variable chorded section", corner{endFace: face, chords: []math.Point3{math.P3(0, 0, 0)}}, false},
		{"conic section", corner{endFace: face, crossW: 0.7}, false},
	}
	for _, tc := range cases {
		if got := plainWallStop(tc.c); got != tc.want {
			t.Errorf("%s: plainWallStop = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// someFaceOf returns any face of a unit block — a real *topo.Face for the dispatch table above.
func someFaceOf(t *testing.T) *topo.Face {
	t.Helper()
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return box.Faces()[0]
}
