// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The points on a body's CURVED boundary that its vertices do not account for.
//
// A body's vertices are a support set for it only where its faces are planes: the convex hull of the
// vertices of a polyhedron contains the polyhedron, so the vertices' extent along any direction IS
// the body's. A curved face breaks that — brep.SolidCylinder has exactly TWO vertices, both on its
// seam, so the vertex extent across its axis is zero — which is the same hole [Body.RangeBox] fills
// by sampling edges and boundary-less or charted faces (see extendBoxByEdges and the note on
// extendBoxByChartedFaces).
//
// This samples the same curved half, per face, so a caller that needs the body's extent along a
// direction OTHER than a world axis can measure it the way the box does (#3524). It shares the box's
// sampling grids (curveSamplesPerEdge, faceSamplesPerAxis) and its chartWindow, and it is deliberately
// NOT the box rewritten on top of it: the box also sweeps PLANAR charted faces, which a width does not
// need and whose sweep would only widen it. Sampling is what the box already does and what this keeps
// doing: it is a bound for culling and for the model-relative resolution, never a modelling decision.
// query.PreciseRangeBox is the certified-tight answer where exactness is the point.

// AppendSupportPoints appends sampled points on this face that the body's VERTICES do not already
// account for: every edge of the face, and the face's own surface where its edges do not bound it (it
// carries a chart, or no boundary loop at all).
//
// Only a curved face needs it — a planar face is bounded by its own vertices — so callers that have
// already asked whether a face is planar skip it, and a wholly faceted body pays nothing.
//
// Together with [Body.Vertices] these are a support set for the body: the extent of the two sets
// along any direction bounds the body along that direction, to the accuracy of the sampling.
//
// Example:
//
//	if _, planar := geom.PlanarNormal(f.Geometry()); !planar { support = f.AppendSupportPoints(support) }
func (f *Face) AppendSupportPoints(pts []math.Point3) []math.Point3 {
	pts = appendFaceCurveSamples(pts, f)
	return appendFaceSurfaceSamples(pts, f)
}

// appendFaceCurveSamples samples every edge of one face across its curve's domain, the way
// extendBoxByEdges bounds a curved edge that its endpoints do not.
func appendFaceCurveSamples(pts []math.Point3, f *Face) []math.Point3 {
	for _, e := range f.Edges() {
		lo, hi := e.curve.Domain()
		if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
			pts = append(pts, e.start.point, e.end.point)
			continue
		}
		for i := 0; i <= curveSamplesPerEdge; i++ {
			pts = append(pts, e.curve.PointAt(lo+(hi-lo)*float64(i)/float64(curveSamplesPerEdge)))
		}
	}
	return pts
}

// appendFaceSurfaceSamples sweeps a face's own surface where its edges do not bound it: over the
// chart window when it carries one, over the whole domain when it has no boundary at all.
func appendFaceSurfaceSamples(pts []math.Point3, f *Face) []math.Point3 {
	u0, u1, v0, v1, ok := chartWindow(f.chart)
	if !ok {
		if len(f.loops) > 0 {
			return pts // bounded by the edges sampled above
		}
		u0, u1 = f.surface.UDomain()
		v0, v1 = f.surface.VDomain()
	}
	if stdmath.IsInf(u0, 0) || stdmath.IsInf(u1, 0) || stdmath.IsInf(v0, 0) || stdmath.IsInf(v1, 0) {
		return pts
	}
	return appendSurfaceGrid(pts, f, u0, u1, v0, v1)
}

// appendSurfaceGrid samples one face's surface over a (u, v) window on the range box's own grid.
func appendSurfaceGrid(pts []math.Point3, f *Face, u0, u1, v0, v1 float64) []math.Point3 {
	for i := 0; i <= faceSamplesPerAxis; i++ {
		for j := 0; j <= faceSamplesPerAxis; j++ {
			u := u0 + (u1-u0)*float64(i)/faceSamplesPerAxis
			v := v0 + (v1-v0)*float64(j)/faceSamplesPerAxis
			pts = append(pts, f.surface.PointAt(u, v))
		}
	}
	return pts
}
