// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// TransformBody returns a copy of b mapped by the similarity transform m
// (rotation / translation / reflection / uniform scale), preserving the exact
// combinatorial topology so the result stays a valid manifold.
//
// Each entity's lineage is remapped through derive: pass the identity for an
// in-place Move (reference keys are preserved, so picks survive), or a
// token-appending function for a pattern/mirror copy so every copy gets distinct
// reference keys. Reflection (negative determinant) reverses every loop's winding
// so face normals stay outward — the winding flip keeps each edge's two uses
// oppositely oriented, so [Validate] still passes.
//
// Example — reflect a body across the X... plane for a mirror feature:
//
//	dst, err := ops.TransformBody(src, reflectM, func(l topo.Lineage) topo.Lineage {
//	    return prepend(topo.Tok("mirror", "copy", 0), l)
//	})
func TransformBody(b *topo.Body, m math.Matrix4, derive func(topo.Lineage) topo.Lineage) (*topo.Body, error) {
	if n := len(b.Shells()); n != 1 {
		return nil, fmt.Errorf("ops.TransformBody: %d shells; only single-shell bodies are supported", n)
	}
	reflected := m.Determinant() < 0
	bld := topo.NewBuilder(b.IsSolid(), derive(b.Lineage()))

	verts := make(map[*topo.Vertex]*topo.Vertex, len(b.Vertices()))
	for _, v := range b.Vertices() {
		verts[v] = bld.AddVertex(m.TransformPoint(v.Point()), derive(v.Lineage()))
	}
	edges := make(map[*topo.Edge]*topo.Edge, len(b.Edges()))
	for _, e := range b.Edges() {
		curve, err := geom.TransformCurve(e.Geometry(), m)
		if err != nil {
			return nil, fmt.Errorf("ops.TransformBody: edge %d: %w", e.ID(), err)
		}
		edges[e] = bld.AddEdge(curve, verts[e.StartVertex()], verts[e.EndVertex()], derive(e.Lineage()))
	}
	for _, f := range b.Faces() {
		surface, err := geom.TransformSurface(f.Geometry(), m)
		if err != nil {
			return nil, fmt.Errorf("ops.TransformBody: face %d: %w", f.ID(), err)
		}
		bld.AddFace(surface, derive(f.Lineage()), loopSpecs(f, edges, reflected)...)
	}
	return bld.Build(), nil
}

// loopSpecs rebuilds a face's loop specs against the cloned edges, reversing the
// winding when the transform is a reflection so normals stay outward.
func loopSpecs(f *topo.Face, edges map[*topo.Edge]*topo.Edge, reflected bool) []topo.LoopSpec {
	specs := make([]topo.LoopSpec, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		oldUses := l.EdgeUses()
		uses := make([]topo.Use, len(oldUses))
		for i, u := range oldUses {
			uses[i] = topo.Use{Edge: edges[u.Edge()], Reversed: u.Reversed()}
		}
		if reflected {
			uses = reverseWinding(uses)
		}
		if l.IsOuter() {
			specs = append(specs, topo.OuterLoop(uses...))
		} else {
			specs = append(specs, topo.InnerLoop(uses...))
		}
	}
	return specs
}

// reverseWinding flips a loop's traversal: reverse the edge-use order and invert
// each reversed flag, yielding the same cycle walked the opposite way.
func reverseWinding(uses []topo.Use) []topo.Use {
	out := make([]topo.Use, len(uses))
	for i, u := range uses {
		out[len(uses)-1-i] = topo.Use{Edge: u.Edge, Reversed: !u.Reversed}
	}
	return out
}
