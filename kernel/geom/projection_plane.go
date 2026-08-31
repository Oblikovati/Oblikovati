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
		if normalParallelToPlane(k.Normal, pl) {
			return NewCircle2d(planeUV(pl, k.Center), k.Radius), true
		}
		return projectCircleToEllipse2d(pl, k) // oblique circle → ellipse
	case Arc3d:
		if normalParallelToPlane(k.Normal, pl) {
			return projectParallelArc2d(pl, k), true
		}
		return projectArcToEllipse2d(pl, k) // oblique arc → elliptical arc
	default:
		return nil, false
	}
}

// projectParallelArc2d projects an arc whose plane is parallel to pl. Orthographic projection preserves
// in-plane distances, so the centre and radius carry over unchanged and only the angles move into pl's
// frame. It is built from the arc's OWN defining data, never from three sampled points: a three-point
// fit cannot see a FULL-sweep arc at all — its start and end coincide, so the fit is degenerate and the
// whole projection used to decline to a 48-segment polyline (the Z1 bore-lip regression, #2247) — and it
// is ill-conditioned near a half turn. The chart's positive rotation is U→V about pl's normal, so an arc
// wound about the opposite normal sweeps the other way in the chart.
func projectParallelArc2d(pl Plane, k Arc3d) Curve2 {
	center := planeUV(pl, k.Center)
	lo, _ := k.Domain()
	sweep := k.SweepAngle
	if float64(k.Normal.AsVector().Dot(pl.Normal())) < 0 {
		sweep = -sweep
	}
	return NewArc2d(center, k.Radius, angleOf2d(center, planeUV(pl, k.PointAt(lo))), sweep)
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

// vecUV projects a 3D VECTOR (no origin shift) into pl's (u,v) frame.
func vecUV(pl Plane, v math.Vector3) math.Vector2 {
	return math.V2(v.Dot(pl.UAxis.AsVector()), v.Dot(pl.VAxis.AsVector()))
}

// projectCircleToEllipse2d projects an oblique circle onto pl as the analytic ellipse it becomes
// (semi-major = r, semi-minor = r·cosθ for a tilt θ; ADR-0055 phase 2). The circle's two in-plane
// unit axes project to conjugate semi-diameters r·û, r·v̂ of the ellipse. ok=false when the circle is
// edge-on (projects to a segment), leaving the caller on the sampled fallback.
func projectCircleToEllipse2d(pl Plane, k Circle) (Curve2, bool) {
	u, v := circleConjugateSemiDiameters(pl, k)
	return ellipseFromConjugate(planeUV(pl, k.Center), u, v)
}

// projectArcToEllipse2d projects an oblique arc as the elliptical arc it becomes: the full ellipse
// of its circle, restricted to the arc's span. The span is recovered by mapping the arc's start,
// mid, and end points to the ellipse's eccentric angle and picking the direction through the mid.
func projectArcToEllipse2d(pl Plane, k Arc3d) (Curve2, bool) {
	u, v := arcConjugateSemiDiameters(pl, k)
	full, ok := ellipseFromConjugate(planeUV(pl, k.Center), u, v)
	if !ok {
		return nil, false
	}
	ell := full.(EllipseFull2d)
	lo, hi := k.Domain()
	a0 := ellipseEccentricAngle(ell, planeUV(pl, k.PointAt(lo)))
	am := ellipseEccentricAngle(ell, planeUV(pl, k.PointAt((lo+hi)/2)))
	an := ellipseEccentricAngle(ell, planeUV(pl, k.PointAt(hi)))
	start, sweep := arcSweepThroughMid(a0, am, an)
	arc, err := NewEllipticalArc2d(ell.Center, ell.MajorAxis.AsVector(), ell.MajorRadius, ell.MinorRadius, start, sweep)
	if err != nil {
		return nil, false
	}
	return arc, true
}

// circleConjugateSemiDiameters returns the two 2D conjugate semi-diameters (r·RefDir, r·binormal
// projected) that generate the circle's projected ellipse.
func circleConjugateSemiDiameters(pl Plane, k Circle) (u, v math.Vector2) {
	binormal := k.Normal.AsVector().Cross(k.RefDir.AsVector())
	return vecUV(pl, k.RefDir.AsVector()).Scale(math.Scalar(k.Radius)), vecUV(pl, binormal).Scale(math.Scalar(k.Radius))
}

// arcConjugateSemiDiameters is circleConjugateSemiDiameters for an arc (same circle frame).
func arcConjugateSemiDiameters(pl Plane, k Arc3d) (u, v math.Vector2) {
	binormal := k.Normal.AsVector().Cross(k.RefDir.AsVector())
	return vecUV(pl, k.RefDir.AsVector()).Scale(math.Scalar(k.Radius)), vecUV(pl, binormal).Scale(math.Scalar(k.Radius))
}

// ellipseFromConjugate builds the ellipse {center + u·cosφ + v·sinφ} (u, v conjugate semi-diameters)
// in principal-axis form. The principal axes are the extrema of the radius: at eccentric angle
// θ = ½·atan2(2 u·v, |u|²−|v|²) and θ+90° the two orthogonal semi-diameters are the axes. ok=false
// when the minor radius collapses (a circle projected edge-on → a segment).
func ellipseFromConjugate(center math.Point2, u, v math.Vector2) (Curve2, bool) {
	uu, vv, uv := float64(u.Dot(u)), float64(v.Dot(v)), float64(u.Dot(v))
	theta := 0.5 * stdmath.Atan2(2*uv, uu-vv)
	c, s := math.Scalar(stdmath.Cos(theta)), math.Scalar(stdmath.Sin(theta))
	p1 := u.Scale(c).Add(v.Scale(s))  // semi-diameter at θ
	p2 := u.Scale(-s).Add(v.Scale(c)) // orthogonal semi-diameter at θ+90°
	majorDir, majorR, minorR := p1, float64(p1.Length()), float64(p2.Length())
	if minorR > majorR {
		majorDir, majorR, minorR = p2, minorR, majorR
	}
	if minorR <= planeParallelTol*majorR || majorR == 0 {
		return nil, false // edge-on: the ellipse is a segment
	}
	e, err := NewEllipseFull2d(center, majorDir, majorR, minorR)
	if err != nil {
		return nil, false
	}
	return e, true
}

// ellipseEccentricAngle returns the angle α with p = center + majorR·cosα·major + minorR·sinα·minor.
func ellipseEccentricAngle(e EllipseFull2d, p math.Point2) float64 {
	d := e.Center.VectorTo(p)
	ca := float64(d.Dot(e.MajorAxis.AsVector())) / e.MajorRadius
	sa := float64(d.Dot(e.minorAxis())) / e.MinorRadius
	return stdmath.Atan2(sa, ca)
}

// arcSweepThroughMid returns the start angle and signed sweep from a0 to an that passes through the
// mid angle am (so the elliptical arc covers the source arc's side, not its complement).
func arcSweepThroughMid(a0, am, an float64) (start, sweep float64) {
	ccw := wrapTwoPi(an - a0)
	if wrapTwoPi(am-a0) <= ccw {
		return a0, ccw
	}
	return a0, ccw - 2*stdmath.Pi
}

// wrapTwoPi maps an angle into [0, 2π).
func wrapTwoPi(a float64) float64 {
	a = stdmath.Mod(a, 2*stdmath.Pi)
	if a < 0 {
		a += 2 * stdmath.Pi
	}
	return a
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
