// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati/math"
	"oblikovati/model/param"
	"oblikovati/model/sketch"
)

const wtol = 1e-9

func TestOriginCoordinateFrame(t *testing.T) {
	g := NewWorkGeometry()
	if g.WorkPoints().Count() != 1 || g.WorkAxes().Count() != 3 || g.WorkPlanes().Count() != 3 {
		t.Fatalf("origin frame counts pts=%d axes=%d planes=%d, want 1/3/3",
			g.WorkPoints().Count(), g.WorkAxes().Count(), g.WorkPlanes().Count())
	}
	// Every origin element is a grounded coordinate-system element.
	for i := 0; i < g.WorkPlanes().Count(); i++ {
		p := g.WorkPlanes().Item(i)
		if !p.IsCoordinateSystemElement() || !p.Grounded() {
			t.Errorf("origin plane %q not a grounded coordinate-system element", p.Name())
		}
	}
	// Well-known origin references resolve to the absolute frame.
	xy, err := g.plane(OriginXYPlane)
	if err != nil || !xy.Normal().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("origin XY plane normal = %v err=%v, want +Z", xy.Normal(), err)
	}
	c, err := g.point(OriginCenter)
	if err != nil || !c.IsEqualTo(math.P3(0, 0, 0), wtol) {
		t.Errorf("origin center = %v err=%v, want (0,0,0)", c, err)
	}
	x, err := g.axis(OriginXAxis)
	if err != nil || !x.Direction().AsVector().IsEqualTo(math.V3(1, 0, 0), wtol) {
		t.Errorf("origin X axis dir = %v err=%v, want +X", x.Direction(), err)
	}
}

func TestOffsetWorkPlaneMovesWithParameter(t *testing.T) {
	ps := param.NewParameters()
	off, _ := ps.AddUserParameter("gap", "5 cm")
	g := NewWorkGeometry()
	wp := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return off.ModelValue() })

	// XY plane offset 5 along +Z → origin at z=5.
	if !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 5), wtol) {
		t.Fatalf("offset plane origin = %v, want (0,0,5)", wp.Plane().Origin())
	}
	if !wp.Health().OK() {
		t.Error("offset plane should be healthy")
	}
	if wp.IsCoordinateSystemElement() {
		t.Error("a user work plane is not a coordinate-system element")
	}
	// Drive the parameter: the datum moves on recompute.
	if err := off.SetExpression("12 cm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	g.Recompute(nil)
	if !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 12), wtol) {
		t.Errorf("after param change, plane origin = %v, want (0,0,12)", wp.Plane().Origin())
	}
}

func TestThreePointWorkPlaneAndDegenerate(t *testing.T) {
	g := NewWorkGeometry()
	a := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 0) })
	b := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 0, 0) })
	c := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 1, 0) })
	wp := g.WorkPlanes().AddByThreePoints(a.Key(), b.Key(), c.Key())
	if !wp.Health().OK() || !wp.Plane().Normal().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("three-point plane normal = %v, want (0,0,1)", wp.Plane().Normal())
	}
	// Collinear points are degenerate → sick, not garbage.
	d := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(2, 0, 0) })
	bad := g.WorkPlanes().AddByThreePoints(a.Key(), b.Key(), d.Key())
	if bad.Health().OK() {
		t.Error("collinear three-point plane should be sick")
	}
}

func TestWorkAxisByTwoPointsAndPlaneIntersection(t *testing.T) {
	g := NewWorkGeometry()
	top := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 4) })
	ax := g.WorkAxes().AddByTwoPoints(OriginCenter, top.Key())
	if !ax.Direction().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("axis dir = %v, want +Z", ax.Direction())
	}
	// XY plane ∩ XZ plane = the X axis.
	inter := g.WorkAxes().AddByPlaneIntersection(OriginXYPlane, OriginXZPlane)
	if !inter.Health().OK() {
		t.Fatalf("plane-intersection axis sick: %+v", inter.Health())
	}
	if !inter.Direction().AsVector().IsParallelTo(math.V3(1, 0, 0), wtol) {
		t.Errorf("XY∩XZ axis dir = %v, want parallel to X", inter.Direction())
	}
	// Parallel planes do not intersect → sick.
	off := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 10 })
	par := g.WorkAxes().AddByPlaneIntersection(OriginXYPlane, off.Key())
	if par.Health().OK() {
		t.Error("parallel planes should yield a sick axis")
	}
}

func TestWorkPointPiercesPlane(t *testing.T) {
	g := NewWorkGeometry()
	wp := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(2, 3, 4) })
	if !wp.Point().IsEqualTo(math.P3(2, 3, 4), wtol) || !wp.Health().OK() {
		t.Errorf("work point = %v", wp.Point())
	}
	// The Z origin axis pierces the z=5 plane at (0,0,5).
	off := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 5 })
	pierce := g.WorkPoints().AddByPlaneAndAxisIntersection(off.Key(), OriginZAxis)
	if !pierce.Point().IsEqualTo(math.P3(0, 0, 5), 1e-6) {
		t.Errorf("pierce point = %v, want (0,0,5)", pierce.Point())
	}
	// The X axis is parallel to the z=5 plane → never pierces → sick.
	bad := g.WorkPoints().AddByPlaneAndAxisIntersection(off.Key(), OriginXAxis)
	if bad.Health().OK() {
		t.Error("axis parallel to plane should give a sick point")
	}
}

func TestWorkFeatureNamesAndKeys(t *testing.T) {
	g := NewWorkGeometry()
	wp := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 1 })
	wp.SetName("Datum1")
	if wp.Name() != "Datum1" || wp.Key() != WorkRef("plane/3") {
		t.Errorf("work plane name/key = %q/%q, want Datum1/plane/3", wp.Name(), wp.Key())
	}
	// The origin center is point 0, so the first user point is point/1.
	up := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 1, 1) })
	if up.Key() != WorkRef("point/1") {
		t.Errorf("first user point key = %q, want point/1", up.Key())
	}
}

func TestWorkPlaneVisibilityDefaultsAndToggles(t *testing.T) {
	g := NewWorkGeometry()
	if g.WorkPlanes().Item(0).Visible() {
		t.Error("origin plane should be hidden by default")
	}
	wp := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 1 })
	if !wp.Visible() {
		t.Error("a new user work plane should be visible by default")
	}
	wp.SetVisible(false)
	if wp.Visible() {
		t.Error("SetVisible(false) should hide the plane")
	}
}

func TestUserCoordinateSystem(t *testing.T) {
	ucs := NewUserCoordinateSystems().AddByPlane(offsetXY(5))
	ucs.SetName("Frame")
	if ucs.Name() != "Frame" || !ucs.Origin().IsEqualTo(math.P3(0, 0, 5), wtol) ||
		!ucs.ZAxis().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("UCS frame wrong: origin=%v z=%v", ucs.Origin(), ucs.ZAxis())
	}
}

// offsetXY returns the XY plane shifted up by z.
func offsetXY(z float64) sketch.Plane {
	p, _ := sketch.NewPlane(math.P3(0, 0, z), mustX(), mustY())
	return p
}

func mustX() math.UnitVector3 { u, _ := math.NewUnitVector3(1, 0, 0); return u }
func mustY() math.UnitVector3 { u, _ := math.NewUnitVector3(0, 1, 0); return u }
