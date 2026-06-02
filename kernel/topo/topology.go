// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/math"
)

// Vertex is a 0-dimensional topological entity: a point with identity.
type Vertex struct {
	id      uint64
	point   math.Point3
	lineage Lineage
	edges   []*Edge
}

// ID/Kind/Lineage/ReferenceKey are the identity surface every entity shares.
func (v *Vertex) ID() uint64           { return v.id }
func (v *Vertex) Kind() EntityKind     { return KindVertex }
func (v *Vertex) Lineage() Lineage     { return v.lineage }
func (v *Vertex) ReferenceKey() []byte { return referenceKey(KindVertex, v.lineage) }

// Point returns the vertex position.
func (v *Vertex) Point() math.Point3 { return v.point }

// Edges returns the edges incident to the vertex.
func (v *Vertex) Edges() []*Edge { return append([]*Edge(nil), v.edges...) }

// RangeBox returns the (degenerate) bounding box of the vertex.
func (v *Vertex) RangeBox() math.Box { return math.BoxFromPoints(v.point) }

// Edge is a 1-dimensional entity bounded by two vertices, carrying its curve
// geometry and the oriented uses that bind it into loops.
type Edge struct {
	id      uint64
	curve   geom.Curve3
	start   *Vertex
	end     *Vertex
	uses    []*EdgeUse
	lineage Lineage
}

func (e *Edge) ID() uint64           { return e.id }
func (e *Edge) Kind() EntityKind     { return KindEdge }
func (e *Edge) Lineage() Lineage     { return e.lineage }
func (e *Edge) ReferenceKey() []byte { return referenceKey(KindEdge, e.lineage) }

// Geometry returns the edge's underlying transient curve (a Circle/Arc3d/Line…).
func (e *Edge) Geometry() geom.Curve3 { return e.curve }

// StartVertex and EndVertex return the bounding vertices.
func (e *Edge) StartVertex() *Vertex { return e.start }
func (e *Edge) EndVertex() *Vertex   { return e.end }

// Vertices returns the edge's two bounding vertices.
func (e *Edge) Vertices() []*Vertex { return []*Vertex{e.start, e.end} }

// Uses returns the oriented uses binding this edge into loops. A manifold solid
// edge has exactly two; a boundary (open) edge has one.
func (e *Edge) Uses() []*EdgeUse { return append([]*EdgeUse(nil), e.uses...) }

// Faces returns the distinct faces this edge bounds (via its loop uses).
func (e *Edge) Faces() []*Face {
	seen := map[*Face]bool{}
	var out []*Face
	for _, u := range e.uses {
		f := u.loop.face
		if !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}

// RangeBox returns the bounding box of the edge's endpoints.
func (e *Edge) RangeBox() math.Box { return math.BoxFromPoints(e.start.point, e.end.point) }

// EdgeUse is an oriented use of an [Edge] within a [Loop]; reversed when the loop
// traverses the edge against its natural start→end direction (a half-edge / coedge).
type EdgeUse struct {
	edge     *Edge
	loop     *Loop
	reversed bool
}

// Edge returns the used edge; Reversed reports traversal direction; Loop the owner.
func (u *EdgeUse) Edge() *Edge    { return u.edge }
func (u *EdgeUse) Reversed() bool { return u.reversed }
func (u *EdgeUse) Loop() *Loop    { return u.loop }

// Loop (EdgeLoop) is an ordered cycle of edge-uses bounding a face — the outer
// boundary or an inner hole.
type Loop struct {
	id    uint64
	face  *Face
	uses  []*EdgeUse
	outer bool
}

func (l *Loop) ID() uint64       { return l.id }
func (l *Loop) Kind() EntityKind { return KindLoop }

// IsOuter reports whether this is the face's outer boundary (vs an inner hole).
func (l *Loop) IsOuter() bool { return l.outer }

// EdgeUses returns the loop's ordered oriented edge uses.
func (l *Loop) EdgeUses() []*EdgeUse { return append([]*EdgeUse(nil), l.uses...) }

// Face returns the loop's owning face.
func (l *Loop) Face() *Face { return l.face }

// Face is a 2-dimensional entity: a bounded region of a surface, bounded by loops.
type Face struct {
	id      uint64
	surface geom.Surface
	loops   []*Loop
	shell   *Shell
	lineage Lineage
}

func (f *Face) ID() uint64           { return f.id }
func (f *Face) Kind() EntityKind     { return KindFace }
func (f *Face) Lineage() Lineage     { return f.lineage }
func (f *Face) ReferenceKey() []byte { return referenceKey(KindFace, f.lineage) }

// Geometry returns the face's underlying transient surface (a Plane/Cylinder…).
func (f *Face) Geometry() geom.Surface { return f.surface }

// Loops returns the face's boundary loops (outer first by construction).
func (f *Face) Loops() []*Loop { return append([]*Loop(nil), f.loops...) }

// Edges returns the distinct edges bounding the face.
func (f *Face) Edges() []*Edge {
	seen := map[*Edge]bool{}
	var out []*Edge
	for _, l := range f.loops {
		for _, u := range l.uses {
			if !seen[u.edge] {
				seen[u.edge] = true
				out = append(out, u.edge)
			}
		}
	}
	return out
}

// Vertices returns the distinct vertices bounding the face.
func (f *Face) Vertices() []*Vertex {
	seen := map[*Vertex]bool{}
	var out []*Vertex
	for _, e := range f.Edges() {
		for _, v := range e.Vertices() {
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	return out
}

// RangeBox returns the bounding box of the face's vertices.
func (f *Face) RangeBox() math.Box {
	box := math.EmptyBox()
	for _, v := range f.Vertices() {
		box = box.ExtendPoint(v.point)
	}
	return box
}

// Shell is a connected set of faces; a closed shell bounds a solid region.
type Shell struct {
	id     uint64
	faces  []*Face
	body   *Body
	closed bool
}

func (s *Shell) ID() uint64       { return s.id }
func (s *Shell) Kind() EntityKind { return KindShell }

// Faces returns the shell's faces; IsClosed reports whether it bounds a solid.
func (s *Shell) Faces() []*Face { return append([]*Face(nil), s.faces...) }
func (s *Shell) IsClosed() bool { return s.closed }
func (s *Shell) Body() *Body    { return s.body }

// referenceKey prefixes a lineage key with its entity kind, so keys of different
// kinds never collide even with identical lineage.
func referenceKey(kind EntityKind, lineage Lineage) []byte {
	return append([]byte{byte(kind)}, lineage.Key()...)
}
