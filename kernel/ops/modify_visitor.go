// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Modification is the geometry-replacement visitor a local B-rep edit supplies — our
// BRepTools_Modification (ADR-0050 P7). For each face/edge/vertex it returns replacement geometry
// and true, or false to keep the original verbatim. It is what makes draft (and later shell,
// move-face, replace-face) work on CURVED bodies: the modifier re-intersects neighbours instead of
// casting every face to a plane (the rebuildWithPlanes limitation behind the #1802 crash).
type Modification interface {
	NewSurface(f *topo.Face) (geom.Surface, bool)
	NewCurve(e *topo.Edge) (geom.Curve3, bool)
	NewPoint(v *topo.Vertex) (math.Point3, bool)
}

// modifyBody rebuilds solid applying mod: each vertex, edge and face takes mod's replacement
// geometry when offered and is otherwise copied verbatim, while the combinatorial structure (loops,
// edge-uses, orientation) and the per-entity lineage are preserved (ADR-0043 — a selection on a
// modified face survives). Mirrors BRepTools_Modifier::Perform. tag names the rebuild's body token.
func modifyBody(solid *topo.Body, mod Modification, tag string) *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok(tag, "body", 0)))
	nv := modifiedVertices(bld, solid, mod, tag)
	ne := modifiedEdges(bld, solid, mod, nv, tag)
	modifiedFaces(bld, solid, mod, ne, tag)
	return bld.Build()
}

// modifiedVertices adds each vertex at its relocated point (NewPoint) or original position.
func modifiedVertices(bld *topo.Builder, solid *topo.Body, mod Modification, tag string) map[uint64]*topo.Vertex {
	nv := make(map[uint64]*topo.Vertex, len(solid.Vertices()))
	for i, v := range solid.Vertices() {
		p := v.Point()
		if np, ok := mod.NewPoint(v); ok {
			p = np
		}
		nv[v.ID()] = bld.AddVertex(p, cloneName(true, v.Lineage(), tag, "v", i))
	}
	return nv
}

// modifiedEdges adds each edge with its re-intersected curve (NewCurve) or, unchanged, its original
// curve — its endpoints did not move, which the modification guarantees.
func modifiedEdges(bld *topo.Builder, solid *topo.Body, mod Modification, nv map[uint64]*topo.Vertex, tag string) map[uint64]*topo.Edge {
	ne := make(map[uint64]*topo.Edge, len(solid.Edges()))
	for i, e := range solid.Edges() {
		crv, ok := mod.NewCurve(e)
		if !ok {
			crv = e.Geometry()
		}
		a, b := nv[e.StartVertex().ID()], nv[e.EndVertex().ID()]
		ne[e.ID()] = bld.AddEdge(crv, a, b, cloneName(true, e.Lineage(), tag, "e", i))
	}
	return ne
}

// modifiedFaces adds each face with its replacement surface (NewSurface) or original, preserving
// the loop structure against the rebuilt edges.
func modifiedFaces(bld *topo.Builder, solid *topo.Body, mod Modification, ne map[uint64]*topo.Edge, tag string) {
	for i, f := range solid.Faces() {
		s, ok := mod.NewSurface(f)
		if !ok {
			s = f.Geometry()
		}
		bld.AddFace(s, cloneName(true, f.Lineage(), tag, "f", i), cloneLoops(f, ne)...)
	}
}
