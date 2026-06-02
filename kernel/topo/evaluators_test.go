// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/math"
)

const evalTol = 1e-6

func near(a, b float64) bool { return stdmath.Abs(a-b) < evalTol }

func TestCurveEvaluatorMatchesAnalyticCircle(t *testing.T) {
	circle, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	e := NewCurveEvaluator(circle)

	// Point and tangent match the analytic circle.
	if !e.PointAt(0).IsEqualTo(circle.PointAt(0), evalTol) {
		t.Error("PointAt mismatch")
	}
	tan := e.TangentAt(0)
	if !near(tan.Length(), 1) { // unit tangent
		t.Errorf("tangent not unit: |t|=%v", tan.Length())
	}
	if !near(tan.Dot(circle.TangentAt(0).Scale(1/circle.TangentAt(0).Length())), 1) {
		t.Error("tangent direction mismatch")
	}
	// Curvature of a radius-2 circle is 1/2.
	if k := e.CurvatureAt(0.7); !near(k, 0.5) {
		t.Errorf("curvature = %v, want 0.5", k)
	}
	// Length of the full circle is 2πr = 4π (integrated over the curve's own domain).
	lo, hi := circle.Domain()
	if l := e.Length(lo, hi); stdmath.Abs(l-4*stdmath.Pi) > 1e-4 {
		t.Errorf("length = %v, want %v", l, 4*stdmath.Pi)
	}
}

func TestEdgeEvaluatorLengthAndClosest(t *testing.T) {
	// A line segment from (0,0,0) to (3,4,0): length 5.
	seg := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(3, 4, 0))
	bld := NewBuilder(false, NewLineage(Tok("f", "body", 0)))
	v0 := bld.AddVertex(math.P3(0, 0, 0), NewLineage(Tok("f", "vertex", 0)))
	v1 := bld.AddVertex(math.P3(3, 4, 0), NewLineage(Tok("f", "vertex", 1)))
	edge := bld.AddEdge(seg, v0, v1, NewLineage(Tok("f", "edge", 0)))
	ev := NewEdgeEvaluator(edge)
	if l := ev.Length(); !near(l, 5) {
		t.Errorf("segment length = %v, want 5", l)
	}
	// Closest point on the segment to (0,5,0): foot of perpendicular at
	// t = (0,5,0)·(3,4,0)/25 = 0.8 → (2.4, 3.2, 0).
	got := ev.ClosestPoint(math.P3(0, 5, 0))
	if got.DistanceTo(math.P3(2.4, 3.2, 0)) > 1e-2 {
		t.Errorf("closest point = %v, want near (2.4,3.2,0)", got)
	}
}

func TestSurfaceEvaluatorOnSphere(t *testing.T) {
	sphere, _ := geom.NewSphere(math.P3(0, 0, 0), 3)
	e := NewSurfaceEvaluator(sphere)
	u, v := 0.6, 1.0
	if !e.PointAt(u, v).IsEqualTo(sphere.PointAt(u, v), evalTol) {
		t.Error("surface PointAt mismatch")
	}
	if n := e.NormalAt(u, v); !near(n.Length(), 1) {
		t.Errorf("normal not unit: %v", n.Length())
	}
	// Closest point to a point well outside, along a radius, lands on the sphere.
	external := math.P3(10, 0, 0)
	cp := e.ClosestPoint(external)
	if !near(cp.DistanceTo(math.P3(0, 0, 0)), 3) {
		t.Errorf("closest point not on sphere: |cp|=%v, want 3", cp.DistanceTo(math.P3(0, 0, 0)))
	}
	if cp.DistanceTo(math.P3(3, 0, 0)) > 1e-2 {
		t.Errorf("closest point = %v, want near (3,0,0)", cp)
	}
}

func TestFaceEvaluatorAreaAndContainment(t *testing.T) {
	body := buildTetra()
	// Face 0 is the triangle (0,0,0)-(1,0,0)-(0,1,0): area 0.5.
	fe := NewFaceEvaluator(body.Faces()[0])
	area, exact := fe.Area()
	if !exact || !near(area, 0.5) {
		t.Errorf("triangle area = %v exact=%v, want 0.5/true", area, exact)
	}
	// A point inside the triangle is contained; one outside is not.
	if !fe.Contains(math.P3(0.25, 0.25, 0)) {
		t.Error("interior point not contained")
	}
	if fe.Contains(math.P3(1, 1, 0)) {
		t.Error("exterior point reported contained")
	}
	// A point off the plane is not contained.
	if fe.Contains(math.P3(0.25, 0.25, 1)) {
		t.Error("off-plane point reported contained")
	}
	// Normal matches the planar face's surface normal.
	if !fe.NormalAt(0, 0).IsEqualTo(math.V3(0, 0, 1), evalTol) {
		t.Errorf("face normal = %v, want (0,0,1)", fe.NormalAt(0, 0))
	}
}

func TestSurfaceTangentsAndVertexEdgeKeys(t *testing.T) {
	sphere, _ := geom.NewSphere(math.P3(0, 0, 0), 1)
	du, dv := NewSurfaceEvaluator(sphere).TangentsAt(0.5, 1.0)
	if du.Length() == 0 && dv.Length() == 0 {
		t.Error("sphere tangents both degenerate at a regular point")
	}
	body := buildTetra()
	if len(body.Vertices()[0].ReferenceKey()) == 0 || len(body.Edges()[0].Lineage().Tokens()) == 0 {
		t.Error("vertex reference key / edge lineage missing")
	}
}

func TestContainmentOnNonZNormalFace(t *testing.T) {
	// A face on the x=0 plane (normal +X) exercises a different dropAxis branch.
	bld := NewBuilder(false, NewLineage(Tok("f", "body", 0)))
	mk := func(p math.Point3, i int) *Vertex { return bld.AddVertex(p, NewLineage(Tok("f", "vertex", i))) }
	a, b, c := mk(math.P3(0, 0, 0), 0), mk(math.P3(0, 2, 0), 1), mk(math.P3(0, 0, 2), 2)
	seg := func(p, q *Vertex, i int) *Edge {
		return bld.AddEdge(geom.NewLineSegment(p.Point(), q.Point()), p, q, NewLineage(Tok("f", "edge", i)))
	}
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(1, 0, 0))
	f := bld.AddFace(plane, NewLineage(Tok("f", "face", 0)),
		OuterLoop(Fwd(seg(a, b, 0)), Fwd(seg(b, c, 1)), Rev(seg(a, c, 2))))
	fe := NewFaceEvaluator(f)
	if !fe.Contains(math.P3(0, 0.5, 0.5)) {
		t.Error("interior point on x=0 face not contained")
	}
	if fe.Contains(math.P3(0, 2, 2)) {
		t.Error("exterior point reported contained")
	}
}

func TestNonPlanarAreaNotExact(t *testing.T) {
	// A face on a cylinder reports inexact area (needs tessellation).
	cyl, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	bld := NewBuilder(false, NewLineage(Tok("f", "body", 0)))
	v := bld.AddVertex(math.P3(1, 0, 0), NewLineage(Tok("f", "vertex", 0)))
	e := bld.AddEdge(geom.NewLineSegment(math.P3(1, 0, 0), math.P3(1, 0, 1)), v, v, NewLineage(Tok("f", "edge", 0)))
	f := bld.AddFace(cyl, NewLineage(Tok("f", "face", 0)), OuterLoop(Fwd(e)))
	if _, exact := NewFaceEvaluator(f).Area(); exact {
		t.Error("cylindrical face area should not be exact yet")
	}
}
