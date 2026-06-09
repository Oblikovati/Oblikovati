// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// curveSamplesPerEdge is how many points we evaluate along each edge's curve
// when bounding a face or body. A curved edge is not bounded by its endpoints —
// a full cylinder cap is a circle with a single seam vertex, so vertex-only
// bounds collapse to that one point and report a wildly wrong (degenerate)
// RangeBox. That silently broke boolean classification: a cylinder's box failed
// to intersect a clearly-overlapping tool, so Cut returned the uncut target.
// Sampling each edge across its Domain recovers the true silhouette extent; 32
// samples land exactly on an axis-aligned circle's extrema and stay tight for
// tilted ones. A conservative (slightly large) box is safe for broad-phase
// classification; an undersized one is not.
const curveSamplesPerEdge = 32

// extendBoxByEdges grows box to enclose sampled points along every edge's curve.
func extendBoxByEdges(box math.Box, edges []*Edge) math.Box {
	for _, e := range edges {
		lo, hi := e.curve.Domain()
		if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
			box = box.ExtendPoint(e.start.point).ExtendPoint(e.end.point)
			continue
		}
		for i := 0; i <= curveSamplesPerEdge; i++ {
			t := lo + (hi-lo)*float64(i)/float64(curveSamplesPerEdge)
			box = box.ExtendPoint(e.curve.PointAt(t))
		}
	}
	return box
}

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
	id        uint64
	curve     geom.Curve3
	start     *Vertex
	end       *Vertex
	uses      []*EdgeUse
	lineage   Lineage
	snapped   []math.Point3
	tolerance float64
}

func (e *Edge) ID() uint64           { return e.id }
func (e *Edge) Kind() EntityKind     { return KindEdge }
func (e *Edge) Lineage() Lineage     { return e.lineage }
func (e *Edge) ReferenceKey() []byte { return referenceKey(KindEdge, e.lineage) }

// Geometry returns the edge's underlying transient curve (a Circle/Arc3d/Line…).
func (e *Edge) Geometry() geom.Curve3 { return e.curve }

// SnappedCurve returns the edge's healed, on-surface discretization (import healing, M25 PBI-324),
// or nil for a native edge sampled directly from its curve. Both faces of the edge share this exact
// polyline, so their tessellation boundaries are identical. Returned directly; callers must not mutate.
func (e *Edge) SnappedCurve() []math.Point3 { return e.snapped }

// SetSnappedCurve stores the healed polyline and the residual (the max distance the original imported
// edge sat off its adjacent surfaces) — see [Edge.SnappedCurve]. Pass nil to clear (revert to native).
func (e *Edge) SetSnappedCurve(polyline []math.Point3, residual float64) {
	e.snapped, e.tolerance = polyline, residual
}

// Tolerance returns the edge's recorded healing residual in model units (0 for a native/clean edge
// whose curve already lies on its surfaces).
func (e *Edge) Tolerance() float64 { return e.tolerance }

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
func (e *Edge) RangeBox() math.Box {
	return extendBoxByEdges(math.BoxFromPoints(e.start.point, e.end.point), []*Edge{e})
}

// EdgeUse is an oriented use of an [Edge] within a [Loop]; reversed when the loop
// traverses the edge against its natural start→end direction (a half-edge / coedge).
type EdgeUse struct {
	edge     *Edge
	loop     *Loop
	reversed bool
	pcurve   []math.Point2
}

// Edge returns the used edge; Reversed reports traversal direction; Loop the owner.
func (u *EdgeUse) Edge() *Edge    { return u.edge }
func (u *EdgeUse) Reversed() bool { return u.reversed }
func (u *EdgeUse) Loop() *Loop    { return u.loop }

// Pcurve returns this use's PCURVE: the edge's (u, v) curve on the loop's face surface, in the use's
// traversal order. It is nil until import healing reconstructs it (M25) — SolidWorks STEP omits
// pcurves — and, once present, gives the face's trim boundary exactly in parameter space (what the
// NURBS mesher needs). Returned directly (not copied); callers must not mutate it.
func (u *EdgeUse) Pcurve() []math.Point2 { return u.pcurve }

// SetPcurve attaches the reconstructed pcurve (see [EdgeUse.Pcurve]).
func (u *EdgeUse) SetPcurve(pc []math.Point2) { u.pcurve = pc }

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
	id       uint64
	surface  geom.Surface
	loops    []*Loop
	shell    *Shell
	lineage  Lineage
	reversed bool
}

func (f *Face) ID() uint64           { return f.id }
func (f *Face) Kind() EntityKind     { return KindFace }
func (f *Face) Lineage() Lineage     { return f.lineage }
func (f *Face) ReferenceKey() []byte { return referenceKey(KindFace, f.lineage) }

// Geometry returns the face's underlying transient surface (a Plane/Cylinder…).
func (f *Face) Geometry() geom.Surface { return f.surface }

// Reversed reports whether the face's outward (material) side is OPPOSITE its surface
// normal — true for the cut wall a Difference carves, where the surface (e.g. a cylinder's
// outward-radial normal) points into the removed material. Tessellation and mass-properties
// negate the surface normal for such faces. Most faces (sense agrees with surface) are false.
func (f *Face) Reversed() bool { return f.reversed }

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

// RangeBox returns the face's bounding box, accounting for curved edges (a
// cylindrical or conical face's circular edges bulge well past their vertices).
func (f *Face) RangeBox() math.Box {
	box := math.EmptyBox()
	for _, v := range f.Vertices() {
		box = box.ExtendPoint(v.point)
	}
	return extendBoxByEdges(box, f.Edges())
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
