// SPDX-License-Identifier: GPL-2.0-only

package transform

import (
	"bytes"
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
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
		if err := transformFaceInto(bld, f, m, edges, reflected, derive); err != nil {
			return nil, err
		}
	}
	out := bld.Build() // multi-shell sources regroup by connectivity here (#629)
	transformWiresInto(out, b, m, edges, derive)
	return out, nil
}

// transformFaceInto clones one face with its surface mapped, PRESERVING its
// material sense — a reversed face (a cut/bore wall) must stay reversed or its
// tessellation winds inward and the divergence-theorem volume flips on it.
func transformFaceInto(bld *topo.Builder, f *topo.Face, m math.Matrix4, edges map[*topo.Edge]*topo.Edge, reflected bool, derive func(topo.Lineage) topo.Lineage) error {
	surface, err := geom.TransformSurface(f.Geometry(), m)
	if err != nil {
		return fmt.Errorf("ops.TransformBody: face %d: %w", f.ID(), err)
	}
	if f.Reversed() {
		bld.AddReversedFace(surface, derive(f.Lineage()), loopSpecs(f, edges, reflected)...)
		return nil
	}
	bld.AddFace(surface, derive(f.Lineage()), loopSpecs(f, edges, reflected)...)
	return nil
}

// transformWiresInto carries the source body's wires onto the transformed copy,
// re-aiming each use at the cloned edges. Wire edges not shared with any face
// are cloned here (they were absent from the face pass).
func transformWiresInto(dst, src *topo.Body, m math.Matrix4, edges map[*topo.Edge]*topo.Edge, derive func(topo.Lineage) topo.Lineage) {
	for _, w := range src.Wires() {
		uses := make([]topo.Use, 0, len(w.Uses()))
		for _, u := range w.Uses() {
			clone, ok := edges[u.Edge]
			if !ok {
				clone = cloneWireEdge(u.Edge, m, derive)
				edges[u.Edge] = clone
			}
			uses = append(uses, topo.Use{Edge: clone, Reversed: u.Reversed})
		}
		dst.AttachWire(derive(w.Lineage()), uses)
	}
}

// cloneWireEdge maps a face-less wire edge (fresh vertices, transformed curve).
// An untransformable curve keeps its polyline sampling — wires are sampled
// consumers, so the seam stays faithful.
func cloneWireEdge(e *topo.Edge, m math.Matrix4, derive func(topo.Lineage) topo.Lineage) *topo.Edge {
	bld := topo.NewBuilder(false, derive(e.Lineage()))
	curve, err := geom.TransformCurve(e.Geometry(), m)
	if err != nil {
		curve = transformSampledCurve(e.Geometry(), m)
	}
	s := bld.AddVertex(m.TransformPoint(e.StartVertex().Point()), derive(e.StartVertex().Lineage()))
	t := bld.AddVertex(m.TransformPoint(e.EndVertex().Point()), derive(e.EndVertex().Lineage()))
	return bld.AddEdge(curve, s, t, derive(e.Lineage()))
}

// transformSampledCurve maps a curve with no analytic transform as a polyline.
func transformSampledCurve(c geom.Curve3, m math.Matrix4) geom.Curve3 {
	const samples = 64
	lo, hi := c.Domain()
	pts := make([]math.Point3, samples+1)
	for i := 0; i <= samples; i++ {
		pts[i] = m.TransformPoint(c.PointAt(lo + (hi-lo)*float64(i)/samples))
	}
	poly, _ := geom.NewPolyline(pts)
	return poly
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

// ReplaceFaceSurface clones a body, swapping the surface of the face with the given reference
// key for `surf` and leaving every vertex/edge/face topology intact — the cheap (no-boolean)
// way to retype a face, e.g. plain cylinder → threaded cylinder. The new surface must still
// pass through the face's boundary edges (a thread's runout ensures this at the end circles).
func ReplaceFaceSurface(b *topo.Body, faceKey []byte, surf geom.Surface) (*topo.Body, error) {
	if n := len(b.Shells()); n != 1 {
		return nil, fmt.Errorf("ops.ReplaceFaceSurface: %d shells; only single-shell bodies are supported", n)
	}
	bld := topo.NewBuilder(b.IsSolid(), b.Lineage())
	verts := make(map[*topo.Vertex]*topo.Vertex, len(b.Vertices()))
	for _, v := range b.Vertices() {
		verts[v] = bld.AddVertex(v.Point(), v.Lineage())
	}
	edges := make(map[*topo.Edge]*topo.Edge, len(b.Edges()))
	for _, e := range b.Edges() {
		edges[e] = bld.AddEdge(e.Geometry(), verts[e.StartVertex()], verts[e.EndVertex()], e.Lineage())
	}
	found := false
	for _, f := range b.Faces() {
		s := f.Geometry()
		if bytes.Equal(f.ReferenceKey(), faceKey) {
			s, found = surf, true
		}
		// Preserve the face's sense: a reversed face (e.g. a bore wall, whose outward-from-solid
		// normal opposes its surface's +radial normal) must stay reversed, or its tessellation
		// winds the wrong way and the divergence-theorem volume flips sign on that face.
		if f.Reversed() {
			bld.AddReversedFace(s, f.Lineage(), loopSpecs(f, edges, false)...)
		} else {
			bld.AddFace(s, f.Lineage(), loopSpecs(f, edges, false)...)
		}
	}
	if !found {
		return nil, fmt.Errorf("ops.ReplaceFaceSurface: no face with the given key")
	}
	return bld.Build(), nil
}
