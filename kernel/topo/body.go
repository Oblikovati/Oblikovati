// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"sync"

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

	// Reference-key indices, memoized lazily on the first lookup of each kind (#1580).
	// The body is immutable after finalizeDerived — a recompute builds a NEW body, never
	// mutates this one — so an index is a pure function of the cached entity lists: built
	// once, never invalidated, and read lock-free like those lists. A body that is never
	// keyed (most intermediate recompute results) pays nothing; a pre-finalize lookup
	// falls back to a linear scan rather than memoizing an incomplete index.
	edgeIndexOnce   sync.Once
	faceIndexOnce   sync.Once
	vertexIndexOnce sync.Once
	edgeIndex       map[string][]*Edge
	faceIndex       map[string][]*Face
	vertexIndex     map[string][]*Vertex
}

// edgeKeyIndex returns the edge reference-key index, building it once. See the index
// fields on [Body] for why a build-once memo is correct (the body is immutable here).
func (b *Body) edgeKeyIndex() map[string][]*Edge {
	b.edgeIndexOnce.Do(func() { b.edgeIndex = buildKeyIndex(b.cachedEdges) })
	return b.edgeIndex
}

// faceKeyIndex returns the face reference-key index, building it once.
func (b *Body) faceKeyIndex() map[string][]*Face {
	b.faceIndexOnce.Do(func() { b.faceIndex = buildKeyIndex(b.cachedFaces) })
	return b.faceIndex
}

// vertexKeyIndex returns the vertex reference-key index, building it once.
func (b *Body) vertexKeyIndex() map[string][]*Vertex {
	b.vertexIndexOnce.Do(func() { b.vertexIndex = buildKeyIndex(b.cachedVertices) })
	return b.vertexIndex
}

func (b *Body) ID() uint64           { return b.id }
func (b *Body) Kind() EntityKind     { return KindBody }
func (b *Body) Lineage() Lineage     { return b.lineage }
func (b *Body) ReferenceKey() []byte { return referenceKey(KindBody, b.lineage) }

// IsSolid reports whether the body is a solid (vs a surface body).
func (b *Body) IsSolid() bool { return b.solid }

// Shells returns the body's shells.
func (b *Body) Shells() []*Shell { return append([]*Shell(nil), b.shells...) }

// EulerCharacteristic returns the surface χ = V − E + 2F − L (the Euler–Poincaré form, correct
// across B-rep seams and holed faces). Shared by the validator and the boolean's tangent-result
// gate so both read χ the same way (#1600).
func (b *Body) EulerCharacteristic() int {
	loops := 0
	for _, f := range b.Faces() {
		loops += len(f.Loops())
	}
	return len(b.Vertices()) - len(b.Edges()) + 2*len(b.Faces()) - loops
}

// EulerAdmissible reports whether a CLOSED solid's χ is topologically possible: even and at most
// 2 per shell (χ = Σ over shells of 2 − 2·genusₛ, genus ≥ 0). A non-solid body is unconstrained
// and reports true. An odd or too-large χ is a pinch defect the per-edge manifold checks miss.
func (b *Body) EulerAdmissible() bool {
	if !b.solid {
		return true
	}
	chi := b.EulerCharacteristic()
	return chi%2 == 0 && chi <= 2*len(b.shells)
}

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
	if m := b.FacesByKey(key); len(m) > 0 {
		return m[0], true
	}
	return nil, false
}

// FindEdgeByKey re-binds an edge reference key by lineage.
func (b *Body) FindEdgeByKey(key []byte) (*Edge, bool) {
	if m := b.EdgesByKey(key); len(m) > 0 {
		return m[0], true
	}
	return nil, false
}

// EdgesByKey returns EVERY edge whose reference key matches — normally one. More than one means a
// topological-naming collision (two distinct edges minted the same lineage), the wrong-rebind
// hazard ADR-0043's resolution guard turns into an honest error instead of a silent first-match.
// The result is a fresh slice the caller may keep or mutate; it never aliases the index.
func (b *Body) EdgesByKey(key []byte) []*Edge {
	if !b.derived {
		return scanByKey(b.Edges(), key)
	}
	return append([]*Edge(nil), b.edgeKeyIndex()[string(key)]...)
}

// FacesByKey returns EVERY face whose reference key matches — the face counterpart of [EdgesByKey].
func (b *Body) FacesByKey(key []byte) []*Face {
	if !b.derived {
		return scanByKey(b.Faces(), key)
	}
	return append([]*Face(nil), b.faceKeyIndex()[string(key)]...)
}

// FindVertexByKey re-binds a vertex reference key by lineage — used to resolve a picked
// B-rep vertex as a work-feature point input after the body is rebuilt.
func (b *Body) FindVertexByKey(key []byte) (*Vertex, bool) {
	if !b.derived {
		if v := scanByKey(b.Vertices(), key); len(v) > 0 {
			return v[0], true
		}
		return nil, false
	}
	if m := b.vertexKeyIndex()[string(key)]; len(m) > 0 {
		return m[0], true
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
