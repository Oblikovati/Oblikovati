// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Minimum distance (M18-F01 PBI-164, #428): the closest approach between two model entities of any
// kind. Each entity reduces to tessellated simplices — a vertex to a zero-length segment, an edge
// to its polyline segments, a face to its mesh triangles — and the minimum distance is the smallest
// over all simplex pairs (see kernel/ops mindistance). Result is millimetres; zero when the
// entities touch or intersect.

// MeasureEntity is a resolved model entity (exactly one of Vertex/Edge/Face) for distance queries.
type MeasureEntity struct {
	Vertex *topo.Vertex
	Edge   *topo.Edge
	Face   *topo.Face
}

// segment is a line segment, triangle a mesh triangle — both in database units (centimetres).
type segment struct{ a, b math.Point3 }
type triangle struct{ a, b, c math.Point3 }

// MinDistanceMm returns the minimum distance in millimetres between two entities.
func MinDistanceMm(a, b MeasureEntity, q ops.Quality) float64 {
	segA, triA := a.simplices(q)
	segB, triB := b.simplices(q)
	best := stdmath.Inf(1)
	best = minSegSeg(best, segA, segB)
	best = minSegTri(best, segA, triB)
	best = minSegTri(best, segB, triA)
	best = minTriTri(best, triA, triB)
	return best * cmToMM
}

// simplices returns the entity's segments and triangles in database units.
func (e MeasureEntity) simplices(q ops.Quality) ([]segment, []triangle) {
	switch {
	case e.Vertex != nil:
		p := e.Vertex.Point()
		return []segment{{p, p}}, nil
	case e.Edge != nil:
		return edgeSegments(e.Edge, q), nil
	case e.Face != nil:
		return nil, faceTriangles(e.Face, q)
	}
	return nil, nil
}

func edgeSegments(e *topo.Edge, q ops.Quality) []segment {
	pts := ops.TessellateEdge(e, q)
	segs := make([]segment, 0, len(pts))
	for i := 1; i < len(pts); i++ {
		segs = append(segs, segment{pts[i-1], pts[i]})
	}
	return segs
}

func faceTriangles(f *topo.Face, q ops.Quality) []triangle {
	mesh := ops.TessellateFace(f, q)
	tris := make([]triangle, 0, mesh.TriangleCount())
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		tris = append(tris, triangle{
			mesh.Positions[mesh.Indices[t]], mesh.Positions[mesh.Indices[t+1]], mesh.Positions[mesh.Indices[t+2]]})
	}
	return tris
}

func minSegSeg(best float64, as, bs []segment) float64 {
	for _, x := range as {
		for _, y := range bs {
			best = stdmath.Min(best, ops.SegmentSegmentDistance(x.a, x.b, y.a, y.b))
		}
	}
	return best
}

func minSegTri(best float64, segs []segment, tris []triangle) float64 {
	for _, s := range segs {
		for _, t := range tris {
			best = stdmath.Min(best, ops.SegmentTriangleDistance(s.a, s.b, t.a, t.b, t.c))
		}
	}
	return best
}

func minTriTri(best float64, as, bs []triangle) float64 {
	for _, x := range as {
		for _, y := range bs {
			best = stdmath.Min(best, ops.TriangleTriangleDistance(x.a, x.b, x.c, y.a, y.b, y.c))
		}
	}
	return best
}
