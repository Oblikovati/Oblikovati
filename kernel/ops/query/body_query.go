// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Body locate/ray queries (M07-F07, Oblikovati/Oblikovati#630): find the
// topology entity nearest a 3D point, and fire a ray (with a pick radius)
// into the body. Face distances and ray pierces are resolved against the
// ANALYTIC B-rep surfaces (not a face tessellation), so a pick →
// reference-key resolution is exact and Quality-independent (M48/C3). Edge and
// vertex proximity still use the edge's curve discretization, a curve
// discretizer — not a face mesh.

// LocatedEntity is one query answer: which entity, of which kind, how far.
type LocatedEntity struct {
	Kind     topo.EntityKind
	Vertex   *topo.Vertex
	Edge     *topo.Edge
	Face     *topo.Face
	Point    math.Point3 // the closest/hit point on the entity
	Distance float64     // from the query point, or along the ray
}

// LocateUsingPoint returns the entity of the requested kind nearest p within
// the proximity tolerance. kind 0 matches vertices, edges and faces, nearest
// of any.
//
// Example: hit, ok := query.LocateUsingPoint(b, 0, p, 1e-3, query.DefaultQuality())
func LocateUsingPoint(b *topo.Body, kind topo.EntityKind, p math.Point3, proximityTol float64, q Quality) (LocatedEntity, bool) {
	best := LocatedEntity{Distance: stdmath.Inf(1)}
	if kind == 0 || kind == topo.KindVertex {
		closerVertex(&best, b, p)
	}
	if kind == 0 || kind == topo.KindEdge {
		closerEdge(&best, b, p, q)
	}
	if kind == 0 || kind == topo.KindFace {
		closerFace(&best, b, p, q)
	}
	return best, best.Distance <= proximityTol
}

func closerVertex(best *LocatedEntity, b *topo.Body, p math.Point3) {
	for _, v := range b.Vertices() {
		if d := float64(p.DistanceTo(v.Point())); d < best.Distance {
			*best = LocatedEntity{Kind: topo.KindVertex, Vertex: v, Point: v.Point(), Distance: d}
		}
	}
}

func closerEdge(best *LocatedEntity, b *topo.Body, p math.Point3, q Quality) {
	for _, e := range b.Edges() {
		pl := tessellate.DiscretizeEdge(e, q)
		for i := 0; i+1 < len(pl); i++ {
			cp := ClosestPointOnSegment(p, pl[i], pl[i+1])
			if d := float64(p.DistanceTo(cp)); d < best.Distance {
				*best = LocatedEntity{Kind: topo.KindEdge, Edge: e, Point: cp, Distance: d}
			}
		}
	}
}

// closerFace updates best with the nearest face, measuring the ANALYTIC perpendicular distance from
// p to each face (geom.ClosestPointOnSurface confirmed against the trim), not a per-triangle mesh
// distance — so the located point is the exact foot on the B-rep and does not drift with the display
// Quality (M48/C3, Oblikovati/Oblikovati#3469).
func closerFace(best *LocatedEntity, b *topo.Body, p math.Point3, q Quality) {
	for _, f := range b.Faces() {
		if cp, d := analyticClosestOnFace(f, p, q); d < best.Distance {
			*best = LocatedEntity{Kind: topo.KindFace, Face: f, Point: cp, Distance: d}
		}
	}
}

// ClosestPointOnSegment clamps p's projection onto segment ab.
func ClosestPointOnSegment(p, a, b math.Point3) math.Point3 {
	ab := a.VectorTo(b)
	denom := float64(ab.LengthSquared())
	if denom == 0 {
		return a
	}
	t := float64(a.VectorTo(p).Dot(ab)) / denom
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return a.TranslateBy(ab.Scale(math.Scalar(t)))
}

// FindUsingRay fires a ray into the body and returns every hit, nearest first:
// faces where the ray crosses their mesh, plus edges and vertices passing
// within radius of the ray. findFirstOnly truncates to the nearest hit.
//
// Example: hits := query.FindUsingRay(b, origin, dir, 0.05, query.DefaultQuality(), false)
func FindUsingRay(b *topo.Body, origin math.Point3, dir math.Vector3, radius float64, q Quality, findFirstOnly bool) []LocatedEntity {
	l := float64(dir.Length())
	if l == 0 {
		return nil
	}
	d := dir.Scale(math.Scalar(1 / l))
	var hits []LocatedEntity
	hits = append(hits, rayFaceHits(b, origin, d)...)
	if radius > 0 {
		hits = append(hits, rayEdgeHits(b, origin, d, radius, q)...)
		hits = append(hits, rayVertexHits(b, origin, d, radius)...)
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Distance < hits[j].Distance })
	if findFirstOnly && len(hits) > 1 {
		hits = hits[:1]
	}
	return hits
}

// rayFaceHits collects every face the ray crosses, each at its nearest analytic pierce
// (analyticFaceRayHit: exact ray∩surface confirmed against the trim), not a per-triangle mesh
// crossing — so the reported hit point is on the exact B-rep and is Quality-independent (M48/C3,
// Oblikovati/Oblikovati#3470). dir is unit (FindUsingRay normalizes it), so Distance is the true
// distance along the ray.
func rayFaceHits(b *topo.Body, origin math.Point3, dir math.Vector3) []LocatedEntity {
	var out []LocatedEntity
	for _, f := range b.Faces() {
		if t, p, ok := analyticFaceRayHit(f, origin, dir); ok {
			out = append(out, LocatedEntity{Kind: topo.KindFace, Face: f, Point: p, Distance: t})
		}
	}
	return out
}

func rayEdgeHits(b *topo.Body, origin math.Point3, dir math.Vector3, radius float64, q Quality) []LocatedEntity {
	var out []LocatedEntity
	for _, e := range b.Edges() {
		if t, p, ok := rayPolylineApproach(origin, dir, tessellate.DiscretizeEdge(e, q), radius); ok {
			out = append(out, LocatedEntity{Kind: topo.KindEdge, Edge: e, Point: p, Distance: t})
		}
	}
	return out
}

func rayVertexHits(b *topo.Body, origin math.Point3, dir math.Vector3, radius float64) []LocatedEntity {
	var out []LocatedEntity
	for _, v := range b.Vertices() {
		t := float64(origin.VectorTo(v.Point()).Dot(dir))
		if t < 0 {
			continue
		}
		onRay := origin.TranslateBy(dir.Scale(math.Scalar(t)))
		if float64(onRay.DistanceTo(v.Point())) <= radius {
			out = append(out, LocatedEntity{Kind: topo.KindVertex, Vertex: v, Point: v.Point(), Distance: t})
		}
	}
	return out
}

// rayPolylineApproach finds the polyline's nearest pass to the ray within
// radius; returns the ray parameter and the polyline point.
func rayPolylineApproach(origin math.Point3, dir math.Vector3, pl []math.Point3, radius float64) (float64, math.Point3, bool) {
	bestT, bestP, found := stdmath.Inf(1), math.Point3{}, false
	for i := 0; i+1 < len(pl); i++ {
		t, p, d := raySegmentApproach(origin, dir, pl[i], pl[i+1])
		if d <= radius && t >= 0 && t < bestT {
			bestT, bestP, found = t, p, true
		}
	}
	return bestT, bestP, found
}

// raySegmentApproach returns the ray parameter, segment point and distance of
// the closest approach between the ray and segment ab (clamped to the segment).
func raySegmentApproach(origin math.Point3, dir math.Vector3, a, b math.Point3) (float64, math.Point3, float64) {
	// Closest point pair between infinite ray (origin + t·dir) and segment
	// (a + s·ab): solve the 2×2 normal equations, clamp s to [0,1], re-derive t.
	ab := a.VectorTo(b)
	w := origin.VectorTo(a)
	aDD := float64(dir.Dot(dir))
	bDA := float64(dir.Dot(ab))
	cAA := float64(ab.Dot(ab))
	dDW := float64(dir.Dot(w))
	eAW := float64(ab.Dot(w))
	denom := aDD*cAA - bDA*bDA
	s := 0.0
	if denom > 1e-15 {
		s = math.Clamp01((bDA*dDW - aDD*eAW) / denom)
	}
	onSeg := a.TranslateBy(ab.Scale(math.Scalar(s)))
	t := float64(origin.VectorTo(onSeg).Dot(dir)) / aDD
	if t < 0 {
		t = 0
	}
	onRay := origin.TranslateBy(dir.Scale(math.Scalar(t)))
	return t, onSeg, float64(onRay.DistanceTo(onSeg))
}
