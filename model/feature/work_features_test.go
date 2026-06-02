// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/param"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

const wtol = 1e-9

func TestOffsetWorkPlaneMovesWithParameter(t *testing.T) {
	ps := param.NewParameters()
	off, _ := ps.AddUserParameter("gap", "5 cm")
	planes := NewWorkPlanes()
	wp := planes.AddByPlaneAndOffset(sketch.XYPlane(), func() float64 { return off.ModelValue() })

	// XY plane offset 5 along +Z → origin at z=5.
	if !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 5), wtol) {
		t.Fatalf("offset plane origin = %v, want (0,0,5)", wp.Plane().Origin())
	}
	if !wp.Health().OK() {
		t.Error("offset plane should be healthy")
	}
	// Drive the parameter: the datum moves on recompute.
	if err := off.SetExpression("12 cm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	wp.Recompute()
	if !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 12), wtol) {
		t.Errorf("after param change, plane origin = %v, want (0,0,12)", wp.Plane().Origin())
	}
}

func TestWorkPlaneServesAsSketchPlane(t *testing.T) {
	planes := NewWorkPlanes()
	wp := planes.AddByPlaneAndOffset(sketch.XYPlane(), func() float64 { return 3 })
	// The datum plane is directly usable as a sketch host.
	s := sketch.NewSketches().Add(wp.Plane())
	got := s.ToModel(math.P2(1, 2)) // sketch (1,2) on the z=3 plane → (1,2,3)
	if !got.IsEqualTo(math.P3(1, 2, 3), wtol) {
		t.Errorf("sketch on work plane mapped to %v, want (1,2,3)", got)
	}
}

func TestThreePointWorkPlaneAndDegenerate(t *testing.T) {
	planes := NewWorkPlanes()
	wp := planes.AddByThreePoints(
		func() math.Point3 { return math.P3(0, 0, 0) },
		func() math.Point3 { return math.P3(1, 0, 0) },
		func() math.Point3 { return math.P3(0, 1, 0) },
	)
	if !wp.Health().OK() || !wp.Plane().Normal().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("three-point plane normal = %v, want (0,0,1)", wp.Plane().Normal())
	}
	// Collinear points are degenerate → sick, not garbage.
	bad := planes.AddByThreePoints(
		func() math.Point3 { return math.P3(0, 0, 0) },
		func() math.Point3 { return math.P3(1, 0, 0) },
		func() math.Point3 { return math.P3(2, 0, 0) },
	)
	if bad.Health().OK() {
		t.Error("collinear three-point plane should be sick")
	}
	if planes.Count() != 2 || planes.Item(0) != wp {
		t.Error("work plane collection tracking wrong")
	}
	if _, ok := planes.ByID(wp.ID()); !ok {
		t.Error("ByID failed")
	}
}

func TestWorkAxisByTwoPointsAndPlaneIntersection(t *testing.T) {
	axes := NewWorkAxes()
	ax := axes.AddByTwoPoints(
		func() math.Point3 { return math.P3(0, 0, 0) },
		func() math.Point3 { return math.P3(0, 0, 4) },
	)
	if !ax.Direction().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("axis dir = %v, want +Z", ax.Direction())
	}
	// XY plane ∩ XZ plane = the X axis.
	inter := axes.AddByPlaneIntersection(sketch.XYPlane(), sketch.XZPlane())
	if !inter.Health().OK() {
		t.Fatalf("plane-intersection axis sick: %+v", inter.Health())
	}
	if !inter.Direction().AsVector().IsParallelTo(math.V3(1, 0, 0), wtol) {
		t.Errorf("XY∩XZ axis dir = %v, want parallel to X", inter.Direction())
	}
	// Parallel planes do not intersect → sick.
	top := axes.AddByPlaneIntersection(sketch.XYPlane(), offsetXY(10))
	if top.Health().OK() {
		t.Error("parallel planes should yield a sick axis")
	}
	if axes.Count() != 3 || axes.Item(0) != ax {
		t.Error("axis collection tracking wrong")
	}
}

func TestWorkPointAndUCS(t *testing.T) {
	pts := NewWorkPoints()
	wp := pts.AddByPoint(func() math.Point3 { return math.P3(2, 3, 4) })
	if !wp.Point().IsEqualTo(math.P3(2, 3, 4), wtol) || !wp.Health().OK() {
		t.Errorf("work point = %v", wp.Point())
	}
	// Z axis through origin pierces the z=5 plane at (0,0,5).
	axes := NewWorkAxes()
	zAxis := axes.AddByTwoPoints(func() math.Point3 { return math.P3(0, 0, 0) }, func() math.Point3 { return math.P3(0, 0, 1) })
	pierce := pts.AddByPlaneAndAxisIntersection(offsetXY(5), zAxis)
	if !pierce.Point().IsEqualTo(math.P3(0, 0, 5), 1e-6) {
		t.Errorf("pierce point = %v, want (0,0,5)", pierce.Point())
	}
	// An axis parallel to the plane never pierces it → sick.
	xAxis := axes.AddByTwoPoints(func() math.Point3 { return math.P3(0, 0, 0) }, func() math.Point3 { return math.P3(1, 0, 0) })
	if pts.AddByPlaneAndAxisIntersection(offsetXY(5), xAxis).Health().OK() {
		t.Error("axis parallel to plane should give a sick point")
	}
	if pts.Count() != 3 {
		t.Errorf("point collection count = %d, want 3", pts.Count())
	}

	ucs := NewUserCoordinateSystems()
	frame := ucs.AddByPlane(offsetXY(5))
	if !frame.Origin().IsEqualTo(math.P3(0, 0, 5), wtol) || !frame.ZAxis().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("UCS frame wrong: origin=%v z=%v", frame.Origin(), frame.ZAxis())
	}
	if !frame.XYPlane().Origin().IsEqualTo(math.P3(0, 0, 5), wtol) || ucs.Count() != 1 || ucs.Item(0) != frame {
		t.Error("UCS plane / collection wrong")
	}
}

// offsetXY returns the XY plane shifted up by z.
func offsetXY(z float64) sketch.Plane {
	p, _ := sketch.NewPlane(math.P3(0, 0, z), mustX(), mustY())
	return p
}

func mustX() math.UnitVector3 { u, _ := math.NewUnitVector3(1, 0, 0); return u }
func mustY() math.UnitVector3 { u, _ := math.NewUnitVector3(0, 1, 0); return u }

func TestWorkFeatureAccessors(t *testing.T) {
	planes := NewWorkPlanes()
	wp := planes.AddByPlaneAndOffset(sketch.XYPlane(), func() float64 { return 1 })
	wp.SetName("Datum1")
	if wp.Name() != "Datum1" {
		t.Error("work plane SetName/Name wrong")
	}

	axes := NewWorkAxes()
	ax := axes.AddByTwoPoints(func() math.Point3 { return math.P3(0, 0, 0) }, func() math.Point3 { return math.P3(1, 0, 0) })
	if ax.ID() == 0 || ax.Name() != "WorkAxis" || !ax.Origin().IsEqualTo(math.P3(0, 0, 0), wtol) || axes.Item(0) != ax {
		t.Error("work axis accessors wrong")
	}

	pts := NewWorkPoints()
	p := pts.AddByPoint(func() math.Point3 { return math.P3(1, 1, 1) })
	if p.ID() == 0 || p.Name() != "WorkPoint" {
		t.Error("work point accessors wrong")
	}

	ucs := NewUserCoordinateSystems().AddByPlane(sketch.XYPlane())
	ucs.SetName("Frame")
	if ucs.ID() == 0 || ucs.Name() != "Frame" ||
		!ucs.XAxis().AsVector().IsEqualTo(math.V3(1, 0, 0), wtol) ||
		!ucs.YAxis().AsVector().IsEqualTo(math.V3(0, 1, 0), wtol) {
		t.Error("UCS accessors wrong")
	}
}
