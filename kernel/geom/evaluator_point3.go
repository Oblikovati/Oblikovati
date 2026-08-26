// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Closest-point classification and range boxes for 3D curves (M01-F06, #603).
// Closed forms cover lines, circles and arcs; everything else runs a sampled
// multistart Newton search whose minima are clustered into a SolutionNature.

// CurveParamAtPoint3 returns the parameter of the point on c closest to p and
// classifies how many equally close answers exist (e.g. a circle queried from
// a point on its axis has infinitely many).
func CurveParamAtPoint3(c Curve3, p math.Point3) (float64, SolutionNature) {
	switch g := c.(type) {
	case Line:
		return float64(g.Origin.VectorTo(p).Dot(g.Dir.AsVector())), UniqueSolution
	case LineSegment:
		return segmentParamAtPoint3(g, p), UniqueSolution
	case Circle:
		return circleParamAtPoint3(g, p)
	case Arc3d:
		return arcParamAtPoint3(g, p)
	case Polyline:
		return polylineParamAtPoint3(g, p)
	case Hyperbola, HyperbolicArc, Parabola, ParabolicArc:
		return openConicParamAtPoint3(c, p)
	default:
		return genericParamAtPoint3(c, p)
	}
}

// openConicParamAtPoint3 inverts an open conic section (hyperbola branch or parabola) at a point on it:
// the unbounded curve maps to its native parameter, the bounded arc rescales that onto [0, 1].
func openConicParamAtPoint3(c Curve3, p math.Point3) (float64, SolutionNature) {
	switch g := c.(type) {
	case Hyperbola:
		return hyperbolaThetaAtPoint3(g.Center, g.ConjugateAxis.AsVector(), g.B, p), UniqueSolution
	case HyperbolicArc:
		theta := hyperbolaThetaAtPoint3(g.Center, g.ConjugateAxis.AsVector(), g.B, p)
		return (theta - g.Theta0) / (g.Theta1 - g.Theta0), UniqueSolution
	case Parabola:
		return parabolaTAtPoint3(g.Vertex, g.CrossDir.AsVector(), p), UniqueSolution
	case ParabolicArc:
		t := parabolaTAtPoint3(g.Vertex, g.CrossDir.AsVector(), p)
		return (t - g.T0) / (g.T1 - g.T0), UniqueSolution
	default:
		return 0, UniqueSolution
	}
}

// parabolaTAtPoint3 inverts a parabola: the parameter t is the cross coordinate directly (x along
// CrossDir), so a point on the curve maps back by projecting onto CrossDir.
func parabolaTAtPoint3(vertex math.Point3, cross math.Vector3, p math.Point3) float64 {
	return float64(vertex.VectorTo(p).Dot(cross))
}

// hyperbolaThetaAtPoint3 inverts a hyperbola branch: with the conjugate coordinate y = B·sinh(θ),
// θ = asinh(y/B). A point on the branch maps back to its exact parameter (the branch is simple, so
// the conjugate component alone fixes θ — no sign ambiguity).
func hyperbolaThetaAtPoint3(center math.Point3, conjugate math.Vector3, b float64, p math.Point3) float64 {
	y := float64(center.VectorTo(p).Dot(conjugate))
	return stdmath.Asinh(y / b)
}

// segmentParamAtPoint3 projects p onto the segment's chord and clamps.
func segmentParamAtPoint3(g LineSegment, p math.Point3) float64 {
	chord := g.StartPoint.VectorTo(g.EndPoint)
	den := float64(chord.LengthSquared())
	if den == 0 {
		return 0
	}
	return math.Clamp01(float64(g.StartPoint.VectorTo(p).Dot(chord)) / den)
}

// circleParamAtPoint3 inverts the circle angle of p's in-plane direction; a
// point on the circle's axis is equidistant from the whole circle.
func circleParamAtPoint3(g Circle, p math.Point3) (float64, SolutionNature) {
	angle, ok := inPlaneAngle(g.Center, g.RefDir.AsVector(), g.binormal(), g.Radius, p)
	if !ok {
		return 0, InfinitelyManySolutions
	}
	return wrap2pi(angle) / twoPi, UniqueSolution
}

// arcParamAtPoint3 inverts the angle and resolves it against the sweep: inside
// the sweep the answer is unique; outside, the closer end point wins, and the
// exact mid-gap point is equidistant from both ends.
func arcParamAtPoint3(g Arc3d, p math.Point3) (float64, SolutionNature) {
	angle, ok := inPlaneAngle(g.Center, g.RefDir.AsVector(), g.binormal(), g.Radius, p)
	if !ok {
		return 0, InfinitelyManySolutions
	}
	return resolveSweep(angle-g.StartAngle, g.SweepAngle, func(t float64) float64 {
		return g.PointAt(t).DistanceTo(p)
	})
}

// inPlaneAngle returns the angle of p's projection into the circle frame,
// ok=false when p sits on the axis (no in-plane direction; scale-relative test).
func inPlaneAngle(center math.Point3, ref, bin math.Vector3, radius float64, p math.Point3) (float64, bool) {
	d := center.VectorTo(p)
	x, y := float64(d.Dot(ref)), float64(d.Dot(bin))
	if stdmath.Hypot(x, y) <= 1e-12*stdmath.Max(1, radius) { // tol:numeric — point on the axis: angular param undefined (relative to radius)
		return 0, false
	}
	return stdmath.Atan2(y, x), true
}

// resolveSweep maps a relative angle onto the arc parameter t∈[0,1] for the
// signed sweep, classifying the off-arc tie between the two end points.
func resolveSweep(rel, sweep float64, distAt func(float64) float64) (float64, SolutionNature) {
	span := stdmath.Abs(sweep)
	if sweep < 0 {
		rel = -rel
	}
	rel = wrap2pi(rel)
	if rel <= span {
		return rel / span, UniqueSolution
	}
	gapMid := span + (twoPi-span)/2
	d0, d1 := distAt(0), distAt(1)
	if stdmath.Abs(rel-gapMid) <= 1e-12 || stdmath.Abs(d0-d1) <= 1e-12*stdmath.Max(1, d0) { // tol:numeric — antipodal/equal-distance degeneracy guard
		return closerEnd(d0, d1), DistinctlyManySolutions
	}
	return closerEnd(d0, d1), UniqueSolution
}

// closerEnd returns the parameter of the nearer arc end.
func closerEnd(d0, d1 float64) float64 {
	if d1 < d0 {
		return 1
	}
	return 0
}

// polylineParamAtPoint3 scans every segment's clamped foot, classifying ties
// between distinct foot positions as distinctly-many.
func polylineParamAtPoint3(g Polyline, p math.Point3) (float64, SolutionNature) {
	segs := len(g.Vertices) - 1
	best, bestT, ties := stdmath.Inf(1), 0.0, 0
	var bestFoot math.Point3
	for i := range segs {
		seg := NewLineSegment(g.Vertices[i], g.Vertices[i+1])
		local := segmentParamAtPoint3(seg, p)
		foot := seg.PointAt(local)
		d := foot.DistanceTo(p)
		tol := 1e-12 * stdmath.Max(1, best) // tol:numeric — first-segment acceptance, relative to best distance
		switch {
		case d < best-tol:
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

// genericParamAtPoint3 is the sampled multistart search for curves without a
// closed-form inverse (ellipses, helices, NURBS): sample densely, refine each
// near-minimal sample by Newton, then cluster the refined winners.
func genericParamAtPoint3(c Curve3, p math.Point3) (float64, SolutionNature) {
	lo, hi := c.Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return 0, UnknownSolutionNature
	}
	ts, ds := sampleDistances3(c, p, lo, hi)
	best := stdmath.Inf(1)
	for _, d := range ds {
		best = stdmath.Min(best, d)
	}
	tol := 1e-9 * stdmath.Max(1, best) // tol:numeric — near-minimum clustering, relative to best distance
	if count := nearCount(ds, best, tol); count > closestSamples/2 {
		return ts[0], InfinitelyManySolutions
	}
	return clusterMinima3(c, p, ts, ds, best, tol)
}

// sampleDistances3 evaluates the distance to p at uniform parameters.
func sampleDistances3(c Curve3, p math.Point3, lo, hi float64) (ts, ds []float64) {
	ts = make([]float64, closestSamples+1)
	ds = make([]float64, closestSamples+1)
	for i := range ts {
		ts[i] = lo + (hi-lo)*float64(i)/float64(closestSamples)
		ds[i] = c.PointAt(ts[i]).DistanceTo(p)
	}
	return ts, ds
}

// nearCount counts samples within tol of the global minimum.
func nearCount(ds []float64, best, tol float64) int {
	n := 0
	for _, d := range ds {
		if d <= best+tol {
			n++
		}
	}
	return n
}

// clusterMinima3 Newton-refines every near-minimal sample and clusters the
// refined parameters: one cluster is unique, several equally-close clusters
// are distinctly many.
func clusterMinima3(c Curve3, p math.Point3, ts, ds []float64, best, tol float64) (float64, SolutionNature) {
	lo, hi := c.Domain()
	var clusters []float64
	bestT, refinedBest := ts[0], stdmath.Inf(1)
	for i, d := range ds {
		if d > best+tol {
			continue
		}
		t := refineClosest3(c, p, ts[i], lo, hi)
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

// appendCluster adds t unless it merges into an existing cluster within width.
func appendCluster(clusters []float64, t, width float64) []float64 {
	for _, c := range clusters {
		if stdmath.Abs(c-t) <= 2*width {
			return clusters
		}
	}
	return append(clusters, t)
}

// refineClosest3 polishes a closest-point candidate by Newton on
// g(t) = (P(t)−p)·P′(t), clamped to the domain.
func refineClosest3(c Curve3, p math.Point3, t, lo, hi float64) float64 {
	for range 16 {
		d1, d2, _ := CurveDerivatives3(c, t)
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

// CurveRangeBox3 returns an enclosing axis-aligned box: exact for analytic
// curves (per-axis sinusoid extrema), the control-hull box for NURBS
// (enclosing but not minimal), ±Inf faces for an unbounded line.
func CurveRangeBox3(c Curve3) math.Box {
	switch g := c.(type) {
	case Line, Hyperbola, Parabola:
		inf := stdmath.Inf(1)
		return math.Box{Min: math.P3(-inf, -inf, -inf), Max: math.P3(inf, inf, inf)}
	case LineSegment:
		return math.BoxFromPoints(g.StartPoint, g.EndPoint)
	case Circle:
		return sinusoidBox(g.Center, g.RefDir.AsVector(), g.binormal(), g.Radius, g.Radius, 0, twoPi)
	case Arc3d:
		return sinusoidBox(g.Center, g.RefDir.AsVector(), g.binormal(), g.Radius, g.Radius, g.StartAngle, g.SweepAngle)
	case EllipseFull:
		return sinusoidBox(g.Center, g.MajorAxis.AsVector(), g.minorAxis(), g.MajorRadius, g.MinorRadius, 0, twoPi)
	case EllipticalArc:
		return sinusoidBox(g.Center, g.MajorAxis.AsVector(), g.minorAxis(), g.MajorRadius, g.MinorRadius, g.StartAngle, g.SweepAngle)
	case Polyline:
		return math.BoxFromPoints(g.Vertices...)
	case BSplineCurve:
		return math.BoxFromPoints(g.Ctrl...)
	default:
		return sampledBox3(c)
	}
}

// sinusoidBox bounds C + a·cosθ·major + b·sinθ·minor over θ in the sweep from
// start: per world axis the extremum angle is atan2(b·minorᵢ, a·majorᵢ), taken
// where it falls inside the sweep, plus the sweep ends.
func sinusoidBox(center math.Point3, major, minor math.Vector3, a, b, start, sweep float64) math.Box {
	angles := []float64{start, start + sweep}
	for axis := range 3 {
		mj, mn := vectorComponent(major, axis), vectorComponent(minor, axis)
		extremum := stdmath.Atan2(b*mn, a*mj)
		angles = append(angles, anglesInSweep(extremum, start, sweep)...)
		angles = append(angles, anglesInSweep(extremum+stdmath.Pi, start, sweep)...)
	}
	pts := make([]math.Point3, len(angles))
	for i, ang := range angles {
		cos, sin := cosSin(ang)
		pts[i] = center.TranslateBy(major.Scale(a * cos).Add(minor.Scale(b * sin)))
	}
	return math.BoxFromPoints(pts...)
}

// anglesInSweep returns the representatives of angle (mod 2π) lying inside the
// signed sweep from start.
func anglesInSweep(angle, start, sweep float64) []float64 {
	lo, hi := start, start+sweep
	if sweep < 0 {
		lo, hi = hi, lo
	}
	base := lo + wrap2pi(angle-lo)
	var out []float64
	for a := base - twoPi; a <= hi; a += twoPi {
		if a >= lo {
			out = append(out, a)
		}
	}
	return out
}

// vectorComponent returns the axis-indexed component of v.
func vectorComponent(v math.Vector3, axis int) float64 {
	switch axis {
	case 0:
		return v.X
	case 1:
		return v.Y
	default:
		return v.Z
	}
}

// sampledBox3 boxes a dense sampling — the fallback for curves without a
// closed-form bound (the helix and unknown implementations).
func sampledBox3(c Curve3) math.Box {
	lo, hi := c.Domain()
	pts := make([]math.Point3, 257)
	for i := range pts {
		pts[i] = c.PointAt(lo + (hi-lo)*float64(i)/256)
	}
	return padBox(math.BoxFromPoints(pts...), sampledBoxPadding(c, lo, hi))
}

// sampledBoxPadding over-estimates the sag between box samples by the maximum
// speed times half the sample step, so the sampled box still encloses the curve.
func sampledBoxPadding(c Curve3, lo, hi float64) float64 {
	maxSpeed := 0.0
	for i := 0; i <= 64; i++ {
		d1, _, _ := CurveDerivatives3(c, lo+(hi-lo)*float64(i)/64)
		maxSpeed = stdmath.Max(maxSpeed, float64(d1.Length()))
	}
	return maxSpeed * (hi - lo) / 512
}

// padBox grows the box by pad on every face.
func padBox(b math.Box, pad float64) math.Box {
	return math.Box{
		Min: math.P3(b.Min.X-pad, b.Min.Y-pad, b.Min.Z-pad),
		Max: math.P3(b.Max.X+pad, b.Max.Y+pad, b.Max.Z+pad),
	}
}
