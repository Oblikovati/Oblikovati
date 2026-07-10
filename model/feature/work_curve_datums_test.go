// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// twoEdgeBody builds an L of two straight edges (p0→p1→p2) as a closed triangular face and returns
// the body plus the two edges, so a centroid work point can reference real B-rep edges by key.
func twoEdgeBody(t *testing.T, p0, p1, p2 math.Point3) (*topo.Body, *topo.Edge, *topo.Edge) {
	t.Helper()
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("f", "body", 0)))
	a := bld.AddVertex(p0, topo.NewLineage(topo.Tok("f", "vertex", 0)))
	b := bld.AddVertex(p1, topo.NewLineage(topo.Tok("f", "vertex", 1)))
	c := bld.AddVertex(p2, topo.NewLineage(topo.Tok("f", "vertex", 2)))
	seg := func(p, q *topo.Vertex) geom.LineSegment { return geom.NewLineSegment(p.Point(), q.Point()) }
	ab := bld.AddEdge(seg(a, b), a, b, topo.NewLineage(topo.Tok("f", "edge", 0)))
	bc := bld.AddEdge(seg(b, c), b, c, topo.NewLineage(topo.Tok("f", "edge", 1)))
	ca := bld.AddEdge(seg(c, a), c, a, topo.NewLineage(topo.Tok("f", "edge", 2)))
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	bld.AddFace(pl, topo.NewLineage(topo.Tok("f", "face", 0)), topo.OuterLoop(topo.Fwd(ab), topo.Fwd(bc), topo.Fwd(ca)))
	return bld.Build(), ab, bc
}

// TestCurveEntityPointLineHitsPlane: a straight edge along +X at y=2 pierces the YZ origin plane
// (x=0) at (0,2,0) — the elementary line∩plane case (proximity irrelevant, one solution).
func TestCurveEntityPointLineHitsPlane(t *testing.T) {
	g := NewWorkGeometry()
	body, e := lineEdgeBody(t, math.P3(-2, 2, 0), math.P3(2, 2, 0))
	wp := g.WorkPoints().AddByCurveAndEntity(EdgeRef(e.ReferenceKey()), OriginYZPlane, nil)
	g.Recompute([]*topo.Body{body})
	if !wp.Health().OK() {
		t.Fatalf("curve-and-entity point sick: %+v", wp.Health())
	}
	if got := wp.Point(); !got.IsEqualTo(math.P3(0, 2, 0), 1e-9) {
		t.Errorf("intersection = %v, want (0,2,0)", got)
	}
}

// TestCentroidLengthWeighted: the centroid of a length-4 edge (midpoint (2,0,0)) and a length-2 edge
// (midpoint (4,1,0)) is (4·(2,0,0)+2·(4,1,0))/6 = (16/6, 2/6, 0) — length-weighted, not a plain mean.
func TestCentroidLengthWeighted(t *testing.T) {
	g := NewWorkGeometry()
	body, e0, e1 := twoEdgeBody(t, math.P3(0, 0, 0), math.P3(4, 0, 0), math.P3(4, 2, 0))
	wp := g.WorkPoints().AddAtCentroid(EdgeRef(e0.ReferenceKey()), EdgeRef(e1.ReferenceKey()))
	g.Recompute([]*topo.Body{body})
	if !wp.Health().OK() {
		t.Fatalf("centroid point sick: %+v", wp.Health())
	}
	if got := wp.Point(); !got.IsEqualTo(math.P3(16.0/6.0, 2.0/6.0, 0), 1e-9) {
		t.Errorf("centroid = %v, want (2.6667, 0.3333, 0)", got)
	}
}

// TestCirclePlaneIntersectionGolden pins circlePlanePoints against the analytic (= OCCT
// IntAna_IntConicQuad) solution: a unit circle in the z=0 plane meets the plane x=0.5 at
// (0.5, ±√0.75, 0), √0.75 = 0.86602540378…
func TestCirclePlaneIntersectionGolden(t *testing.T) {
	circle, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	if err != nil {
		t.Fatal(err)
	}
	target, err := sketch.NewPlane(math.P3(0.5, 0, 0), mustUnit(0, 1, 0), mustUnit(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	hits := circlePlanePoints(circle, target)
	if len(hits) != 2 {
		t.Fatalf("expected 2 intersections, got %d: %v", len(hits), hits)
	}
	const want = 0.8660254037844386 // √0.75, the OCCT IntAna contact ordinate
	for _, h := range hits {
		if stdmath.Abs(float64(h.X)-0.5) > 1e-12 || stdmath.Abs(stdmath.Abs(float64(h.Y))-want) > 1e-12 {
			t.Errorf("intersection %v off the golden (0.5, ±%.16f, 0)", h, want)
		}
	}
	// proximity picks the near side deterministically
	near, _ := nearestHit(hits, ptr(math.P3(0, 10, 0)))
	if float64(near.Y) < 0 {
		t.Errorf("proximity toward +Y picked %v, want the +Y contact", near)
	}
	far, _ := nearestHit(hits, ptr(math.P3(0, -10, 0)))
	if float64(far.Y) > 0 {
		t.Errorf("proximity toward -Y picked %v, want the -Y contact", far)
	}
}

func ptr(p math.Point3) *math.Point3 { return &p }
