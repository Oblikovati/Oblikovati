// SPDX-License-Identifier: GPL-2.0-only

package subd

import (
	"testing"

	"oblikovati/math"
)

// TestQuadBall projects a twice-subdivided box onto a sphere; every vertex must then sit at
// the requested radius from the center the primitive uses, (1,1,1).
func TestQuadBall(t *testing.T) {
	const r = 3.0
	center := math.P3(1, 1, 1)
	m := QuadBall(r)
	if len(m.Verts) == 0 {
		t.Fatal("quad ball has no vertices")
	}
	for i, p := range m.Verts {
		if d := p.DistanceTo(center); !approx(d, r, 1e-9) {
			t.Fatalf("vertex %d at radius %g, want %g", i, d, r)
		}
	}
}

// TestEdgeListDeterministic checks a unit box reports its 12 undirected edges and that the
// order is stable across calls (the free-form UI relies on a stable selection index).
func TestEdgeListDeterministic(t *testing.T) {
	m := Box(1, 1, 1)
	e1 := m.EdgeList()
	if len(e1) != 12 {
		t.Fatalf("box has %d edges, want 12", len(e1))
	}
	e2 := m.EdgeList()
	for i := range e1 {
		if e1[i] != e2[i] {
			t.Fatalf("edge order not stable at %d: %v vs %v", i, e1[i], e2[i])
		}
	}
}

// TestSetCrease covers allocation on first use, clamping above 1, and removal at ≤0.
func TestSetCrease(t *testing.T) {
	m := Box(1, 1, 1)
	m.SetCrease(0, 1, 2.0) // over-range → clamped to 1, allocates the map
	if got := m.Creases[edgeKey(0, 1)]; got != 1 {
		t.Errorf("crease = %g, want clamped to 1", got)
	}
	m.SetCrease(0, 1, 0) // ≤0 removes it
	if _, ok := m.Creases[edgeKey(0, 1)]; ok {
		t.Error("crease should be removed at sharpness 0")
	}
}

// TestCloneIsDeep confirms Clone copies verts, faces, and creases so edits don't alias.
func TestCloneIsDeep(t *testing.T) {
	m := Box(1, 1, 1)
	m.SetCrease(0, 1, 0.5)
	c := m.Clone()
	c.Verts[0] = c.Verts[0].TranslateBy(math.V3(10, 0, 0))
	c.Faces[0][0] = -1
	c.Creases[edgeKey(0, 1)] = 0.9
	if m.Verts[0] == c.Verts[0] {
		t.Error("clone aliases the vertex slice")
	}
	if m.Faces[0][0] == -1 {
		t.Error("clone aliases the face slice")
	}
	if m.Creases[edgeKey(0, 1)] != 0.5 {
		t.Error("clone aliases the crease map")
	}
}
