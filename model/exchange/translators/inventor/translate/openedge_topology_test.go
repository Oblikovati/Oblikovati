// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
)

// Regression for Oblikovati/Oblikovati#3450: openEdgeCount must read the body's edge topology
// (an edge used by fewer than two faces is open), NOT a tessellation of it. The count therefore
// equals ops.BoundaryEdges exactly and is independent of tessellation quality.

// openQuadBody builds a single planar quad face — an open shell whose four straight edges are each
// used by exactly one face, so it has four boundary (open) edges by topology.
func openQuadBody() *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("quad", "body", 0)))
	p := m.P3
	v0 := bld.AddVertex(p(0, 0, 0), topo.NewLineage(topo.Tok("quad", "v", 0)))
	v1 := bld.AddVertex(p(2, 0, 0), topo.NewLineage(topo.Tok("quad", "v", 1)))
	v2 := bld.AddVertex(p(2, 2, 0), topo.NewLineage(topo.Tok("quad", "v", 2)))
	v3 := bld.AddVertex(p(0, 2, 0), topo.NewLineage(topo.Tok("quad", "v", 3)))
	e0 := bld.AddEdge(geom.NewLineSegment(p(0, 0, 0), p(2, 0, 0)), v0, v1, topo.NewLineage(topo.Tok("quad", "e", 0)))
	e1 := bld.AddEdge(geom.NewLineSegment(p(2, 0, 0), p(2, 2, 0)), v1, v2, topo.NewLineage(topo.Tok("quad", "e", 1)))
	e2 := bld.AddEdge(geom.NewLineSegment(p(2, 2, 0), p(0, 2, 0)), v2, v3, topo.NewLineage(topo.Tok("quad", "e", 2)))
	e3 := bld.AddEdge(geom.NewLineSegment(p(0, 2, 0), p(0, 0, 0)), v3, v0, topo.NewLineage(topo.Tok("quad", "e", 3)))
	surf, _ := geom.NewPlane(p(0, 0, 0), m.V3(0, 0, 1))
	bld.AddFace(surf, topo.NewLineage(topo.Tok("quad", "f", 0)),
		topo.OuterLoop(topo.Fwd(e0), topo.Fwd(e1), topo.Fwd(e2), topo.Fwd(e3)))
	return bld.Build()
}

func TestOpenEdgeCountMatchesTopology(t *testing.T) {
	cube, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(5, 5, 5), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	// A curved closed solid: its lateral surface and caps are tessellated with a seam and per-face
	// meshes, so a mesh-welded count is only 0 if every seam happens to weld. Topology guarantees 0.
	cyl, err := brep.SolidCylinder(m.P3(0, 0, 0), m.V3(0, 0, 1), 3, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	cases := []struct {
		name string
		body *topo.Body
		want int
	}{
		{"closed cube", cube, 0},
		{"closed cylinder", cyl, 0},
		{"open quad", openQuadBody(), 4},
	}
	for _, c := range cases {
		got := openEdgeCount(c.body)
		if got != c.want {
			t.Errorf("%s: openEdgeCount = %d, want %d", c.name, got, c.want)
		}
		if topo := len(ops.BoundaryEdges(c.body)); got != topo {
			t.Errorf("%s: openEdgeCount %d disagrees with ops.BoundaryEdges %d", c.name, got, topo)
		}
	}
}
