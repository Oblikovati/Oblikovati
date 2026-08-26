// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// intersectSurfacesNear returns the point common to three surfaces nearest a starting estimate — a
// Gauss-Newton solve on the three signed-distance residuals (each surface contributes its normal as
// the gradient), converging from the old vertex position. It is how the draft modifier relocates a
// corner when its faces tilt: exact for three planes, and it handles a plane∩plane∩cylinder corner
// (a filleted edge's endpoint) that the plane-only rebuild could not. ok=false if the normals are
// degenerate (parallel) or fewer than three surfaces meet.
func intersectSurfacesNear(start math.Point3, surfaces []geom.Surface, tol float64) (math.Point3, bool) {
	if len(surfaces) < 3 {
		return start, false
	}
	p := start
	for range 40 {
		a, b, ok := surfaceResidualSystem(p, surfaces)
		if !ok {
			return p, false
		}
		step, ok := solve3(a, b)
		if !ok {
			return p, false
		}
		p = p.TranslateBy(math.V3(step[0], step[1], step[2]))
		if step[0]*step[0]+step[1]*step[1]+step[2]*step[2] < tol*tol {
			return p, true
		}
	}
	return p, true
}

// surfaceResidualSystem builds the 3×3 Newton system n_i·Δ = −d_i from the first three surfaces:
// each row is the unit normal at p's projection, each right-hand side the negated signed distance.
func surfaceResidualSystem(p math.Point3, surfaces []geom.Surface) ([3][3]float64, [3]float64, bool) {
	var a [3][3]float64
	var b [3]float64
	for i := range 3 {
		u, v := surfaces[i].ParamAt(p)
		foot := surfaces[i].PointAt(u, v)
		n := surfaces[i].NormalAt(u, v)
		l := float64(n.Length())
		if l == 0 {
			return a, b, false
		}
		n = n.Scale(math.Scalar(1 / l))
		a[i] = [3]float64{float64(n.X), float64(n.Y), float64(n.Z)}
		b[i] = -float64(foot.VectorTo(p).Dot(n))
	}
	return a, b, true
}

// reintersectEdge recomputes an edge's curve as the intersection of its two (new) support surfaces,
// trimmed to the edge's new endpoints — a straight segment between planes, an arc between a plane
// and a cylinder (circular when perpendicular, elliptical when the drafted plane cuts obliquely).
func (d *draftMod) reintersectEdge(e *topo.Edge, sA, sB geom.Surface, p0, p1 math.Point3) (geom.Curve3, bool) {
	curves, handled := geom.IntersectSurfacesAnalytic(sA, sB, d.res)
	if !handled || len(curves) == 0 {
		return geom.NewLineSegment(p0, p1), true // both planar (a line) or no analytic cross: straight
	}
	return trimCurveToEnds(nearestBranch(curves, edgeMidpoint(e)), p0, p1)
}

// nearestBranch picks the intersection branch whose sampled points pass nearest ref — the branch
// the original edge lay on (an oblique plane∩cylinder yields one ellipse, but a cylinder can meet a
// plane in two lines, so the choice matters).
func nearestBranch(curves []geom.Curve3, ref math.Point3) geom.Curve3 {
	best := curves[0]
	bestD := stdmath.Inf(1)
	for _, c := range curves {
		for i := 0; i <= 8; i++ {
			if d := float64(c.PointAt(float64(i) / 8).DistanceTo(ref)); d < bestD {
				best, bestD = c, d
			}
		}
	}
	return best
}

// trimCurveToEnds bounds a full intersection curve to the arc between the two endpoints, choosing
// the concrete bounded type: a line segment, a circular arc, or an elliptical arc.
func trimCurveToEnds(curve geom.Curve3, p0, p1 math.Point3) (geom.Curve3, bool) {
	switch c := curve.(type) {
	case geom.Line, geom.LineSegment:
		return geom.NewLineSegment(p0, p1), true
	case geom.Circle:
		return circleArcBetween(c, p0, p1)
	case geom.EllipseFull:
		return ellipseArcBetween(c, p0, p1)
	default:
		return geom.NewLineSegment(p0, p1), true // unknown analytic branch: straight best-effort
	}
}

// circleArcBetween returns the arc of circle c from p0 to p1 that stays nearest the chord midpoint
// (the shorter arc for a fillet edge, but chosen geometrically so a reflex edge also works).
func circleArcBetween(c geom.Circle, p0, p1 math.Point3) (geom.Curve3, bool) {
	x, y := c.RefDir.AsVector(), c.Normal.Cross(c.RefDir)
	a0 := angleInPlane(c.Center, x, y, p0)
	a1 := angleInPlane(c.Center, x, y, p1)
	start, sweep := pickArc(a0, a1, p0.Midpoint(p1), func(s, sw float64) math.Point3 {
		return pointOnCircleAt(c, s+0.5*sw)
	})
	arc, err := geom.NewArc3d(c.Center, c.Normal.AsVector(), x, c.Radius, start, sweep)
	return arc, err == nil
}

// ellipseArcBetween returns the arc of ellipse c from p0 to p1 nearest the chord midpoint.
func ellipseArcBetween(c geom.EllipseFull, p0, p1 math.Point3) (geom.Curve3, bool) {
	x, y := c.MajorAxis.AsVector(), c.Normal.Cross(c.MajorAxis)
	a0 := stdmath.Atan2(float64(c.Center.VectorTo(p0).Dot(y))/c.MinorRadius, float64(c.Center.VectorTo(p0).Dot(x))/c.MajorRadius)
	a1 := stdmath.Atan2(float64(c.Center.VectorTo(p1).Dot(y))/c.MinorRadius, float64(c.Center.VectorTo(p1).Dot(x))/c.MajorRadius)
	start, sweep := pickArc(a0, a1, p0.Midpoint(p1), func(s, sw float64) math.Point3 {
		return ellipsePointAt(c, s+0.5*sw)
	})
	arc, err := geom.NewEllipticalArc(c.Center, c.Normal.AsVector(), x, c.MajorRadius, c.MinorRadius, start, sweep)
	return arc, err == nil
}

// pickArc chooses (start=a0, sweep) so the arc from a0 to a1 bulges toward chordMid: it compares the
// short (positive) sweep against its 2π complement by which one's midpoint is nearer chordMid.
func pickArc(a0, a1 float64, chordMid math.Point3, midAt func(start, sweep float64) math.Point3) (float64, float64) {
	sweep := wrapTwoPi(a1 - a0)
	alt := sweep - 2*stdmath.Pi
	if midAt(a0, alt).DistanceTo(chordMid) < midAt(a0, sweep).DistanceTo(chordMid) {
		return a0, alt
	}
	return a0, sweep
}

// angleInPlane returns the angle of point p about origin in the (x,y) in-plane basis.
func angleInPlane(origin math.Point3, x, y math.Vector3, p math.Point3) float64 {
	return stdmath.Atan2(float64(origin.VectorTo(p).Dot(y)), float64(origin.VectorTo(p).Dot(x)))
}

// pointOnCircleAt evaluates circle c at angle a (RefDir at a=0).
func pointOnCircleAt(c geom.Circle, a float64) math.Point3 {
	x, y := c.RefDir.AsVector(), c.Normal.Cross(c.RefDir)
	return c.Center.TranslateBy(x.Scale(math.Scalar(c.Radius * stdmath.Cos(a))).Add(y.Scale(math.Scalar(c.Radius * stdmath.Sin(a)))))
}

// ellipsePointAt evaluates ellipse c at angle a (MajorAxis at a=0).
func ellipsePointAt(c geom.EllipseFull, a float64) math.Point3 {
	x, y := c.MajorAxis.AsVector(), c.Normal.Cross(c.MajorAxis)
	return c.Center.TranslateBy(x.Scale(math.Scalar(c.MajorRadius * stdmath.Cos(a))).Add(y.Scale(math.Scalar(c.MinorRadius * stdmath.Sin(a)))))
}

// wrapTwoPi maps an angle into [0, 2π).
func wrapTwoPi(a float64) float64 {
	for a < 0 {
		a += 2 * stdmath.Pi
	}
	for a >= 2*stdmath.Pi {
		a -= 2 * stdmath.Pi
	}
	return a
}
