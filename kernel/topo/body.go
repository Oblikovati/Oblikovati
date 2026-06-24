// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"bytes"

	"oblikovati.org/math"
)

// Body (SurfaceBody) is the top of the topology graph: one or more shells. A solid
// body is bounded by closed shells; a surface body may have open shells.
//
// derived/cached* memoize the distinct face/edge/vertex lists, which the render hot paths
// (range box, drawlist) re-query every frame — re-deriving them rebuilt a dedup map each call,
// ~28% of orbit allocation at 30k (M34-F2b). They are written exactly once, by finalizeDerived
// at the end of every body-construction path (RegroupShells / BodyFromShells / MergeBodies), and
// read-only afterward, so reads need no lock even under the parallel flatten (which never queries
// them). Before finalize (derived==false) the accessors derive live, so a partially-built body —
// e.g. RegroupShells querying Faces() mid-regroup — still sees current geometry.
type Body struct {
	id             uint64
	shells         []*Shell
	wires          []*Wire
	solid          bool
	lineage        Lineage
	derived        bool
	cachedFaces    []*Face
	cachedEdges    []*Edge
	cachedVertices []*Vertex
}

func (b *Body) ID() uint64           { return b.id }
func (b *Body) Kind() EntityKind     { return KindBody }
func (b *Body) Lineage() Lineage     { return b.lineage }
func (b *Body) ReferenceKey() []byte { return referenceKey(KindBody, b.lineage) }

// IsSolid reports whether the body is a solid (vs a surface body).
func (b *Body) IsSolid() bool { return b.solid }

// Shells returns the body's shells.
func (b *Body) Shells() []*Shell { return append([]*Shell(nil), b.shells...) }

// Faces returns every face in the body, across all shells. The result is a fresh slice the
// caller may keep; once the body is finalized it is a copy of the cached list.
func (b *Body) Faces() []*Face {
	if b.derived {
		return append([]*Face(nil), b.cachedFaces...)
	}
	return b.deriveFaces()
}

// Edges returns every distinct edge in the body.
func (b *Body) Edges() []*Edge {
	if b.derived {
		return append([]*Edge(nil), b.cachedEdges...)
	}
	return deriveEdgesFrom(b.deriveFaces())
}

// Vertices returns every distinct vertex in the body.
func (b *Body) Vertices() []*Vertex {
	if b.derived {
		return append([]*Vertex(nil), b.cachedVertices...)
	}
	return deriveVerticesFrom(deriveEdgesFrom(b.deriveFaces()))
}

// finalizeDerived precomputes the distinct face/edge/vertex lists once the body's shells are
// final, so later queries return a cached copy instead of rebuilding a dedup map every call. It
// must be the last step of every construction path; reads after it are race-free because the
// lists are written here once and never mutated.
func (b *Body) finalizeDerived() {
	b.cachedFaces = b.deriveFaces()
	for _, f := range b.cachedFaces {
		f.finalizeDerived()
	}
	b.cachedEdges = deriveEdgesFrom(b.cachedFaces)
	b.cachedVertices = deriveVerticesFrom(b.cachedEdges)
	b.derived = true
}

// deriveFaces collects the faces across all shells in shell order (the live, uncached form).
func (b *Body) deriveFaces() []*Face {
	var out []*Face
	for _, s := range b.shells {
		out = append(out, s.faces...)
	}
	return out
}

// deriveEdgesFrom returns the distinct edges of the given faces, in first-seen order.
func deriveEdgesFrom(faces []*Face) []*Edge {
	seen := map[*Edge]bool{}
	var out []*Edge
	for _, f := range faces {
		for _, e := range f.Edges() {
			if !seen[e] {
				seen[e] = true
				out = append(out, e)
			}
		}
	}
	return out
}

// deriveVerticesFrom returns the distinct vertices of the given edges, in first-seen order.
func deriveVerticesFrom(edges []*Edge) []*Vertex {
	seen := map[*Vertex]bool{}
	var out []*Vertex
	for _, e := range edges {
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
	box = extendBoxByEdges(box, b.Edges())
	return extendBoxByBoundarylessFaces(box, b.Faces())
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

// BodyFromShells builds a body owning exactly the given shells, re-parenting each. It
// rebuilds a body from a subset of another body's shells — e.g. dropping the inner
// void shells to fill internal cavities (ops.FillInternalVoids, M11-F06 shrinkwrap).
// The donor shells must not be reused afterward, as their owning body is rewritten.
//
// Example: solid := BodyFromShells(b.Lineage(), b.IsSolid(), outerShellsOf(b)...)
func BodyFromShells(lineage Lineage, solid bool, shells ...*Shell) *Body {
	body := &Body{id: nextID(), solid: solid, lineage: lineage}
	for _, sh := range shells {
		sh.body = body
		body.shells = append(body.shells, sh)
	}
	body.finalizeDerived()
	return body
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
	merged.finalizeDerived()
	return merged
}
