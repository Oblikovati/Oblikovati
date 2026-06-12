// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"bytes"

	"oblikovati.org/math"
)

// Body (SurfaceBody) is the top of the topology graph: one or more shells. A solid
// body is bounded by closed shells; a surface body may have open shells.
type Body struct {
	id      uint64
	shells  []*Shell
	wires   []*Wire
	solid   bool
	lineage Lineage
}

func (b *Body) ID() uint64           { return b.id }
func (b *Body) Kind() EntityKind     { return KindBody }
func (b *Body) Lineage() Lineage     { return b.lineage }
func (b *Body) ReferenceKey() []byte { return referenceKey(KindBody, b.lineage) }

// IsSolid reports whether the body is a solid (vs a surface body).
func (b *Body) IsSolid() bool { return b.solid }

// Shells returns the body's shells.
func (b *Body) Shells() []*Shell { return append([]*Shell(nil), b.shells...) }

// Faces returns every face in the body, across all shells.
func (b *Body) Faces() []*Face {
	var out []*Face
	for _, s := range b.shells {
		out = append(out, s.faces...)
	}
	return out
}

// Edges returns every distinct edge in the body.
func (b *Body) Edges() []*Edge {
	seen := map[*Edge]bool{}
	var out []*Edge
	for _, f := range b.Faces() {
		for _, e := range f.Edges() {
			if !seen[e] {
				seen[e] = true
				out = append(out, e)
			}
		}
	}
	return out
}

// Vertices returns every distinct vertex in the body.
func (b *Body) Vertices() []*Vertex {
	seen := map[*Vertex]bool{}
	var out []*Vertex
	for _, e := range b.Edges() {
		for _, v := range e.Vertices() {
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	return out
}

// RangeBox returns the body's bounding box, sampling curved edges so cylinders,
// cones and arcs are bounded by their true silhouette rather than collapsing to
// their seam vertices (see extendBoxByEdges).
func (b *Body) RangeBox() math.Box {
	box := math.EmptyBox()
	for _, v := range b.Vertices() {
		box = box.ExtendPoint(v.point)
	}
	return extendBoxByEdges(box, b.Edges())
}

// FindFaceByKey re-binds a face reference key to the matching face by lineage,
// returning false if the topology no longer contains it. This is the
// rebind-after-recompute mechanism, proven against a rebuilt body: a face recreated
// with the same lineage is re-found even though it is a different object.
func (b *Body) FindFaceByKey(key []byte) (*Face, bool) {
	for _, f := range b.Faces() {
		if bytes.Equal(f.ReferenceKey(), key) {
			return f, true
		}
	}
	return nil, false
}

// FindEdgeByKey re-binds an edge reference key by lineage.
func (b *Body) FindEdgeByKey(key []byte) (*Edge, bool) {
	for _, e := range b.Edges() {
		if bytes.Equal(e.ReferenceKey(), key) {
			return e, true
		}
	}
	return nil, false
}

// FindVertexByKey re-binds a vertex reference key by lineage — used to resolve a picked
// B-rep vertex as a work-feature point input after the body is rebuilt.
func (b *Body) FindVertexByKey(key []byte) (*Vertex, bool) {
	for _, v := range b.Vertices() {
		if bytes.Equal(v.ReferenceKey(), key) {
			return v, true
		}
	}
	return nil, false
}

// MergeBodies combines the shells of several bodies into one (a multi-lump body),
// re-parenting each shell. Used by a boolean Join of non-touching bodies
// (kernel/ops). The input bodies should not be used afterward.
func MergeBodies(lineage Lineage, solid bool, bodies ...*Body) *Body {
	merged := &Body{id: nextID(), solid: solid, lineage: lineage}
	for _, b := range bodies {
		for _, sh := range b.shells {
			sh.body = merged
			merged.shells = append(merged.shells, sh)
		}
	}
	return merged
}
