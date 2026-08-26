// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// ballDevGrid is the interior sample grid MaxBallDev is measured on: 12×12 parameters strictly inside
// the patch domain, at (i+1)/(n+1), so no sample sits on a boundary the other certificate fields
// already cover. 144 points is ample — a 5-point Gauss rule already converges these patches' area
// integrals at 4×4 (n4PatchAreaCells) — and it costs a few hundred conic sections per build, against a
// corpus run that tessellates every one of these patches anyway.
const ballDevGrid = 12

// maxBallDev is the INTERIOR rolling-ball envelope residual, and the ONLY certificate field that says
// anything about a patch's interior. coons4-audit.md §C.3 showed the rest are boundary or structural
// properties — MaxDev is ~1e-14 after boundary pinning, MaxAngleDev is defined only ON the rails — so a
// fill could be ANY fold-free surface through the four rails and still certify. Nine corpus greens were
// exactly that: rails right to 1e-14, interior 9–19% of r off OCCT's own surface.
//
// The measure, for each interior sample p:
//
//	s      = p's SECTION plane: through p, normal env.Spine (exact for a straight spine — every point
//	         of one rolling-ball section shares the same spine coordinate)
//	c      = the ball centre solved IN that plane from p plus ONE declared host
//	residual = |(the OTHER declared hosts' own distance to c) − Radius|
//
// It is normal-free by construction. An earlier draft took c = p ± Radius·n from the surface normal and
// was discarded on measurement: the loft's station-to-station normal ripple is ~0.013 rad even where the
// surface agrees with OCCT's to 0.023% of r, so that form read 3.9% of r on an exactly-correct band and
// could not have separated the false greens from the true ones. Solving c from the geometry instead of
// reading it off the surface removes the parametrisation entirely.
//
// It returns 0 when the extractor supplied no envelope, or one this measure cannot express. That is
// deliberate: coons4-audit.md §B.4 measured a certify-time GUESS at the roll hosts reading 5–19%
// residual even on OCCT's own CORRECT patches, so an inferred payload would be worse than none — only an
// extractor that knows the topology may make this claim.
func maxBallDev(surf geom.Surface, env *BallEnvelope) float64 {
	if !env.measurable() {
		return 0
	}
	worst := 0.0
	for _, p := range ballDevSamplePoints(surf) {
		d, ok := ballSampleResidual(p, env)
		if !ok {
			return stdmath.Inf(1) // the section does not meet a declared host: honest reject
		}
		worst = stdmath.Max(worst, d)
	}
	return worst
}

// measurable reports whether the envelope names a constraint pair this residual can solve: exactly the
// two SETBACK-CLOSE run-out flavours — one tangency host plus one restriction curve (a flank / one-boss
// central), or two restriction curves (a two-boss central). Anything else measures 0 rather than
// pretending.
func (env *BallEnvelope) measurable() bool {
	if env == nil || env.Radius <= 0 || float64(env.Spine.Length()) == 0 {
		return false
	}
	return (len(env.Tangents) == 1 && len(env.Through) == 1) || (len(env.Tangents) == 0 && len(env.Through) == 2)
}

// ballDevSamplePoints lays the strictly-interior grid over the patch domain and evaluates it.
func ballDevSamplePoints(surf geom.Surface) []math.Point3 {
	u0, u1, v0, v1 := surfaceDomain(surf)
	out := make([]math.Point3, 0, ballDevGrid*ballDevGrid)
	for i := range ballDevGrid {
		u := u0 + (u1-u0)*float64(i+1)/float64(ballDevGrid+1)
		for j := range ballDevGrid {
			out = append(out, surf.PointAt(u, v0+(v1-v0)*float64(j+1)/float64(ballDevGrid+1)))
		}
	}
	return out
}

// surfaceDomain reads a surface's parameter box, falling back to the unit square for a surface that
// does not expose one (every patch surface in this engine is a BSplineSurface, which does).
func surfaceDomain(surf geom.Surface) (u0, u1, v0, v1 float64) {
	if bs, ok := surf.(geom.BSplineSurface); ok {
		u0, u1 = bs.UDomain()
		v0, v1 = bs.VDomain()
		return u0, u1, v0, v1
	}
	return 0, 1, 0, 1
}

// ballSampleResidual is the residual at one sample: solve the ball centre in p's section plane from p
// plus the FIRST declared host, then measure the SECOND host's own distance to it. Every candidate root
// is tried and the best kept — the roots are ~2r apart, so a genuinely wrong surface cannot be rescued
// by the alternative branch, while a correct one needs no branch rule at all.
func ballSampleResidual(p math.Point3, env *BallEnvelope) (float64, bool) {
	other, ok := sectionCurvePoint(env.Through[len(env.Through)-1], p, env.Spine)
	if !ok {
		return 0, false
	}
	centres, ok := ballCentreCandidates(p, env)
	if !ok {
		return 0, false
	}
	best := stdmath.Inf(1)
	for _, c := range centres {
		best = stdmath.Min(best, stdmath.Abs(float64(c.DistanceTo(other))-env.Radius))
	}
	return best, true
}

// ballCentreCandidates solves, inside p's section plane, the ball centres of radius Radius that touch p
// AND satisfy the FIRST declared host: the tangency plane for a surf-rst band (a line∩circle solve), or
// the first restriction curve's section point for a rst-rst band (a circle∩circle solve).
func ballCentreCandidates(p math.Point3, env *BallEnvelope) ([]math.Point3, bool) {
	if len(env.Tangents) == 1 {
		plane, ok := env.Tangents[0].(geom.Plane)
		if !ok {
			return nil, false
		}
		return tangentPlaneCentres(p, plane, env), true
	}
	q, ok := sectionCurvePoint(env.Through[0], p, env.Spine)
	if !ok {
		return nil, false
	}
	return equidistantCentres(p, q, env), true
}

// tangentPlaneCentres returns the section-plane points at distance Radius from BOTH p and the tangency
// plane. The centre lies on one of the two offset lines L± = Π ∩ (plane ± Radius); each meets the
// radius-Radius circle about p in at most two points.
func tangentPlaneCentres(p math.Point3, plane geom.Plane, env *BallEnvelope) []math.Point3 {
	n := plane.Normal()
	nl := float64(n.Length())
	if nl == 0 {
		return nil
	}
	unit := n.Scale(math.Scalar(1 / nl))
	dir := unit.Cross(env.Spine)
	dl := float64(dir.Length())
	if dl == 0 {
		return nil // the spine lies along the host normal: not a fillet host
	}
	dir = dir.Scale(math.Scalar(1 / dl))
	signed := float64(plane.Origin.VectorTo(p).Dot(unit))
	var out []math.Point3
	for _, side := range []float64{1, -1} {
		base := p.TranslateBy(unit.Scale(math.Scalar(side*env.Radius - signed)))
		out = append(out, circleLineHits(p, base, dir, env.Radius)...)
	}
	return out
}

// circleLineHits returns the points on the line base+t·dir at distance radius from p (0, 1 or 2 of them).
func circleLineHits(p, base math.Point3, dir math.Vector3, radius float64) []math.Point3 {
	w := p.VectorTo(base)
	b := float64(w.Dot(dir))
	disc := radius*radius - (float64(w.LengthSquared()) - b*b)
	if disc < 0 {
		return nil
	}
	h := stdmath.Sqrt(disc)
	return []math.Point3{
		base.TranslateBy(dir.Scale(math.Scalar(-b + h))),
		base.TranslateBy(dir.Scale(math.Scalar(-b - h))),
	}
}

// equidistantCentres returns the section-plane points at distance Radius from both p and q — the
// circle∩circle solve a rst-rst band's centre satisfies.
func equidistantCentres(p, q math.Point3, env *BallEnvelope) []math.Point3 {
	mid := p.Midpoint(q)
	delta := p.VectorTo(q)
	half := 0.5 * float64(delta.Length())
	disc := env.Radius*env.Radius - half*half
	cross := env.Spine.Cross(delta)
	l := float64(cross.Length())
	if disc < 0 || l == 0 {
		return nil
	}
	off := cross.Scale(math.Scalar(stdmath.Sqrt(disc) / l))
	return []math.Point3{mid.TranslateBy(off), mid.TranslateBy(off.Negate())}
}

// ballCurveSamples is the coarse scan resolution for sectionCurvePoint — fine enough to isolate every
// crossing of a footprint conic with a section plane (a conic meets a plane at most twice) before the
// bisection takes each root to parameter precision.
const ballCurveSamples = 256

// sectionCurvePoint is where p's SECTION plane (through p, normal spine) crosses the restriction curve,
// taking the crossing nearest p when the plane meets the conic twice (the band side). ok=false when the
// plane misses the curve entirely — a patch whose interior wandered off the boss's own spine span, which
// must reject rather than fall back on some unrelated curve point.
func sectionCurvePoint(c geom.Curve3, p math.Point3, spine math.Vector3) (math.Point3, bool) {
	lo, hi := c.Domain()
	height := func(t float64) float64 { return float64(p.VectorTo(c.PointAt(t)).Dot(spine)) }
	best, found := math.Point3{}, false
	bestD := stdmath.Inf(1)
	prevT, prevH := lo, height(lo)
	for i := 1; i <= ballCurveSamples; i++ {
		t := lo + (hi-lo)*float64(i)/ballCurveSamples
		h := height(t)
		if prevH == 0 || (prevH < 0) != (h < 0) {
			hit := c.PointAt(bisectHeight(height, prevT, t))
			if d := float64(hit.DistanceTo(p)); d < bestD {
				best, bestD, found = hit, d, true
			}
		}
		prevT, prevH = t, h
	}
	return best, found
}

// bisectHeight refines a bracketed sign change of the section-plane height function to parameter
// precision (50 halvings take any real span to ~1e-15 relative).
func bisectHeight(height func(float64) float64, a, b float64) float64 {
	fa := height(a)
	for range 50 {
		m := 0.5 * (a + b)
		if fm := height(m); (fm < 0) == (fa < 0) {
			a, fa = m, fm
			continue
		}
		b = m
	}
	return 0.5 * (a + b)
}
