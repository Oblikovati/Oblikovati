// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A reconstructed body must not carry a spurious 2-valent collinear vertex — a straight edge the mesh
// arrangement split mid-way (#2247). This builds a unit box whose top-front edge (0,0,1)-(1,0,1) is
// split at (0.5,0,1) and asserts the dissolve merges it back: the vertex is gone and the edge whole,
// while a genuine box corner (3-valent) is untouched.

// splitBox builds the unit cube with the edge (0,0,1)-(1,0,1) split at m=(0.5,0,1): the top and front
// faces each carry the extra vertex, so it is a valid 2-valent collinear split.
func splitBox(t *testing.T) *topo.Body {
	t.Helper()
	bld := topo.NewBuilder(true, lin("box"))
	v := map[string]*topo.Vertex{}
	add := func(name string, x, y, z float64) { v[name] = bld.AddVertex(math.P3(x, y, z), lin("v", name)) }
	add("000", 0, 0, 0)
	add("100", 1, 0, 0)
	add("110", 1, 1, 0)
	add("010", 0, 1, 0)
	add("001", 0, 0, 1)
	add("101", 1, 0, 1)
	add("111", 1, 1, 1)
	add("011", 0, 1, 1)
	add("m", 0.5, 0, 1)

	e := map[string]*topo.Edge{}
	mk := func(a, b string) {
		e[a+b] = bld.AddEdge(geom.NewLineSegment(v[a].Point(), v[b].Point()), v[a], v[b], lin("e", a, b))
	}
	mk("000", "100")
	mk("100", "110")
	mk("110", "010")
	mk("010", "000")
	mk("001", "m")
	mk("m", "101") // the split top-front edge, in two collinear pieces
	mk("101", "111")
	mk("111", "011")
	mk("011", "001")
	mk("000", "001")
	mk("100", "101")
	mk("110", "111")
	mk("010", "011")

	use := func(a, b string) topo.Use {
		if edge, ok := e[a+b]; ok {
			return topo.Use{Edge: edge}
		}
		return topo.Use{Edge: e[b+a], Reversed: true}
	}
	face := func(normal math.Vector3, ring ...string) {
		pl, _ := geom.NewPlane(v[ring[0]].Point(), normal)
		uses := make([]topo.Use, 0, len(ring))
		for i := range ring {
			uses = append(uses, use(ring[i], ring[(i+1)%len(ring)]))
		}
		bld.AddFace(pl, lin("f"), topo.OuterLoop(uses...))
	}
	face(math.V3(0, 0, -1), "000", "010", "110", "100")      // bottom
	face(math.V3(0, 0, 1), "001", "m", "101", "111", "011")  // top (split)
	face(math.V3(0, -1, 0), "000", "100", "101", "m", "001") // front (split)
	face(math.V3(0, 1, 0), "010", "011", "111", "110")       // back
	face(math.V3(-1, 0, 0), "000", "001", "011", "010")      // left
	face(math.V3(1, 0, 0), "100", "110", "111", "101")       // right
	return bld.Build()
}

func TestDissolveMergesCollinearSplit(t *testing.T) {
	t.Parallel()
	body := splitBox(t)
	if got := vertexAt(body, math.P3(0.5, 0, 1)); got == nil {
		t.Fatal("fixture: split vertex (0.5,0,1) missing")
	}
	e0 := len(body.Edges())
	out := dissolveCollinearVertices(body)
	if v := vertexAt(out, math.P3(0.5, 0, 1)); v != nil {
		t.Errorf("dissolve left the spurious 2-valent collinear vertex (0.5,0,1)")
	}
	if got := len(out.Edges()); got != e0-1 {
		t.Errorf("dissolve edge count = %d, want %d (one merged)", got, e0-1)
	}
	if v := vertexAt(out, math.P3(0, 0, 0)); v == nil || len(v.Edges()) != 3 {
		t.Errorf("a genuine 3-valent corner (0,0,0) must be kept intact")
	}
}

// vertexAt returns the body vertex at p (within 1e-9), or nil.
func vertexAt(b *topo.Body, p math.Point3) *topo.Vertex {
	for _, v := range b.Vertices() {
		if v.Point().IsEqualTo(p, 1e-9) {
			return v
		}
	}
	return nil
}

// TestDissolveNoOpOnCleanBodyAndNil: a nil body and a clean box (every vertex a genuine 3-valent
// corner, no 2-valent collinear split) are returned unchanged — the dissolve only fires on a spurious
// split.
func TestDissolveNoOpOnCleanBodyAndNil(t *testing.T) {
	t.Parallel()
	if dissolveCollinearVertices(nil) != nil {
		t.Error("nil body must pass through")
	}
	box, err := SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "b")
	if err != nil {
		t.Fatal(err)
	}
	if dissolveCollinearVertices(box) != box {
		t.Error("a clean body with no 2-valent collinear vertex must be returned unchanged")
	}
}

// TestLessPointTotalOrder covers the point tie-break (X, then Y, then Z) that makes the merged edge's
// endpoint order — and thus the result — deterministic across runs.
func TestLessPointTotalOrder(t *testing.T) {
	t.Parallel()
	cases := []struct {
		p, q math.Point3
		want bool
	}{
		{math.P3(0, 9, 9), math.P3(1, 0, 0), true},  // X decides
		{math.P3(1, 0, 0), math.P3(0, 9, 9), false}, // X decides (other way)
		{math.P3(2, 0, 9), math.P3(2, 1, 0), true},  // Y decides
		{math.P3(2, 1, 0), math.P3(2, 0, 9), false}, // Y decides (other way)
		{math.P3(2, 2, 0), math.P3(2, 2, 1), true},  // Z decides
		{math.P3(2, 2, 1), math.P3(2, 2, 1), false}, // equal → not less
	}
	for _, c := range cases {
		if got := lessPoint(c.p, c.q); got != c.want {
			t.Errorf("lessPoint(%v,%v) = %v, want %v", c.p, c.q, got, c.want)
		}
	}
}

// TestOrderEndsIsPointOrdered covers both branches of orderEnds: it returns the two vertices sorted by
// point regardless of the argument order.
func TestOrderEndsIsPointOrdered(t *testing.T) {
	t.Parallel()
	bld := topo.NewBuilder(true, lin("t"))
	lo := bld.AddVertex(math.P3(0, 0, 0), lin("lo"))
	hi := bld.AddVertex(math.P3(1, 0, 0), lin("hi"))
	for _, in := range [][2]*topo.Vertex{{lo, hi}, {hi, lo}} {
		a, b := orderEnds(in[0], in[1])
		if a != lo || b != hi {
			t.Errorf("orderEnds(%v) did not sort to (lo,hi)", in)
		}
	}
}

func lin(tokens ...string) topo.Lineage {
	toks := make([]topo.LineageToken, len(tokens))
	for i, s := range tokens {
		toks[i] = topo.Tok("t", s, i)
	}
	return topo.NewLineage(toks...)
}
