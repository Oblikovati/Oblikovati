// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Closest-point classification and range boxes for 2D curves — the
// sketch-space twin of evaluator_point3.go (M01-F06, #603).

// CurveParamAtPoint2 returns the parameter of the point on c closest to p and
// classifies how many equally close answers exist.
func CurveParamAtPoint2(c Curve2, p math.Point2) (float64, SolutionNature) {
	switch g := c.(type) {
	case Line2d:
		return float64(g.Origin.VectorTo(p).Dot(g.Dir.AsVector())), UniqueSolution
	case LineSegment2d:
		return segmentParamAtPoint2(g, p), UniqueSolution
	case Circle2d:
		return circleParamAtPoint2(g, p)
	case Arc2d:
		return arcParamAtPoint2(g, p)
	case Polyline2d:
		return polylineParamAtPoint2(g, p)
	default:
		return genericParamAtPoint2(c, p)
	}
}

// segmentParamAtPoint2 projects p onto the segment's chord and clamps.
func segmentParamAtPoint2(g LineSegment2d, p math.Point2) float64 {
	chord := g.StartPoint.VectorTo(g.EndPoint)
	den := float64(chord.LengthSquared())
	if den == 0 {
		return 0
	}
	return math.Clamp01(float64(g.StartPoint.VectorTo(p).Dot(chord)) / den)
}

// circleParamAtPoint2 inverts the angle of p about the center; the center
// itself is equidistant from the whole circle.
func circleParamAtPoint2(g Circle2d, p math.Point2) (float64, SolutionNature) {
	d := g.Center.VectorTo(p)
	if float64(d.Length()) <= 1e-12*stdmath.Max(1, g.Radius) { // tol:numeric — point AT the circle/arc centre: param undefined (relative to radius)
		return 0, InfinitelyManySolutions
	}
	return wrap2pi(stdmath.Atan2(d.Y, d.X)) / twoPi, UniqueSolution
}

// arcParamAtPoint2 inverts the angle and resolves it against the sweep.
func arcParamAtPoint2(g Arc2d, p math.Point2) (float64, SolutionNature) {
	d := g.Center.VectorTo(p)
	if float64(d.Length()) <= 1e-12*stdmath.Max(1, g.Radius) { // tol:numeric — point AT the circle/arc centre: param undefined (relative to radius)
		return 0, InfinitelyManySolutions
	}
	angle := stdmath.Atan2(d.Y, d.X)
	return resolveSweep(angle-g.StartAngle, g.SweepAngle, func(t float64) float64 {
		return g.PointAt(t).DistanceTo(p)
	})
}

// polylineParamAtPoint2 scans every segment's clamped foot, classifying ties
// between distinct foot positions as distinctly-many.
func polylineParamAtPoint2(g Polyline2d, p math.Point2) (float64, SolutionNature) {
	segs := len(g.Vertices) - 1
	best, bestT, ties := stdmath.Inf(1), 0.0, 0
	var bestFoot math.Point2
	for i := range segs {
		seg := NewLineSegment2d(g.Vertices[i], g.Vertices[i+1])
		local := segmentParamAtPoint2(seg, p)
		foot := seg.PointAt(local)
		d := foot.DistanceTo(p)
		// tol is a scale-relative band for the closer/tie decision. It must be computed from a
		// FINITE reference: seeding best with +Inf made 1e-12*best = +Inf, so best-tol was NaN and
		// the "strictly closer" guard was always false — every point on a polyline resolved to the
		// start (param 0). Base it on the candidate distance, and always accept the first segment.
		tol := 1e-12 * stdmath.Max(1, d) // tol:numeric — first-segment acceptance, relative to candidate distance
		switch {
		case stdmath.IsInf(best, 1) || d < best-tol:
			best, bestT, ties, bestFoot = d, (float64(i)+local)/float64(segs), 0, foot
		case d <= best+tol && foot.DistanceTo(bestFoot) > math.DefaultTolerance:
			ties++
		}
	}
	if ties > 0 {
		return bestT, DistinctlyManySolutions
	}
	return bestT, UniqueSolution
}

// genericParamAtPoint2 is the sampled multistart search for curves without a
// closed-form inverse (ellipses, NURBS).
func genericParamAtPoint2(c Curve2, p math.Point2) (float64, SolutionNature) {
	lo, hi := c.Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return 0, UnknownSolutionNature
	}
	ts, ds := sampleDistances2(c, p, lo, hi)
	best := stdmath.Inf(1)
	for _, d := range ds {
		best = stdmath.Min(best, d)
	}
	tol := 1e-9 * stdmath.Max(1, best) // tol:numeric — near-minimum clustering, relative to best distance
	if count := nearCount(ds, best, tol); count > closestSamples/2 {
		return ts[0], InfinitelyManySolutions
	}
	return clusterMinima2(c, p, ts, ds, best, tol)
}

// sampleDistances2 evaluates the distance to p at uniform parameters.
func sampleDistances2(c Curve2, p math.Point2, lo, hi float64) (ts, ds []float64) {
	ts = make([]float64, closestSamples+1)
	ds = make([]float64, closestSamples+1)
	for i := range ts {
		ts[i] = lo + (hi-lo)*float64(i)/float64(closestSamples)
		ds[i] = c.PointAt(ts[i]).DistanceTo(p)
	}
	return ts, ds
}

// clusterMinima2 Newton-refines every near-minimal sample and clusters the
// refined parameters.
func clusterMinima2(c Curve2, p math.Point2, ts, ds []float64, best, tol float64) (float64, SolutionNature) {
	lo, hi := c.Domain()
	var clusters []float64
	bestT, refinedBest := ts[0], stdmath.Inf(1)
	for i, d := range ds {
		if d > best+tol {
			continue
		}
		t := refineClosest2(c, p, ts[i], lo, hi)
		rd := c.PointAt(t).DistanceTo(p)
		if rd < refinedBest-tol {
			refinedBest, bestT, clusters = rd, t, clusters[:0]
		}
		if rd <= refinedBest+tol {
			clusters = appendCluster(clusters, t, (hi-lo)/closestSamples)
		}
	}
	if len(clusters) > 1 {
		return bestT, DistinctlyManySolutions
	}
	return bestT, UniqueSolution
}

// refineClosest2 polishes a closest-point candidate by Newton on
// g(t) = (P(t)−p)·P′(t), clamped to the domain.
func refineClosest2(c Curve2, p math.Point2, t, lo, hi float64) float64 {
	for range 16 {
		d1, d2, _ := CurveDerivatives2(c, t)
		r := p.VectorTo(c.PointAt(t))
		g := float64(r.Dot(d1))
		dg := float64(d1.LengthSquared()) + float64(r.Dot(d2))
		if dg == 0 || stdmath.Abs(g) < 1e-14 { // tol:numeric — Newton denominator near-zero guard
			return t
		}
		t = math.Clamp(t-g/dg, lo, hi)
	}
	return t
}

// CurveRangeBox2 returns an enclosing axis-aligned 2D box: exact for analytic
// curves, the control-hull box for NURBS, ±Inf for an unbounded line.
func CurveRangeBox2(c Curve2) math.Box2d {
	switch g := c.(type) {
	case Line2d:
		inf := stdmath.Inf(1)
		return math.Box2d{Min: math.P2(-inf, -inf), Max: math.P2(inf, inf)}
	case LineSegment2d:
		return math.Box2dFromPoints(g.StartPoint, g.EndPoint)
	case Circle2d:
		return math.Box2dFromPoints(
			math.P2(g.Center.X-g.Radius, g.Center.Y-g.Radius),
			math.P2(g.Center.X+g.Radius, g.Center.Y+g.Radius))
	case Arc2d:
		return sinusoidBox2(g.Center, math.V2(1, 0), math.V2(0, 1), g.Radius, g.Radius, g.StartAngle, g.SweepAngle)
	case EllipseFull2d:
		return sinusoidBox2(g.Center, g.MajorAxis.AsVector(), g.minorAxis(), g.MajorRadius, g.MinorRadius, 0, twoPi)
	case EllipticalArc2d:
		return sinusoidBox2(g.Center, g.MajorAxis.AsVector(), g.minorAxis(), g.MajorRadius, g.MinorRadius, g.StartAngle, g.SweepAngle)
	case Polyline2d:
		return math.Box2dFromPoints(g.Vertices...)
	case BSplineCurve2d:
		return math.Box2dFromPoints(g.Ctrl...)
	default:
		return sampledBox2(c)
	}
}

// sinusoidBox2 bounds C + a·cosθ·major + b·sinθ·minor over the sweep — the 2D
// twin of sinusoidBox.
func sinusoidBox2(center math.Point2, major, minor math.Vector2, a, b, start, sweep float64) math.Box2d {
	angles := []float64{start, start + sweep}
	for axis := range 2 {
		mj, mn := vectorComponent2(major, axis), vectorComponent2(minor, axis)
		extremum := stdmath.Atan2(b*mn, a*mj)
		angles = append(angles, anglesInSweep(extremum, start, sweep)...)
		angles = append(angles, anglesInSweep(extremum+stdmath.Pi, start, sweep)...)
	}
	pts := make([]math.Point2, len(angles))
	for i, ang := range angles {
		cos, sin := cosSin(ang)
		pts[i] = center.TranslateBy(major.Scale(a * cos).Add(minor.Scale(b * sin)))
	}
	return math.Box2dFromPoints(pts...)
}

// vectorComponent2 returns the axis-indexed component of v.
func vectorComponent2(v math.Vector2, axis int) float64 {
	if axis == 0 {
		return v.X
	}
	return v.Y
}

// sampledBox2 boxes a dense padded sampling — the fallback for unknown curves.
func sampledBox2(c Curve2) math.Box2d {
	lo, hi := c.Domain()
	pts := make([]math.Point2, 257)
	for i := range pts {
		pts[i] = c.PointAt(lo + (hi-lo)*float64(i)/256)
	}
	b := math.Box2dFromPoints(pts...)
	pad := sampledBoxPadding2(c, lo, hi)
	return math.Box2d{
		Min: math.P2(b.Min.X-pad, b.Min.Y-pad),
		Max: math.P2(b.Max.X+pad, b.Max.Y+pad),
	}
}

// sampledBoxPadding2 over-estimates the sag between samples.
func sampledBoxPadding2(c Curve2, lo, hi float64) float64 {
	maxSpeed := 0.0
	for i := 0; i <= 64; i++ {
		d1, _, _ := CurveDerivatives2(c, lo+(hi-lo)*float64(i)/64)
		maxSpeed = stdmath.Max(maxSpeed, float64(d1.Length()))
	}
	return maxSpeed * (hi - lo) / 512
}
