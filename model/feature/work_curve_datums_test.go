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
	near, _ := nearestHit(hits, new(math.P3(0, 10, 0)))
	if float64(near.Y) < 0 {
		t.Errorf("proximity toward +Y picked %v, want the +Y contact", near)
	}
	far, _ := nearestHit(hits, new(math.P3(0, -10, 0)))
	if float64(far.Y) > 0 {
		t.Errorf("proximity toward -Y picked %v, want the -Y contact", far)
	}
}

// TestCurveEntityPointRoundTrips: a curve-and-entity point serializes its refs and its proximity
// solution point, and restores to an equivalent definition (#1842).
func TestCurveEntityPointRoundTrips(t *testing.T) {
	def := curveEntityPointDef{curve: WorkRef("edge/x"), entity: OriginYZPlane, proximity: new(math.P3(1, 2, 3))}
	d, err := serializePointDef(def)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	if d.Kind != "curve-and-entity" || len(d.Refs) != 2 || len(d.Solution) != 3 {
		t.Fatalf("serialized = %+v, want kind curve-and-entity, 2 refs, a 3-vec solution", d)
	}
	restored := NewWorkGeometry()
	if err := restorePointFeature(restored.WorkPoints(), d); err != nil {
		t.Fatalf("restore: %v", err)
	}
	back, err := serializePointDef(restored.WorkPoints().Item(restored.WorkPoints().Count() - 1).def)
	if err != nil {
		t.Fatalf("re-serialize: %v", err)
	}
	if back.Kind != d.Kind || len(back.Solution) != 3 || back.Solution[0] != 1 || back.Solution[2] != 3 {
		t.Errorf("round-trip lost the proximity: %+v", back)
	}
}

// TestCentroidPointRoundTrips: a centroid point persists its edge references (any count) and restores.
func TestCentroidPointRoundTrips(t *testing.T) {
	def := centroidPointDef{edges: []WorkRef{WorkRef("edge/a"), WorkRef("edge/b"), WorkRef("edge/c")}}
	d, err := serializePointDef(def)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	if d.Kind != "centroid" || len(d.Refs) != 3 {
		t.Fatalf("serialized = %+v, want kind centroid with 3 refs", d)
	}
	restored := NewWorkGeometry()
	if err := restorePointFeature(restored.WorkPoints(), d); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := restored.WorkPoints().Item(restored.WorkPoints().Count() - 1).def.refs(); len(got) != 3 {
		t.Errorf("restored centroid refs = %v, want 3", got)
	}
}

// TestCirclePlaneNonIntersections covers the empty cases of circlePlanePoints: a plane parallel to
// the circle's plane (no crossing) and a plane that clears the circle entirely (line misses).
func TestCirclePlaneNonIntersections(t *testing.T) {
	circle, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 1) // unit circle in z=0
	if err != nil {
		t.Fatal(err)
	}
	parallel, _ := sketch.NewPlane(math.P3(0, 0, 5), mustUnit(1, 0, 0), mustUnit(0, 1, 0)) // z=5, parallel
	if hits := circlePlanePoints(circle, parallel); hits != nil {
		t.Errorf("parallel plane should not intersect, got %v", hits)
	}
	clear, _ := sketch.NewPlane(math.P3(2, 0, 0), mustUnit(0, 1, 0), mustUnit(0, 0, 1)) // x=2, beyond radius 1
	if hits := circlePlanePoints(circle, clear); hits != nil {
		t.Errorf("a plane clearing the circle should not intersect, got %v", hits)
	}
	tangent, _ := sketch.NewPlane(math.P3(1, 0, 0), mustUnit(0, 1, 0), mustUnit(0, 0, 1)) // x=1, tangent
	if hits := circlePlanePoints(circle, tangent); len(hits) != 1 {
		t.Errorf("a tangent plane should touch at one point, got %v", hits)
	}
}

// TestCurveEntityAndCentroidUnhealthy: a curve parallel to its entity (a line lying in the plane)
// and a centroid whose only edge is unresolved both go unhealthy rather than producing garbage.
func TestCurveEntityAndCentroidUnhealthy(t *testing.T) {
	g := NewWorkGeometry()
	// A +X edge in the z=0 plane is parallel to the XY entity plane → no pierce.
	body, e := lineEdgeBody(t, math.P3(-2, 1, 0), math.P3(2, 1, 0))
	miss := g.WorkPoints().AddByCurveAndEntity(EdgeRef(e.ReferenceKey()), OriginXYPlane, nil)
	orphan := g.WorkPoints().AddAtCentroid(WorkRef("edge/does-not-exist"))
	g.Recompute([]*topo.Body{body})
	if miss.Health().OK() {
		t.Error("a curve parallel to its entity should be unhealthy")
	}
	if orphan.Health().OK() {
		t.Error("a centroid with no resolvable edge should be unhealthy")
	}
}
