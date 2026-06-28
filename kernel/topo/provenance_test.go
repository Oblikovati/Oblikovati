// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestNameByParentsIsOrderIndependent pins the defining property (ADR-0043): the provenance name
// depends only on the SET of parents, not the order the generator discovered them in. Swapping the
// parents yields byte-identical keys; the parents' token runs are joined by the separator.
func TestNameByParentsIsOrderIndependent(t *testing.T) {
	a := NewLineage(Tok("Extrusion1", "side", 2))
	b := NewLineage(Tok("Extrusion1", "side", 0))
	sep := Tok("brep", "x", 0)

	ab := NameByParents([]Lineage{a, b}, sep, Tok("brep", "seg", 0), 0)
	ba := NameByParents([]Lineage{b, a}, sep, Tok("brep", "seg", 0), 0)
	if string(ab.Key()) != string(ba.Key()) {
		t.Fatalf("order changed the name: %q vs %q", ab.Key(), ba.Key())
	}
	// b sorts before a (…side#0 < …side#2), so the canonical form is b / brep:x#0 / a.
	if got, want := string(ab.Key()), "Extrusion1:side#0/brep:x#0/Extrusion1:side#2"; got != want {
		t.Errorf("name = %q, want %q", got, want)
	}
}

// TestNameByParentsRank appends the disambiguator only when rank > 0, carrying the rank seed's
// feature/role with the rank as its index.
func TestNameByParentsRank(t *testing.T) {
	a := NewLineage(Tok("f", "a", 0))
	b := NewLineage(Tok("f", "b", 0))
	sep, seed := Tok("brep", "x", 0), Tok("brep", "seg", 0)

	if got, want := string(NameByParents([]Lineage{a, b}, sep, seed, 0).Key()), "f:a#0/brep:x#0/f:b#0"; got != want {
		t.Errorf("rank 0 name = %q, want %q (no disambiguator)", got, want)
	}
	if got, want := string(NameByParents([]Lineage{a, b}, sep, seed, 3).Key()), "f:a#0/brep:x#0/f:b#0/brep:seg#3"; got != want {
		t.Errorf("rank 3 name = %q, want %q", got, want)
	}
}

// TestNameByParentsThreeParents generalizes past the boolean's two-parent case: a vertex where
// three faces meet is named by all three, canonically ordered and sep-joined.
func TestNameByParentsThreeParents(t *testing.T) {
	p := []Lineage{NewLineage(Tok("f", "c", 0)), NewLineage(Tok("f", "a", 0)), NewLineage(Tok("f", "b", 0))}
	got := string(NameByParents(p, Tok("f", "at", 0), Tok("f", "dup", 0), 0).Key())
	want := "f:a#0/f:at#0/f:b#0/f:at#0/f:c#0"
	if got != want {
		t.Errorf("three-parent name = %q, want %q", got, want)
	}
}

// TestRelineageByFaceProvenance pins the finalize pass: an edge shared by two provenanced faces is
// renamed by NameByParents over their provenance; a boundary edge of a provenanced face by that one
// parent; and an entity touching a face with NO provenance keeps its build-order name.
func TestRelineageByFaceProvenance(t *testing.T) {
	bld := NewBuilder(false, NewLineage(Tok("f", "body", 0)))
	mk := func(x, y float64, i int) *Vertex {
		return bld.AddVertex(math.P3(x, y, 0), NewLineage(Tok("f", "vertex", i)))
	}
	a, b, c, d := mk(0, 0, 0), mk(1, 0, 1), mk(0, 1, 2), mk(1, 1, 3)
	edge := func(p, q *Vertex, i int) *Edge {
		return bld.AddEdge(geom.NewLineSegment(p.Point(), q.Point()), p, q, NewLineage(Tok("f", "edge", i)))
	}
	ab, bc, ca := edge(a, b, 0), edge(b, c, 1), edge(c, a, 2)
	bd, dc := edge(b, d, 3), edge(d, c, 4)
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	f1 := bld.AddFace(pl, NewLineage(Tok("f", "face", 0)), OuterLoop(Fwd(ab), Fwd(bc), Fwd(ca)))
	f2 := bld.AddFace(pl, NewLineage(Tok("f", "face", 1)), OuterLoop(Fwd(bd), Fwd(dc), Rev(bc))) // shares bc
	body := bld.Build()

	provA := NewLineage(Tok("g", "a", 0))
	provB := NewLineage(Tok("g", "b", 0))
	body.RelineageByFaceProvenance(map[*Face]Lineage{f1: provA, f2: provB}, Tok("x", "x", 0), Tok("x", "seg", 0))

	// bc borders both provenanced faces, the only edge to do so ⇒ named by both, canonically ordered,
	// no rank (a unique parent set needs none).
	if got, want := string(bc.Lineage().Key()), "g:a#0/x:x#0/g:b#0"; got != want {
		t.Errorf("shared edge bc = %q, want %q", got, want)
	}
	// ab and ca BOTH border only f1 ⇒ they share the parent set {provA}; without disambiguation both
	// would collide on "g:a#0". The geometric rank (ordered by midpoint) keeps the keys distinct.
	abKey, caKey := string(ab.Lineage().Key()), string(ca.Lineage().Key())
	if abKey == caKey {
		t.Errorf("boundary edges ab and ca collide on %q — rank disambiguation missing", abKey)
	}
	for _, k := range []string{abKey, caKey} {
		if !strings.HasPrefix(k, "g:a#0/x:seg#") {
			t.Errorf("ranked boundary edge = %q, want prefix g:a#0/x:seg#", k)
		}
	}
}

// TestRelineageRanksBigonByCurveMidpoint pins the bigon case (ADR-0043 SSI-edge provenance): two
// edges that share BOTH the same face pair AND the same endpoints — the two arcs of a bigon, as a cone
// crossing a cylinder cuts two branches between one face pair — would collide on the bare parent name
// AND on the endpoint midpoint. Ranking by the CURVE midpoint (edgeRankPoint) gives them distinct,
// stable keys, ordered by geometry not build order.
func TestRelineageRanksBigonByCurveMidpoint(t *testing.T) {
	bld := NewBuilder(false, NewLineage(Tok("f", "body", 0)))
	a := bld.AddVertex(math.P3(1, 0, 0), NewLineage(Tok("f", "vertex", 0)))
	b := bld.AddVertex(math.P3(-1, 0, 0), NewLineage(Tok("f", "vertex", 1)))
	// Two semicircle arcs a→b: e1 bows through (0,+1,0), e2 through (0,-1,0) — same endpoints, distinct
	// curve midpoints.
	up, _ := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 1, 0, stdmath.Pi)
	dn, _ := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 1, 0, -stdmath.Pi)
	e1 := bld.AddEdge(up, a, b, NewLineage(Tok("f", "edge", 0)))
	e2 := bld.AddEdge(dn, a, b, NewLineage(Tok("f", "edge", 1)))
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	f1 := bld.AddFace(pl, NewLineage(Tok("f", "face", 0)), OuterLoop(Fwd(e1), Rev(e2)))
	f2 := bld.AddFace(pl, NewLineage(Tok("f", "face", 1)), OuterLoop(Fwd(e2), Rev(e1)))
	body := bld.Build()

	provA, provB := NewLineage(Tok("g", "a", 0)), NewLineage(Tok("g", "b", 0))
	body.RelineageByFaceProvenance(map[*Face]Lineage{f1: provA, f2: provB}, Tok("x", "x", 0), Tok("x", "seg", 0))

	k1, k2 := string(e1.Lineage().Key()), string(e2.Lineage().Key())
	if k1 == k2 {
		t.Fatalf("bigon arcs collide on %q — curve-midpoint rank disambiguation missing", k1)
	}
	for _, k := range []string{k1, k2} {
		if !strings.HasPrefix(k, "g:a#0/x:x#0/g:b#0/x:seg#") {
			t.Errorf("ranked bigon arc = %q, want prefix g:a#0/x:x#0/g:b#0/x:seg#", k)
		}
	}
	// e2 (curve midpoint y=-1) sorts before e1 (y=+1), so e2 is rank 1, e1 rank 2 — geometry, not order.
	if !strings.HasSuffix(k2, "x:seg#1") || !strings.HasSuffix(k1, "x:seg#2") {
		t.Errorf("ranks not ordered by curve midpoint: e1=%q e2=%q", k1, k2)
	}
}

// TestRelineageSkipsUnprovenancedFaces leaves an entity untouched when one of its faces has no
// provenance — so a partially-provenanced body keeps its build-order names where provenance is unknown.
func TestRelineageSkipsUnprovenancedFaces(t *testing.T) {
	bld := NewBuilder(false, NewLineage(Tok("f", "body", 0)))
	a := bld.AddVertex(math.P3(0, 0, 0), NewLineage(Tok("f", "vertex", 0)))
	b := bld.AddVertex(math.P3(1, 0, 0), NewLineage(Tok("f", "vertex", 1)))
	c := bld.AddVertex(math.P3(0, 1, 0), NewLineage(Tok("f", "vertex", 2)))
	ab := bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, NewLineage(Tok("f", "edge", 0)))
	bc := bld.AddEdge(geom.NewLineSegment(b.Point(), c.Point()), b, c, NewLineage(Tok("f", "edge", 1)))
	ca := bld.AddEdge(geom.NewLineSegment(c.Point(), a.Point()), c, a, NewLineage(Tok("f", "edge", 2)))
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(pl, NewLineage(Tok("f", "face", 0)), OuterLoop(Fwd(ab), Fwd(bc), Fwd(ca)))
	body := bld.Build()

	before := string(ab.Lineage().Key())
	body.RelineageByFaceProvenance(map[*Face]Lineage{}, Tok("x", "x", 0), Tok("x", "seg", 0)) // no face provenanced
	if got := string(ab.Lineage().Key()); got != before {
		t.Errorf("edge renamed despite no face provenance: %q → %q", before, got)
	}
}

// TestInheritOriginalEdges pins ADR-0043 P2′: a result edge coincident with an input edge takes
// that edge's lineage (an unsplit survivor), while a result edge matching no input edge keeps its
// build-order fallback. Only a 1:1 match inherits.
func TestInheritOriginalEdges(t *testing.T) {
	bld := NewBuilder(false, NewLineage(Tok("res", "body", 0)))
	a := bld.AddVertex(math.P3(0, 0, 0), NewLineage(Tok("res", "v", 0)))
	b := bld.AddVertex(math.P3(1, 0, 0), NewLineage(Tok("res", "v", 1)))
	c := bld.AddVertex(math.P3(0, 1, 0), NewLineage(Tok("res", "v", 2)))
	ab := bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, NewLineage(Tok("brep", "edge", 0)))
	bc := bld.AddEdge(geom.NewLineSegment(b.Point(), c.Point()), b, c, NewLineage(Tok("brep", "edge", 1)))
	ca := bld.AddEdge(geom.NewLineSegment(c.Point(), a.Point()), c, a, NewLineage(Tok("brep", "edge", 2)))
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(pl, NewLineage(Tok("res", "face", 0)), OuterLoop(Fwd(ab), Fwd(bc), Fwd(ca)))
	body := bld.Build()

	// Original edges: one coincident with ab (reversed, to exercise both directions), one far away.
	ob := NewBuilder(false, NewLineage(Tok("orig", "body", 0)))
	o1 := ob.AddVertex(math.P3(1, 0, 0), NewLineage(Tok("orig", "v", 0)))
	o2 := ob.AddVertex(math.P3(0, 0, 0), NewLineage(Tok("orig", "v", 1)))
	origAB := ob.AddEdge(geom.NewLineSegment(o1.Point(), o2.Point()), o1, o2, NewLineage(Tok("Extrusion1", "bottom-edge", 3)))
	f1 := ob.AddVertex(math.P3(9, 9, 9), NewLineage(Tok("orig", "v", 2)))
	f2 := ob.AddVertex(math.P3(9, 9, 8), NewLineage(Tok("orig", "v", 3)))
	far := ob.AddEdge(geom.NewLineSegment(f1.Point(), f2.Point()), f1, f2, NewLineage(Tok("Extrusion1", "x", 0)))

	body.InheritOriginalEdges([]*Edge{origAB, far})

	if got := string(ab.Lineage().Key()); got != "Extrusion1:bottom-edge#3" {
		t.Errorf("coincident edge ab = %q, want it to inherit Extrusion1:bottom-edge#3", got)
	}
	if got := string(bc.Lineage().Key()); got != "brep:edge#1" {
		t.Errorf("non-matching edge bc = %q, want it unchanged (brep:edge#1)", got)
	}
}
