// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// rebuildWithRimFillet rebuilds the body with the rim rounded: every vertex/edge/face is copied
// (the TransformBody pattern) EXCEPT the rim circle and the rim vertex (removed) and the wall seam
// (re-aimed to the receded cyl-tangent vertex). It then inserts the cyl-tangent + cap-tangent circles,
// the torus seam, re-trims the cylinder wall and cap onto them, and adds the torus band.
func rebuildWithRimFillet(b *topo.Body, rf *rimFillet) (*topo.Body, error) {
	g := &rimBuild{rf: rf, bld: topo.NewBuilder(b.IsSolid(), b.Lineage()), verts: map[*topo.Vertex]*topo.Vertex{}, edges: map[*topo.Edge]*topo.Edge{}}
	g.copyVerts(b)
	g.addRimVerts()
	g.copyEdges(b)
	g.addRimEdges()
	for _, f := range b.Faces() {
		g.copyFace(f)
	}
	g.addTorusFace()
	return g.bld.Build(), nil
}

// rimBuild carries the in-progress rebuild: the old→new vertex/edge maps plus the new rim entities.
type rimBuild struct {
	rf    *rimFillet
	bld   *topo.Builder
	verts map[*topo.Vertex]*topo.Vertex
	edges map[*topo.Edge]*topo.Edge
	vc    *topo.Vertex // cyl-tangent seam vertex (replaces the rim vertex on the wall)
	vt    *topo.Vertex // cap-tangent seam vertex
	cylE  *topo.Edge   // cyl-tangent circle
	capE  *topo.Edge   // cap-tangent circle
	seamE *topo.Edge   // torus seam arc vc→vt
	wallE *topo.Edge   // re-aimed wall seam bottom→vc
}

func (g *rimBuild) copyVerts(b *topo.Body) {
	for _, v := range b.Vertices() {
		if v == g.rf.rimV {
			continue // the rim vertex is replaced by the two tangent-circle seam vertices
		}
		g.verts[v] = g.bld.AddVertex(v.Point(), v.Lineage())
	}
}

func (g *rimBuild) addRimVerts() {
	g.vc = g.bld.AddVertex(g.rf.cylTan.PointAt(0), topo.NewLineage(topo.Tok("rimfillet", "vc", 0)))
	g.vt = g.bld.AddVertex(g.rf.capTan.PointAt(0), topo.NewLineage(topo.Tok("rimfillet", "vt", 0)))
}

func (g *rimBuild) copyEdges(b *topo.Body) {
	for _, e := range b.Edges() {
		if e == g.rf.rimEdge || e == g.rf.seamEdge {
			continue // rim removed; wall seam re-aimed (added in addRimEdges)
		}
		g.edges[e] = g.bld.AddEdge(e.Geometry(), g.verts[e.StartVertex()], g.verts[e.EndVertex()], e.Lineage())
	}
}

func (g *rimBuild) addRimEdges() {
	lin := func(role string) topo.Lineage { return topo.NewLineage(topo.Tok("rimfillet", role, 0)) }
	g.cylE = g.bld.AddEdge(g.rf.cylTan, g.vc, g.vc, lin("cyltan"))
	g.capE = g.bld.AddEdge(g.rf.capTan, g.vt, g.vt, lin("captan"))
	mid := g.rf.torus.PointAt(0, quarterTube) // the seam arc's midpoint: v=π/4, halfway cyl-tangent→cap-tangent
	seam, _ := geom.Arc3dByThreePoints(g.rf.cylTan.PointAt(0), mid, g.rf.capTan.PointAt(0))
	g.seamE = g.bld.AddEdge(seam, g.vc, g.vt, lin("seam"))
	bottom := g.verts[g.rf.bottomV]
	g.wallE = g.bld.AddEdge(geom.NewLineSegment(bottom.Point(), g.vc.Point()), bottom, g.vc, lin("wallseam"))
}

// quarterTube is v=π/4 — the tube midpoint between the cyl-tangent contact (v=0) and the cap-tangent
// contact (v=π/2) of a convex rim, used as the seam arc's on-arc point.
const quarterTube = 0.7853981633974483

// copyFace copies one face, re-aiming the cylinder wall and the cap onto the new circles and leaving
// every other face untouched.
func (g *rimBuild) copyFace(f *topo.Face) {
	specs := g.loopSpecsWithRim(f)
	if f.Reversed() {
		g.bld.AddReversedFace(f.Geometry(), f.Lineage(), specs...)
		return
	}
	g.bld.AddFace(f.Geometry(), f.Lineage(), specs...)
}

// loopSpecsWithRim rebuilds a face's loops against the new edges, substituting the rim circle (→ the
// cyl-tangent circle on the wall, the cap-tangent circle on the cap) and the wall seam (→ re-aimed).
func (g *rimBuild) loopSpecsWithRim(f *topo.Face) []topo.LoopSpec {
	specs := make([]topo.LoopSpec, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		uses := make([]topo.Use, 0, len(l.EdgeUses()))
		for _, u := range l.EdgeUses() {
			uses = append(uses, g.mapUse(f, u))
		}
		if l.IsOuter() {
			specs = append(specs, topo.OuterLoop(uses...))
		} else {
			specs = append(specs, topo.InnerLoop(uses...))
		}
	}
	return specs
}

// mapUse maps one edge use onto the rebuilt edges, swapping the rim and the wall seam.
func (g *rimBuild) mapUse(f *topo.Face, u *topo.EdgeUse) topo.Use {
	switch u.Edge() {
	case g.rf.rimEdge:
		if f == g.rf.cap {
			return topo.Use{Edge: g.capE, Reversed: u.Reversed()}
		}
		return topo.Use{Edge: g.cylE, Reversed: u.Reversed()}
	case g.rf.seamEdge:
		return topo.Use{Edge: g.wallE, Reversed: u.Reversed()}
	}
	return topo.Use{Edge: g.edges[u.Edge()], Reversed: u.Reversed()}
}

// addTorusFace adds the toroidal band: seam up the tube, around the cap-tangent circle (opposite the
// cap), seam down, around the cyl-tangent circle (opposite the wall) — the SolidCylinderFilletedTop
// pattern, so each circle is shared with its neighbour in the opposite orientation.
func (g *rimBuild) addTorusFace() {
	g.bld.AddFace(g.rf.torus, topo.NewLineage(topo.Tok("rimfillet", "torus", 0)),
		topo.OuterLoop(topo.Fwd(g.seamE), topo.Rev(g.capE), topo.Rev(g.seamE), topo.Fwd(g.cylE)))
}
