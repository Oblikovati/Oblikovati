// SPDX-License-Identifier: GPL-2.0-only

package brepfixture

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// tetraEdges names the six edges of a tetrahedron by their end vertices, so the four faces can be
// wound from them without repeating the vertex pairs.
type tetraEdges struct{ ab, ac, ad, bc, bd, cd *topo.Edge }

// Tetra builds a unit tetrahedron scaled by s and translated by off, carrying lineage. Four
// planar faces is the smallest closed solid there is, so it is the operand that isolates a
// topological question from any geometric complexity.
//
// Example: a, b := brepfixture.Tetra(1, math.V3(0,0,0)), brepfixture.Tetra(1, math.V3(0.2,0.2,0.2))
func Tetra(s float64, off math.Vector3) *topo.Body {
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("f", role, i)) }
	bld := topo.NewBuilder(true, lin("body", 0))
	e := tetraSkeleton(bld, s, off, lin)
	pl := func(o, n math.Vector3) geom.Surface {
		p, _ := geom.NewPlane(o.AsPoint().TranslateBy(off), n)
		return p
	}
	// Consistently-outward loops: each shared edge is traversed in opposite
	// directions by its two faces (one Fwd, one Rev), as a valid manifold requires.
	bld.AddFace(pl(math.V3(0, 0, 0), math.V3(0, 0, -1)), lin("face", 0), topo.OuterLoop(topo.Fwd(e.ac), topo.Rev(e.bc), topo.Rev(e.ab)))
	bld.AddFace(pl(math.V3(0, 0, 0), math.V3(0, -1, 0)), lin("face", 1), topo.OuterLoop(topo.Fwd(e.ab), topo.Fwd(e.bd), topo.Rev(e.ad)))
	bld.AddFace(pl(math.V3(0, 0, 0), math.V3(-1, 0, 0)), lin("face", 2), topo.OuterLoop(topo.Fwd(e.ad), topo.Rev(e.cd), topo.Rev(e.ac)))
	// The slant plane must actually contain b,c,d (the plane x+y+z=s): its origin is vertex b,
	// scaled with s. A fixed (1,1,1) origin puts the plane at x+y+z=3 — off the face for any s≠3,
	// which the analytic point classifier (unlike mesh tessellation) correctly rejects.
	bld.AddFace(pl(math.V3(1, 0, 0).Scale(s), math.V3(1, 1, 1)), lin("face", 3), topo.OuterLoop(topo.Fwd(e.bc), topo.Fwd(e.cd), topo.Rev(e.bd)))
	return bld.Build()
}

// tetraSkeleton adds the four corners and the six edges between them.
func tetraSkeleton(bld *topo.Builder, s float64, off math.Vector3, lin func(string, int) topo.Lineage) tetraEdges {
	p := func(x, y, z float64) math.Point3 { return math.P3(x*s, y*s, z*s).TranslateBy(off) }
	a := bld.AddVertex(p(0, 0, 0), lin("vertex", 0))
	b := bld.AddVertex(p(1, 0, 0), lin("vertex", 1))
	c := bld.AddVertex(p(0, 1, 0), lin("vertex", 2))
	d := bld.AddVertex(p(0, 0, 1), lin("vertex", 3))
	seg := func(x, y *topo.Vertex, i int) *topo.Edge {
		return bld.AddEdge(geom.NewLineSegment(x.Point(), y.Point()), x, y, lin("edge", i))
	}
	return tetraEdges{
		ab: seg(a, b, 0), ac: seg(a, c, 1), ad: seg(a, d, 2),
		bc: seg(b, c, 3), bd: seg(b, d, 4), cd: seg(c, d, 5),
	}
}
