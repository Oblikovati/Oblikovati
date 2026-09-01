// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Create→redefine closure over every work-plane definition kind (#1634, audit I11): for each
// kind, a plane created with one set of inputs and then redefined (slots re-picked / scalars
// edited) to a second set must land exactly where a fresh create with the second set lands —
// proving create and redefine share one dispatch path (the definition). A kind creatable but
// not redefinable fails here; a kind missing the redefine methods fails to compile
// (work_plane_redefine.go's assertion block).

// redefineClosureCase is one kind's round-trip: create with the original inputs, perturb via
// the public redefine surface, and create the expected plane fresh from the perturbed inputs.
type redefineClosureCase struct {
	kind    string
	create  func(t *testing.T, g *WorkGeometry) *WorkPlane
	perturb func(t *testing.T, g *WorkGeometry, wp *WorkPlane)
	fresh   func(t *testing.T, g *WorkGeometry) *WorkPlane
}

// setSlot re-picks slot i of wp to ref, failing the test on a rejected reference.
func setSlot(t *testing.T, wp *WorkPlane, i int, ref WorkRef) {
	t.Helper()
	slots := wp.RedefineSlots()
	if i >= len(slots) {
		t.Fatalf("%s: RedefineSlots has %d slots, wanted index %d", wp.Kind(), len(slots), i)
	}
	if err := slots[i].Set(ref); err != nil {
		t.Fatalf("%s: slot %d (%q) Set: %v", wp.Kind(), i, slots[i].Label, err)
	}
}

// closureFixture is the shared geometry every case picks from: two work points on either
// side of the origin frame and a cylinder body for the tangent kinds.
type closureFixture struct {
	g          *WorkGeometry
	bodies     []*topo.Body
	pA, pB, pC *WorkPoint
	axisZ4     *WorkAxis // +Z line at x=4, outside the r=2 cylinder
	axisZ5     *WorkAxis // +Z line at x=5, the perturbed pick
	cylKey     []byte
}

func newClosureFixture(t *testing.T) *closureFixture {
	t.Helper()
	g := NewWorkGeometry()
	body, key := faceBody(t, mustCylinder(t)) // axis +Z, radius 2 at origin
	bodies := []*topo.Body{body}
	g.Recompute(bodies)
	f := &closureFixture{g: g, bodies: bodies, cylKey: key}
	f.pA = g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(2, 0, 0) })
	f.pB = g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 2, 0) })
	f.pC = g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 3) })
	a4 := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(4, 0, 0) })
	b4 := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(4, 0, 1) })
	f.axisZ4 = g.WorkAxes().AddByTwoPoints(a4.Key(), b4.Key())
	a5 := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 5, 0) })
	b5 := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 5, 1) })
	f.axisZ5 = g.WorkAxes().AddByTwoPoints(a5.Key(), b5.Key())
	return f
}

// redefineClosureCases enumerates every redefinable definition kind with its perturbation.
func redefineClosureCases(f *closureFixture) []redefineClosureCase {
	return append(referenceKindCases(f), tangentKindCases(f)...)
}

// referenceKindCases covers the kinds built on other work features (planes/axes/points).
func referenceKindCases(f *closureFixture) []redefineClosureCase {
	return []redefineClosureCase{
		{
			kind: "plane-offset",
			create: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 2 })
			},
			perturb: func(t *testing.T, g *WorkGeometry, wp *WorkPlane) {
				setSlot(t, wp, 0, OriginXZPlane)
				wp.EditableScalars()[0].Set(5)
			},
			fresh: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByPlaneAndOffset(OriginXZPlane, func() float64 { return 5 })
			},
		},
		{
			kind: "three-points",
			create: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByThreePoints(f.pA.Key(), f.pB.Key(), OriginCenter)
			},
			perturb: func(t *testing.T, g *WorkGeometry, wp *WorkPlane) {
				setSlot(t, wp, 2, f.pC.Key())
			},
			fresh: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByThreePoints(f.pA.Key(), f.pB.Key(), f.pC.Key())
			},
		},
		{
			kind: "plane-point",
			create: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByPlaneAndPoint(OriginXYPlane, f.pA.Key())
			},
			perturb: func(t *testing.T, g *WorkGeometry, wp *WorkPlane) {
				setSlot(t, wp, 0, OriginXZPlane)
				setSlot(t, wp, 1, f.pC.Key())
			},
			fresh: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByPlaneAndPoint(OriginXZPlane, f.pC.Key())
			},
		},
		{
			kind: "two-planes",
			create: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByTwoPlanes(OriginXYPlane, OriginXZPlane)
			},
			perturb: func(t *testing.T, g *WorkGeometry, wp *WorkPlane) {
				setSlot(t, wp, 1, OriginYZPlane)
			},
			fresh: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByTwoPlanes(OriginXYPlane, OriginYZPlane)
			},
		},
		{
			kind: "line-plane-angle",
			create: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByLinePlaneAndAngle(OriginXAxis, OriginXYPlane, func() float64 { return 0 })
			},
			perturb: func(t *testing.T, g *WorkGeometry, wp *WorkPlane) {
				setSlot(t, wp, 0, OriginYAxis)
				wp.EditableScalars()[0].Set(stdmath.Pi / 4)
			},
			fresh: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByLinePlaneAndAngle(OriginYAxis, OriginXYPlane, func() float64 { return stdmath.Pi / 4 })
			},
		},
		{
			kind: "two-lines",
			create: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByTwoLines(OriginXAxis, OriginYAxis)
			},
			perturb: func(t *testing.T, g *WorkGeometry, wp *WorkPlane) {
				setSlot(t, wp, 1, OriginZAxis)
			},
			fresh: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByTwoLines(OriginXAxis, OriginZAxis)
			},
		},
		{
			kind: "normal-to-curve",
			create: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByNormalToCurve(OriginZAxis, f.pC.Key())
			},
			perturb: func(t *testing.T, g *WorkGeometry, wp *WorkPlane) {
				setSlot(t, wp, 0, OriginXAxis)
				setSlot(t, wp, 1, f.pA.Key())
			},
			fresh: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByNormalToCurve(OriginXAxis, f.pA.Key())
			},
		},
	}
}

// tangentKindCases covers the surface-tangent kinds (built on the fixture's cylinder face);
// the torus mid-plane gets its own two-face fixture in TestTorusMidPlaneRedefineClosure.
func tangentKindCases(f *closureFixture) []redefineClosureCase {
	return []redefineClosureCase{
		{
			kind: "point-tangent",
			create: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByPointAndTangent(f.pA.Key(), FaceRef(f.cylKey))
			},
			perturb: func(t *testing.T, g *WorkGeometry, wp *WorkPlane) {
				setSlot(t, wp, 0, f.pB.Key())
			},
			fresh: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByPointAndTangent(f.pB.Key(), FaceRef(f.cylKey))
			},
		},
		{
			kind: "plane-tangent",
			create: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByPlaneAndTangent(OriginYZPlane, FaceRef(f.cylKey))
			},
			perturb: func(t *testing.T, g *WorkGeometry, wp *WorkPlane) {
				setSlot(t, wp, 0, OriginXZPlane)
			},
			fresh: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByPlaneAndTangent(OriginXZPlane, FaceRef(f.cylKey))
			},
		},
		{
			kind: "line-tangent",
			create: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByLineAndTangent(f.axisZ4.Key(), FaceRef(f.cylKey))
			},
			perturb: func(t *testing.T, g *WorkGeometry, wp *WorkPlane) {
				setSlot(t, wp, 0, f.axisZ5.Key())
			},
			fresh: func(t *testing.T, g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByLineAndTangent(f.axisZ5.Key(), FaceRef(f.cylKey))
			},
		},
	}
}

func TestWorkPlaneRedefineClosureOverEveryKind(t *testing.T) {
	t.Parallel()
	f := newClosureFixture(t)
	for _, tc := range redefineClosureCases(f) {
		t.Run(tc.kind, func(t *testing.T) {
			wp := tc.create(t, f.g)
			f.g.Recompute(f.bodies)
			if wp.Kind() != tc.kind {
				t.Fatalf("created plane Kind() = %q, want %q", wp.Kind(), tc.kind)
			}
			tc.perturb(t, f.g, wp)
			f.g.Recompute(f.bodies)
			want := tc.fresh(t, f.g)
			f.g.Recompute(f.bodies)
			assertSamePlane(t, wp, want)
		})
	}
}

// assertSamePlane compares origin and normal only: kinds that fix just a normal (bisector,
// angle, normal-to-curve, tangent) choose an arbitrary — but deterministic — in-plane X axis,
// so origin+normal is the full geometric identity of the datum.
func assertSamePlane(t *testing.T, got, want *WorkPlane) {
	t.Helper()
	if !got.Health().OK() || !want.Health().OK() {
		t.Fatalf("plane health: redefined=%+v fresh=%+v", got.Health(), want.Health())
	}
	if !got.Plane().Origin().IsEqualTo(want.Plane().Origin(), wtol) {
		t.Errorf("redefined origin = %v, fresh create = %v", got.Plane().Origin(), want.Plane().Origin())
	}
	if !got.Plane().Normal().AsVector().IsEqualTo(want.Plane().Normal().AsVector(), wtol) {
		t.Errorf("redefined normal = %v, fresh create = %v", got.Plane().Normal(), want.Plane().Normal())
	}
}

// TestTorusMidPlaneRedefineClosure re-picks the torus face — the mid-plane's only slot — to a
// second torus and checks the plane re-derives from it (the tangent-family redefine pinned
// against pre-I11 behavior).
func TestTorusMidPlaneRedefineClosure(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	torA, err := geom.NewTorus(math.P3(0, 0, 5), math.V3(0, 0, 1), 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	torB, err := geom.NewTorus(math.P3(0, 0, -3), math.V3(0, 0, 1), 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	bodyA, keyA := faceBody(t, torA)
	bodyB, keyB := faceBodyAt(t, torB, 1)
	bodies := []*topo.Body{bodyA, bodyB}
	g.Recompute(bodies)
	wp := g.WorkPlanes().AddByTorusMidPlane(FaceRef(keyA))
	g.Recompute(bodies)
	if !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 5), wtol) {
		t.Fatalf("initial torus mid-plane origin = %v, want (0,0,5)", wp.Plane().Origin())
	}
	setSlot(t, wp, 0, FaceRef(keyB))
	g.Recompute(bodies)
	want := g.WorkPlanes().AddByTorusMidPlane(FaceRef(keyB))
	g.Recompute(bodies)
	assertSamePlane(t, wp, want)
}

// faceBodyAt is faceBody with a distinct lineage index, so two single-face bodies in one test
// carry different reference keys (faceBody always mints index 0).
func faceBodyAt(t *testing.T, surface geom.Surface, idx int) (*topo.Body, []byte) {
	t.Helper()
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("f", "body", idx)))
	f := bld.AddFace(surface, topo.NewLineage(topo.Tok("f", "face", idx)))
	return bld.Build(), f.ReferenceKey()
}

// TestNonRedefinableKindsDeclareEmptyStory pins the deliberate non-editables: the grounded
// origin plane and the point-cloud-fit plane report no slots/scalars and a no-op snapshot —
// explicitly (their definitions implement the redefine methods), not via a default case.
func TestNonRedefinableKindsDeclareEmptyStory(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	origin := g.WorkPlanes().Item(0)
	if origin.IsRedefinable() {
		t.Errorf("origin plane %q is redefinable, want grounded/non-editable", origin.Name())
	}
	origin.SnapshotDefinition()() // must be a callable no-op

	cloud := g.WorkPlanes().AddByPointCloudFit(staticPlaneFit{})
	g.Recompute(nil)
	if cloud.IsRedefinable() {
		t.Errorf("point-cloud-fit plane is redefinable, want fit-driven/non-editable (#645)")
	}
	cloud.SnapshotDefinition()()
}

// staticPlaneFit is a fixed-frame PlaneFitSource stand-in for the cloud-fit redefine check.
type staticPlaneFit struct{}

func (staticPlaneFit) SourceID() string { return "cloud-1" }
func (staticPlaneFit) FitFrame() (math.Point3, math.UnitVector3, math.UnitVector3, bool) {
	x, _ := math.NewUnitVector3(1, 0, 0)
	y, _ := math.NewUnitVector3(0, 1, 0)
	return math.P3(0, 0, 1), x, y, true
}
