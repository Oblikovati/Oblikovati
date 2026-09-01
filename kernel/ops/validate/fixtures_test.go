// SPDX-License-Identifier: GPL-2.0-only

package validate

// Fixture builders restated from kernel/ops' test package. Go cannot share a _test.go
// helper across packages, and a shared fixture package would import kernel/ops, which
// kernel/ops' own tests could then not use (import cycle). This is the test scaffolding
// sonar.cpd.exclusions already accounts for. They build bodies through topo alone, so
// they carry no dependency on the operation layer.

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// tetra builds a unit tetrahedron scaled by s and translated by off, with lineage.
func tetra(s float64, off math.Vector3) *topo.Body {
	feat := "f"
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok(feat, "body", 0)))
	p := func(x, y, z float64) math.Point3 { return math.P3(x*s, y*s, z*s).TranslateBy(off) }
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok(feat, role, i)) }
	a := bld.AddVertex(p(0, 0, 0), lin("vertex", 0))
	b := bld.AddVertex(p(1, 0, 0), lin("vertex", 1))
	c := bld.AddVertex(p(0, 1, 0), lin("vertex", 2))
	d := bld.AddVertex(p(0, 0, 1), lin("vertex", 3))
	seg := func(x, y *topo.Vertex, i int) *topo.Edge {
		return bld.AddEdge(geom.NewLineSegment(x.Point(), y.Point()), x, y, lin("edge", i))
	}
	ab, ac, ad := seg(a, b, 0), seg(a, c, 1), seg(a, d, 2)
	bc, bd, cd := seg(b, c, 3), seg(b, d, 4), seg(c, d, 5)
	pl := func(o, n math.Vector3) geom.Surface { s, _ := geom.NewPlane(o.AsPoint().TranslateBy(off), n); return s }
	// Consistently-outward loops: each shared edge is traversed in opposite
	// directions by its two faces (one Fwd, one Rev), as a valid manifold requires.
	bld.AddFace(pl(math.V3(0, 0, 0), math.V3(0, 0, -1)), lin("face", 0), topo.OuterLoop(topo.Fwd(ac), topo.Rev(bc), topo.Rev(ab)))
	bld.AddFace(pl(math.V3(0, 0, 0), math.V3(0, -1, 0)), lin("face", 1), topo.OuterLoop(topo.Fwd(ab), topo.Fwd(bd), topo.Rev(ad)))
	bld.AddFace(pl(math.V3(0, 0, 0), math.V3(-1, 0, 0)), lin("face", 2), topo.OuterLoop(topo.Fwd(ad), topo.Rev(cd), topo.Rev(ac)))
	// The slant plane must actually contain b,c,d (the plane x+y+z=s): its origin is vertex b,
	// scaled with s. A fixed (1,1,1) origin puts the plane at x+y+z=3 — off the face for any s≠3,
	// which the analytic point classifier (unlike mesh tessellation) correctly rejects.
	bld.AddFace(pl(math.V3(1, 0, 0).Scale(s), math.V3(1, 1, 1)), lin("face", 3), topo.OuterLoop(topo.Fwd(bc), topo.Fwd(cd), topo.Rev(bd)))
	return bld.Build()
}

// near reports whether two parameter samples coincide within tol (inclusive, so exact
// duplicates collapse even on a degenerate zero-span axis).
func near(a, b, tol float64) bool { return stdmath.Abs(a-b) <= tol }
