// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/depend"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

const wtol = 1e-9

func TestOriginCoordinateFrame(t *testing.T) {
	t.Parallel()
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

// With a footprint tracker injected, recomputing an offset work plane must record the
// parameter its offset closure read, so a sketch hosted on the plane can attribute an offset
// edit to its features instead of forcing a wholesale rebuild (ADR-0044). A plane with no
// tracker (or a grounded origin plane) records nothing.
func TestOffsetWorkPlaneRecordsParameterFootprint(t *testing.T) {
	t.Parallel()
	ps := param.NewParameters()
	off, _ := ps.AddUserParameter("gap", "5 cm")
	g := NewWorkGeometry()
	g.SetFootprintTracker(ps)
	wp := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return off.ModelValue() })

	g.Recompute(nil)

	fp := wp.ParameterFootprint()
	want := depend.Key{Kind: depend.ParameterKey, ID: uint64(off.ID())}
	if len(fp) != 1 || fp[0] != want {
		t.Errorf("offset plane footprint = %v, want one ParameterKey for gap", fp)
	}
	// The grounded origin planes read no parameter, so their footprints stay empty.
	for _, origin := range g.OriginPlanes() {
		if len(origin.ParameterFootprint()) != 0 {
			t.Errorf("origin plane %q footprint = %v, want empty", origin.Name(), origin.ParameterFootprint())
		}
	}
}

func TestOffsetWorkPlaneMovesWithParameter(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestWorkAxisByLine covers the grounded "line" axis: a fixed origin + direction, tracked as a
// user axis that lists as the "line" kind (not an origin coordinate-system element).
func TestWorkAxisByLine(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	before := g.WorkAxes().Count()
	ax := g.WorkAxes().AddByLine(math.P3(1, 2, 3), mustUnit(0, 0, 1))
	if g.WorkAxes().Count() != before+1 {
		t.Fatalf("axis count = %d, want %d", g.WorkAxes().Count(), before+1)
	}
	if !ax.Health().OK() {
		t.Fatalf("line axis sick: %+v", ax.Health())
	}
	if !ax.Origin().IsEqualTo(math.P3(1, 2, 3), wtol) {
		t.Errorf("line axis origin = %v, want (1,2,3)", ax.Origin())
	}
	if !ax.Direction().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("line axis dir = %v, want +Z", ax.Direction())
	}
	if ax.Kind() != "line" || ax.IsCoordinateSystemElement() {
		t.Errorf("line axis Kind=%q origin=%v, want kind=line and not an origin element", ax.Kind(), ax.IsCoordinateSystemElement())
	}
}

// TestWorkAxisLineSerializeRoundTrip pins that a grounded line axis persists by its origin +
// direction and restores to the same geometry (serializeAxisDef / restoreLineAxis).
func TestWorkAxisLineSerializeRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	g.WorkAxes().AddByLine(math.P3(1, 2, 3), mustUnit(0, 1, 0))
	d, err := serializeAxisDef(fixedAxisDef{origin: math.P3(1, 2, 3), dir: mustUnit(0, 1, 0)})
	if err != nil {
		t.Fatalf("serialize line axis: %v", err)
	}
	if d.Kind != "line" || len(d.Position) != 3 || len(d.XAxis) != 3 {
		t.Fatalf("serialized line axis = %+v, want kind=line with a position and direction", d)
	}
	restored := NewWorkGeometry()
	if err := restoreAxisFeature(restored.WorkAxes(), d); err != nil {
		t.Fatalf("restore line axis: %v", err)
	}
	ax := restored.WorkAxes().Item(restored.WorkAxes().Count() - 1)
	if !ax.Origin().IsEqualTo(math.P3(1, 2, 3), wtol) || !ax.Direction().AsVector().IsEqualTo(math.V3(0, 1, 0), wtol) {
		t.Errorf("restored line axis = origin %v dir %v, want (1,2,3) / +Y", ax.Origin(), ax.Direction())
	}
}

func TestWorkPointPiercesPlane(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestShownForHostPickRevealsOnlyGroundedOrigins pins the #1752 rule the app-side picker and the
// viewport overlay share: a hidden grounded origin plane is shown ONLY while a datum host is being
// picked, a visible plane is always shown, and a hidden USER plane is never revealed — the reveal is
// scoped to the origin frame so Create Sketch does not un-hide planes the user chose to hide.
func TestShownForHostPickRevealsOnlyGroundedOrigins(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	origin := g.WorkPlanes().Item(0) // grounded, hidden by default
	if origin.ShownForHostPick(false) {
		t.Error("a hidden origin plane must not show without a host pick (#1520 guard)")
	}
	if !origin.ShownForHostPick(true) {
		t.Error("a hidden origin plane must show while revealing for a host pick (#1752)")
	}

	user := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 1 })
	user.SetVisible(false) // the user hid it
	if user.ShownForHostPick(true) {
		t.Error("a hidden USER plane must not be revealed — the reveal is scoped to the grounded origin frame")
	}
	user.SetVisible(true)
	if !user.ShownForHostPick(false) {
		t.Error("a visible plane is always shown regardless of the host-pick reveal")
	}
}

func TestUserCoordinateSystem(t *testing.T) {
	t.Parallel()
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
