// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// lineEdgeBody builds a one-face triangular body and returns it plus its first edge (start→end),
// so a work axis/point can be built on a real B-rep edge with a reference key.
func lineEdgeBody(t *testing.T, start, end math.Point3) (*topo.Body, *topo.Edge) {
	t.Helper()
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("f", "body", 0)))
	a := bld.AddVertex(start, topo.NewLineage(topo.Tok("f", "vertex", 0)))
	b := bld.AddVertex(end, topo.NewLineage(topo.Tok("f", "vertex", 1)))
	c := bld.AddVertex(math.P3(0, 3, 0), topo.NewLineage(topo.Tok("f", "vertex", 2)))
	seg := func(p, q *topo.Vertex) geom.LineSegment { return geom.NewLineSegment(p.Point(), q.Point()) }
	ab := bld.AddEdge(seg(a, b), a, b, topo.NewLineage(topo.Tok("f", "edge", 0)))
	bc := bld.AddEdge(seg(b, c), b, c, topo.NewLineage(topo.Tok("f", "edge", 1)))
	ca := bld.AddEdge(seg(c, a), c, a, topo.NewLineage(topo.Tok("f", "edge", 2)))
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	bld.AddFace(pl, topo.NewLineage(topo.Tok("f", "face", 0)), topo.OuterLoop(topo.Fwd(ab), topo.Fwd(bc), topo.Fwd(ca)))
	return bld.Build(), ab
}

// TestAnalyticEdgeAxisFromLineageKey: an axis built on a lineage-key edge reference lies along the
// edge (#1840).
func TestAnalyticEdgeAxisFromLineageKey(t *testing.T) {
	g := NewWorkGeometry()
	body, e := lineEdgeBody(t, math.P3(0, 0, 0), math.P3(4, 0, 0)) // edge along +X
	wa := g.WorkAxes().AddByAnalyticEdge(EdgeRef(e.ReferenceKey()))
	g.Recompute([]*topo.Body{body})
	if !wa.Health().OK() {
		t.Fatalf("analytic-edge axis sick: %+v", wa.Health())
	}
	if !wa.Direction().AsVector().IsParallelTo(math.V3(1, 0, 0), wtol) {
		t.Errorf("axis direction = %v, want +X", wa.Direction())
	}
	if wa.Kind() != "analytic-edge" {
		t.Errorf("kind = %q, want analytic-edge", wa.Kind())
	}
}

// TestLineByEntityAxisKind: the line-by-entity constructor shares the geometry but keeps its own
// kind name (#1840).
func TestLineByEntityAxisKind(t *testing.T) {
	g := NewWorkGeometry()
	body, e := lineEdgeBody(t, math.P3(0, 0, 0), math.P3(0, 4, 0)) // edge along +Y
	wa := g.WorkAxes().AddByLineByEntity(EdgeRef(e.ReferenceKey()))
	g.Recompute([]*topo.Body{body})
	if !wa.Health().OK() || wa.Kind() != "line-by-entity" {
		t.Fatalf("line-by-entity axis: kind %q healthy %v", wa.Kind(), wa.Health().OK())
	}
	if !wa.Direction().AsVector().IsParallelTo(math.V3(0, 1, 0), wtol) {
		t.Errorf("axis direction = %v, want +Y", wa.Direction())
	}
}

// TestAnalyticEdgeAxisFromGeometricRef: the ADR-0040 geometric edge reference (midpoint+direction)
// binds the matching edge on the running body — the external-author path (#1840).
func TestAnalyticEdgeAxisFromGeometricRef(t *testing.T) {
	g := NewWorkGeometry()
	body, _ := lineEdgeBody(t, math.P3(0, 0, 0), math.P3(4, 0, 0)) // edge along +X, midpoint (2,0,0)
	ref := types.GeometricEdgeRef{Midpoint: [3]float64{2, 0, 0}, Direction: [3]float64{1, 0, 0}}.Ref()
	wa := g.WorkAxes().AddByAnalyticEdge(WorkRef(ref))
	g.Recompute([]*topo.Body{body})
	if !wa.Health().OK() {
		t.Fatalf("geometric-ref axis sick: %+v", wa.Health())
	}
	if !wa.Direction().AsVector().IsParallelTo(math.V3(1, 0, 0), wtol) {
		t.Errorf("axis direction = %v, want +X", wa.Direction())
	}
}

// TestGeometricEdgeRefMissesHonestly: a descriptor far from any edge binds nothing, so the datum
// goes sick rather than binding the wrong edge (#1840).
func TestGeometricEdgeRefMissesHonestly(t *testing.T) {
	g := NewWorkGeometry()
	body, _ := lineEdgeBody(t, math.P3(0, 0, 0), math.P3(4, 0, 0))
	ref := types.GeometricEdgeRef{Midpoint: [3]float64{100, 100, 100}, Direction: [3]float64{1, 0, 0}}.Ref()
	wa := g.WorkAxes().AddByAnalyticEdge(WorkRef(ref))
	g.Recompute([]*topo.Body{body})
	if wa.Health().OK() {
		t.Error("a far-away geometric edge reference should bind nothing (sick), not the wrong edge")
	}
}

// TestMidpointOfEdge: the edge-midpoint point sits at the edge midpoint (#1842).
func TestMidpointOfEdge(t *testing.T) {
	g := NewWorkGeometry()
	body, e := lineEdgeBody(t, math.P3(0, 0, 0), math.P3(4, 0, 0)) // midpoint (2,0,0)
	wp := g.WorkPoints().AddByMidpointOfEdge(EdgeRef(e.ReferenceKey()))
	g.Recompute([]*topo.Body{body})
	if !wp.Health().OK() {
		t.Fatalf("edge-midpoint sick: %+v", wp.Health())
	}
	if !wp.Point().IsEqualTo(math.P3(2, 0, 0), wtol) {
		t.Errorf("edge midpoint = %v, want (2,0,0)", wp.Point())
	}
	if wp.Kind() != "edge-midpoint" {
		t.Errorf("kind = %q, want edge-midpoint", wp.Kind())
	}
}

// TestEdgeDatumsRoundTrip: an analytic-edge axis and an edge-midpoint point restore from the recipe
// and re-bind against the body (#1840, #1842).
func TestEdgeDatumsRoundTrip(t *testing.T) {
	g := NewWorkGeometry()
	body, e := lineEdgeBody(t, math.P3(0, 0, 0), math.P3(4, 0, 0))
	g.WorkAxes().AddByAnalyticEdge(EdgeRef(e.ReferenceKey()))
	g.WorkPoints().AddByMidpointOfEdge(EdgeRef(e.ReferenceKey()))
	g.Recompute([]*topo.Body{body})

	data, err := MarshalWork(g)
	if err != nil {
		t.Fatalf("MarshalWork: %v", err)
	}
	restored := NewWorkGeometry()
	if err := ApplyWork(restored, data); err != nil {
		t.Fatalf("ApplyWork: %v", err)
	}
	restored.Recompute([]*topo.Body{body})
	ra := restored.WorkAxes().Item(restored.WorkAxes().Count() - 1)
	rp := restored.WorkPoints().Item(restored.WorkPoints().Count() - 1)
	if ra.Kind() != "analytic-edge" || !ra.Health().OK() {
		t.Errorf("restored axis kind %q healthy %v", ra.Kind(), ra.Health().OK())
	}
	if rp.Kind() != "edge-midpoint" || !rp.Point().IsEqualTo(math.P3(2, 0, 0), wtol) {
		t.Errorf("restored point kind %q at %v, want edge-midpoint at (2,0,0)", rp.Kind(), rp.Point())
	}
}

// TestEdgeRefErrors: a non-edge ref and an edge ref with no body both fail to resolve (#1840).
func TestEdgeRefErrors(t *testing.T) {
	g := NewWorkGeometry()
	if _, err := g.edge(WorkRef("plane/3")); err == nil {
		t.Error("a non-edge ref should not resolve as an edge")
	}
	if _, err := g.edge(EdgeRef([]byte("k"))); err == nil {
		t.Error("with no body an edge ref should not resolve")
	}
}
