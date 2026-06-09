// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"reflect"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// These cover the WorkPlanes constructors added for Inventor parity: the reference-model
// ones (built on planes/axes/points) and the surface-tangent ones (built on a B-rep
// face), plus the recipe round-trip for every new kind.

func TestAddFixedWorkPlane(t *testing.T) {
	g := NewWorkGeometry()
	wp := g.WorkPlanes().AddFixed(func() math.Point3 { return math.P3(1, 2, 3) }, mustX(), mustY())
	if !wp.Health().OK() {
		t.Fatalf("fixed plane sick: %+v", wp.Health())
	}
	if !wp.Plane().Origin().IsEqualTo(math.P3(1, 2, 3), wtol) ||
		!wp.Plane().Normal().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("fixed plane origin=%v normal=%v, want (1,2,3)/+Z", wp.Plane().Origin(), wp.Plane().Normal())
	}
}

func TestPlaneAndPointWorkPlane(t *testing.T) {
	g := NewWorkGeometry()
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(5, 6, 7) })
	wp := g.WorkPlanes().AddByPlaneAndPoint(OriginXYPlane, pt.Key())
	if !wp.Plane().Origin().IsEqualTo(math.P3(5, 6, 7), wtol) ||
		!wp.Plane().Normal().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("plane-point origin=%v normal=%v, want (5,6,7)/+Z", wp.Plane().Origin(), wp.Plane().Normal())
	}
}

func TestTwoPlanesBisector(t *testing.T) {
	g := NewWorkGeometry()
	// XY (+Z) bisected with XZ (−Y) → normal halfway between, on the X-axis intersection.
	wp := g.WorkPlanes().AddByTwoPlanes(OriginXYPlane, OriginXZPlane)
	if !wp.Health().OK() {
		t.Fatalf("bisector sick: %+v", wp.Health())
	}
	if !wp.Plane().Normal().AsVector().IsParallelTo(math.V3(0, -1, 1), wtol) {
		t.Errorf("bisector normal=%v, want parallel to (0,-1,1)", wp.Plane().Normal())
	}
	// Parallel planes bisect at the mid-plane: XY and XY+10 → z=5.
	off := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 10 })
	mid := g.WorkPlanes().AddByTwoPlanes(OriginXYPlane, off.Key())
	if !mid.Plane().Origin().IsEqualTo(math.P3(0, 0, 5), wtol) {
		t.Errorf("mid-plane origin=%v, want z=5", mid.Plane().Origin())
	}
}

func TestLinePlaneAndAngle(t *testing.T) {
	g := NewWorkGeometry()
	// XY plane swung 90° about the X axis: normal +Z → ±Y, still holding the X axis.
	wp := g.WorkPlanes().AddByLinePlaneAndAngle(OriginXAxis, OriginXYPlane, func() float64 { return stdmath.Pi / 2 })
	if !wp.Health().OK() {
		t.Fatalf("line-plane-angle sick: %+v", wp.Health())
	}
	if !wp.Plane().Normal().AsVector().IsParallelTo(math.V3(0, 1, 0), wtol) {
		t.Errorf("swung normal=%v, want parallel to Y", wp.Plane().Normal())
	}
	if d := wp.Plane().Normal().AsVector().Dot(math.V3(1, 0, 0)); !math.IsNearZero(d, wtol) {
		t.Errorf("swung plane does not hold the X axis (normal·X=%g)", d)
	}
}

func TestTwoLinesWorkPlane(t *testing.T) {
	g := NewWorkGeometry()
	wp := g.WorkPlanes().AddByTwoLines(OriginXAxis, OriginYAxis)
	if !wp.Plane().Normal().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("two-lines normal=%v, want +Z", wp.Plane().Normal())
	}
	// Parallel lines have no plane → sick.
	off := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 1 })
	axB := g.WorkAxes().AddByPlaneIntersection(OriginXYPlane, OriginXZPlane) // along X
	bad := g.WorkPlanes().AddByTwoLines(OriginXAxis, axB.Key())
	_ = off
	if bad.Health().OK() {
		t.Error("two parallel lines should give a sick plane")
	}
}

func TestNormalToCurveWorkPlane(t *testing.T) {
	g := NewWorkGeometry()
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 4) })
	wp := g.WorkPlanes().AddByNormalToCurve(OriginZAxis, pt.Key())
	if !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 4), wtol) ||
		!wp.Plane().Normal().AsVector().IsParallelTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("normal-to-curve origin=%v normal=%v, want (0,0,4)/+Z", wp.Plane().Origin(), wp.Plane().Normal())
	}
}

func TestPointAndTangentWorkPlane(t *testing.T) {
	g := NewWorkGeometry()
	body, key := faceBody(t, mustCylinder(t)) // axis +Z, radius 2 at origin
	g.Recompute([]*topo.Body{body})
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(2, 0, 0) }) // on the surface
	wp := g.WorkPlanes().AddByPointAndTangent(pt.Key(), FaceRef(key))
	g.Recompute([]*topo.Body{body})
	if !wp.Health().OK() {
		t.Fatalf("point-tangent sick: %+v", wp.Health())
	}
	if !wp.Plane().Origin().IsEqualTo(math.P3(2, 0, 0), wtol) ||
		!wp.Plane().Normal().AsVector().IsParallelTo(math.V3(1, 0, 0), wtol) {
		t.Errorf("point-tangent origin=%v normal=%v, want (2,0,0)/+X", wp.Plane().Origin(), wp.Plane().Normal())
	}
}

func TestPlaneAndTangentWorkPlane(t *testing.T) {
	g := NewWorkGeometry()
	body, key := faceBody(t, mustCylinder(t))
	g.Recompute([]*topo.Body{body})
	// YZ plane normal is +X (⊥ the cylinder axis), so the parallel tangent sits at x=2.
	wp := g.WorkPlanes().AddByPlaneAndTangent(OriginYZPlane, FaceRef(key))
	g.Recompute([]*topo.Body{body})
	if !wp.Health().OK() {
		t.Fatalf("plane-tangent sick: %+v", wp.Health())
	}
	if !wp.Plane().Origin().IsEqualTo(math.P3(2, 0, 0), wtol) {
		t.Errorf("plane-tangent origin=%v, want (2,0,0)", wp.Plane().Origin())
	}
	// A reference plane not perpendicular to the axis has no parallel tangent → sick.
	bad := g.WorkPlanes().AddByPlaneAndTangent(OriginXYPlane, FaceRef(key))
	g.Recompute([]*topo.Body{body})
	if bad.Health().OK() {
		t.Error("XY plane (parallel to axis) should give a sick tangent plane")
	}
}

func TestLineAndTangentWorkPlane(t *testing.T) {
	g := NewWorkGeometry()
	body, key := faceBody(t, mustCylinder(t)) // axis +Z, radius 2
	g.Recompute([]*topo.Body{body})
	a := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(4, 0, 0) })
	b := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(4, 0, 1) })
	line := g.WorkAxes().AddByTwoPoints(a.Key(), b.Key()) // parallel to +Z, offset 4
	wp := g.WorkPlanes().AddByLineAndTangent(line.Key(), FaceRef(key))
	g.Recompute([]*topo.Body{body})
	if !wp.Health().OK() {
		t.Fatalf("line-tangent sick: %+v", wp.Health())
	}
	n := wp.Plane().Normal().AsVector()
	if !math.IsNearZero(n.Dot(math.V3(0, 0, 1)), wtol) {
		t.Errorf("line-tangent plane should hold the +Z line (normal·Z=%g)", n.Dot(math.V3(0, 0, 1)))
	}
	// Tangency: the cylinder axis (origin) is exactly radius 2 from the plane.
	dist := wp.Plane().Origin().VectorTo(math.P3(0, 0, 0)).Dot(n)
	if !math.IsNearZero(dist*dist-4, 1e-6) {
		t.Errorf("axis-to-plane distance %g, want radius 2", dist)
	}
}

func TestTorusMidPlaneWorkPlane(t *testing.T) {
	g := NewWorkGeometry()
	tor, err := geom.NewTorus(math.P3(0, 0, 5), math.V3(0, 0, 1), 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	body, key := faceBody(t, tor)
	g.Recompute([]*topo.Body{body})
	wp := g.WorkPlanes().AddByTorusMidPlane(FaceRef(key))
	g.Recompute([]*topo.Body{body})
	if !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 5), wtol) ||
		!wp.Plane().Normal().AsVector().IsParallelTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("torus mid-plane origin=%v normal=%v, want (0,0,5)/+Z", wp.Plane().Origin(), wp.Plane().Normal())
	}
}

func TestThreePointPlaneFromModelVertices(t *testing.T) {
	g := NewWorkGeometry()
	body, keys := triangleBody(t) // three vertices in the z=0 plane
	g.Recompute([]*topo.Body{body})
	wp := g.WorkPlanes().AddByThreePoints(VertexRef(keys[0]), VertexRef(keys[1]), VertexRef(keys[2]))
	g.Recompute([]*topo.Body{body})
	if !wp.Health().OK() {
		t.Fatalf("three-point plane from vertices sick: %+v", wp.Health())
	}
	if !wp.Plane().Normal().AsVector().IsParallelTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("normal = %v, want +Z", wp.Plane().Normal())
	}
	// When the vertices are gone (no body) the datum goes sick, not garbage.
	g.Recompute(nil)
	if wp.Health().OK() {
		t.Error("three-point plane should go sick when its vertices are lost")
	}
}

func TestTangentWorkPlaneSickWhenFaceLost(t *testing.T) {
	g := NewWorkGeometry()
	// A face reference that never binds (no body) must go sick, not panic or guess.
	wp := g.WorkPlanes().AddByTorusMidPlane(FaceRef([]byte("ghost")))
	g.Recompute(nil)
	if wp.Health().OK() {
		t.Error("tangent plane with an unbound face should be sick")
	}
}

func TestWorkPlaneRecipeRoundTrip(t *testing.T) {
	g := seedAllWorkPlanes()
	data, err := MarshalWork(g)
	if err != nil {
		t.Fatalf("MarshalWork: %v", err)
	}
	restored := NewWorkGeometry()
	if err := ApplyWork(restored, data); err != nil {
		t.Fatalf("ApplyWork: %v", err)
	}
	again, err := MarshalWork(restored)
	if err != nil {
		t.Fatalf("re-MarshalWork: %v", err)
	}
	if !reflect.DeepEqual(data, again) {
		t.Errorf("recipe did not round-trip:\n first=%+v\n again=%+v", data, again)
	}
}

// seedAllWorkPlanes builds a geometry exercising every new plane kind, so the round-trip
// test covers each codec. Face refs are synthetic (the structural recipe round-trips
// without a body; tangent geometry is covered by the body-backed tests above).
func seedAllWorkPlanes() *WorkGeometry {
	g := NewWorkGeometry()
	p1 := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 0, 0) })
	p2 := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 1, 0) })
	p3 := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 1) })
	ax := g.WorkAxes().AddByTwoPoints(OriginCenter, p3.Key())
	fk := FaceRef([]byte("face-key"))
	g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 2 })
	g.WorkPlanes().AddByThreePoints(p1.Key(), p2.Key(), p3.Key())
	g.WorkPlanes().AddFixed(func() math.Point3 { return math.P3(1, 2, 3) }, mustX(), mustY())
	g.WorkPlanes().AddByPlaneAndPoint(OriginXYPlane, p1.Key())
	g.WorkPlanes().AddByTwoPlanes(OriginXYPlane, OriginXZPlane)
	g.WorkPlanes().AddByLinePlaneAndAngle(OriginXAxis, OriginXYPlane, func() float64 { return 0.5 })
	g.WorkPlanes().AddByTwoLines(OriginXAxis, OriginYAxis)
	g.WorkPlanes().AddByNormalToCurve(ax.Key(), p3.Key())
	g.WorkPlanes().AddByTorusMidPlane(fk)
	g.WorkPlanes().AddByPointAndTangent(p1.Key(), fk)
	g.WorkPlanes().AddByPlaneAndTangent(OriginXYPlane, fk)
	g.WorkPlanes().AddByLineAndTangent(ax.Key(), fk)
	return g
}

// faceBody wraps a surface in a one-face body and returns the face's reference key, so
// a surface-tangent work plane can resolve it through the work geometry.
func faceBody(t *testing.T, surface geom.Surface) (*topo.Body, []byte) {
	t.Helper()
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("f", "body", 0)))
	f := bld.AddFace(surface, topo.NewLineage(topo.Tok("f", "face", 0)))
	return bld.Build(), f.ReferenceKey()
}

func mustCylinder(t *testing.T) geom.Cylinder {
	t.Helper()
	c, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// triangleBody builds a one-face triangular body and returns the reference keys of its
// three corner vertices, so a three-point work plane can be built on model vertices.
func triangleBody(t *testing.T) (*topo.Body, [3][]byte) {
	t.Helper()
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("f", "body", 0)))
	a := bld.AddVertex(math.P3(0, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 0)))
	b := bld.AddVertex(math.P3(2, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 1)))
	c := bld.AddVertex(math.P3(0, 2, 0), topo.NewLineage(topo.Tok("f", "vertex", 2)))
	seg := func(p, q *topo.Vertex) geom.LineSegment { return geom.NewLineSegment(p.Point(), q.Point()) }
	ab := bld.AddEdge(seg(a, b), a, b, topo.NewLineage(topo.Tok("f", "edge", 0)))
	bc := bld.AddEdge(seg(b, c), b, c, topo.NewLineage(topo.Tok("f", "edge", 1)))
	ca := bld.AddEdge(seg(c, a), c, a, topo.NewLineage(topo.Tok("f", "edge", 2)))
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	bld.AddFace(pl, topo.NewLineage(topo.Tok("f", "face", 0)), topo.OuterLoop(topo.Fwd(ab), topo.Fwd(bc), topo.Fwd(ca)))
	return bld.Build(), [3][]byte{a.ReferenceKey(), b.ReferenceKey(), c.ReferenceKey()}
}

// TestOffsetPlaneFromFace offsets a work plane from a planar B-rep face (FaceRef as the
// base of AddByPlaneAndOffset): a face at z=3 (+Z) offset by 2 lands at z=5. Regression for
// plane references not resolving a planar face (the offset-from-face work-plane bug).
func TestOffsetPlaneFromFace(t *testing.T) {
	g := NewWorkGeometry()
	pl, err := geom.NewPlane(math.P3(0, 0, 3), math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	body, key := faceBody(t, pl)
	g.Recompute([]*topo.Body{body})
	wp := g.WorkPlanes().AddByPlaneAndOffset(FaceRef(key), func() float64 { return 2 })
	g.Recompute([]*topo.Body{body})
	if !wp.Health().OK() {
		t.Fatalf("offset-from-face sick: %+v", wp.Health())
	}
	if !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 5), wtol) {
		t.Errorf("offset-from-face origin = %v, want (0,0,5)", wp.Plane().Origin())
	}
}
