// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Analytic projection of a 3D curve onto a plane (ADR-0055). Projecting a model edge into a
// sketch must keep the curve ANALYTIC — a projected circle is a circle, an arc an arc, a line a
// line — not a sampled polyline that bloats files, breaks offset, and facets every downstream
// solid. This is the single canonical projection the sketch (and any other projector) shares, so
// the old analytic→polyline→analytic round-trip (sample the edge, then re-fit a shape from the
// points) is deleted. It dispatches on the curve's concrete type; the projection onto a plane is an
// affine map, so each class is closed under it (line→line, circle→circle) and only the defining
// data transforms (Piegl & Tiller, affine invariance of control points).

// planeParallelTol bounds how close a curve's plane normal must be to the target plane normal for
// the projection to stay a circle/arc rather than an ellipse. |cos angle| within this of 1.
const planeParallelTol = 1e-9

// ProjectCurveToPlane projects c orthographically onto pl (along pl's normal) and returns the
// analytic 2D curve in pl's (u,v) frame, or ok=false when the type is not yet handled analytically
// (an oblique circle/arc — which projects to an ellipse, ADR-0055 phase 2 — a spline, or any curve
// whose plane is not parallel to pl). A false return tells the caller to fall back to the sampled
// polyline. A LINE always projects to a line; a CIRCLE/ARC whose plane is parallel to pl projects
// exactly (orthographic projection preserves in-plane distances), so the radius is unchanged.
func ProjectCurveToPlane(pl Plane, c Curve3) (Curve2, bool) {
	switch k := c.(type) {
	case LineSegment:
		return NewLineSegment2d(planeUV(pl, k.StartPoint), planeUV(pl, k.EndPoint)), true
	case Line:
		return projectInfiniteLine(pl, k)
	case Circle:
		if !normalParallelToPlane(k.Normal, pl) {
			return nil, false // oblique circle → ellipse (phase 2)
		}
		return NewCircle2d(planeUV(pl, k.Center), k.Radius), true
	case Arc3d:
		if !normalParallelToPlane(k.Normal, pl) {
			return nil, false // oblique arc → elliptical arc (phase 2)
		}
		lo, hi := k.Domain()
		arc, err := Arc2dByThreePoints(
			planeUV(pl, k.PointAt(lo)), planeUV(pl, k.PointAt((lo+hi)/2)), planeUV(pl, k.PointAt(hi)))
		if err != nil {
			return nil, false
		}
		return arc, true
	default:
		return nil, false
	}
}

// projectInfiniteLine projects an unbounded line onto pl through two of its points.
func projectInfiniteLine(pl Plane, l Line) (Curve2, bool) {
	a := planeUV(pl, l.Origin)
	b := planeUV(pl, l.Origin.TranslateBy(l.Dir.AsVector()))
	if a.DistanceTo(b) < planeParallelTol {
		return nil, false // line parallel to the projection direction → projects to a point
	}
	line2d, err := NewLine2d(a, math.V2(b.X-a.X, b.Y-a.Y))
	if err != nil {
		return nil, false
	}
	return line2d, true
}

// planeUV returns p's coordinates in pl's (u,v) frame — the orthographic projection of p onto pl,
// u = (p−Origin)·UAxis, v = (p−Origin)·VAxis.
func planeUV(pl Plane, p math.Point3) math.Point2 {
	d := pl.Origin.VectorTo(p)
	return math.P2(d.Dot(pl.UAxis.AsVector()), d.Dot(pl.VAxis.AsVector()))
}

// normalParallelToPlane reports whether a curve's plane (given its unit normal) is parallel to pl,
// so a circle/arc on it projects to a circle/arc rather than an ellipse.
func normalParallelToPlane(n math.UnitVector3, pl Plane) bool {
	return stdmath.Abs(float64(n.AsVector().Dot(pl.Normal()))) > 1-planeParallelTol
}

// SampleCurve3 walks c's parameter domain into n+1 evenly spaced points — the one shared fixed-count
// 3D curve sampler (ADR-0055), replacing the per-call copies in the projection/reference sources. It
// is the polyline FALLBACK for a curve with no analytic plane projection; analytic curves go through
// ProjectCurveToPlane instead. For adaptive, tolerance-driven tessellation use the ops tessellator,
// not this.
func SampleCurve3(c Curve3, n int) []math.Point3 {
	if n < 1 {
		n = 1
	}
	lo, hi := c.Domain()
	pts := make([]math.Point3, n+1)
	for i := range pts {
		pts[i] = c.PointAt(lo + (hi-lo)*float64(i)/float64(n))
	}
	return pts
}
