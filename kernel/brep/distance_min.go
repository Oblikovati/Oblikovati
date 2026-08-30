// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Numeric minimisers for the analytic distance recursion (M48/C3 #3458): a multistart bounded scalar
// search that drives a curve's parameter, and an alternating-projection interior solver for a
// surface pair. Both bottom out in geom's exact point projections, so they locate the stationary
// region and the exact projections supply the accuracy — no tessellation enters.

const (
	distScanSamples = 24 // seeds across a curve's domain: dense enough to separate the local minima of a spline edge
	distGridU       = 10 // interior seed grid on a face's parameter box (U); alternating projection refines each seed
	distGridV       = 10 // interior seed grid on a face's parameter box (V)
	distRefineIters = 48 // golden-section refinements around the best curve sample
	distAltIters    = 32 // alternating-projection sweeps between the two surfaces before giving up
)

// distConvergeRel is the alternating-projection stopping test: the foot has stopped moving relative
// to its own magnitude. It floors the point-inversion residual geom.ClosestPointOnSurface already
// drives to, so the outer sweep stops once the inner projections agree.
const distConvergeRel = 1e-11 // tol:relative — foot movement relative to |foot|; see geom projectTol

// invPhi is 1/φ, the golden-section contraction ratio.
const invPhi = 0.6180339887498949

// minimizeOverCurve returns the minimum of field along curve c: a coarse scan over c's bounded domain
// seeds a golden-section refinement around the best sample, so a wiggly spline's global minimum is
// found and then tightened to the exact stationary value.
func minimizeOverCurve(c geom.Curve3, field func(math.Point3) float64) float64 {
	lo, hi := c.Domain()
	bestT, best := lo, stdmath.Inf(1)
	for i := 0; i <= distScanSamples; i++ {
		t := lo + (hi-lo)*float64(i)/float64(distScanSamples)
		if d := field(c.PointAt(t)); d < best {
			best, bestT = d, t
		}
	}
	step := (hi - lo) / float64(distScanSamples)
	return refineOverCurve(c, field, bestT, step, best)
}

// refineOverCurve golden-section contracts the bracket [t-step, t+step] around the best sample,
// clamped to the curve's domain, and returns the tightened minimum value.
func refineOverCurve(c geom.Curve3, field func(math.Point3) float64, t, step, seed float64) float64 {
	lo, hi := c.Domain()
	a, b := clampParam(t-step, lo, hi), clampParam(t+step, lo, hi)
	best := seed
	for i := 0; i < distRefineIters && b-a > 0; i++ {
		m1, m2 := b-(b-a)*invPhi, a+(b-a)*invPhi
		f1, f2 := field(c.PointAt(m1)), field(c.PointAt(m2))
		best = stdmath.Min(best, stdmath.Min(f1, f2))
		if f1 < f2 {
			b = m2
		} else {
			a = m1
		}
	}
	return best
}

// interiorFacePairDistance finds the closest interior approach of two faces — both feet strictly
// inside their trims — by alternating projection from a grid of interior seeds on f1. A seed whose
// iterates ever leave either trim is abandoned (its minimum lies on a boundary, which distFaceFace's
// edge terms carry); the best surviving seed's converged gap wins. +Inf when no interior stationary
// exists (the faces face away, or their closest approach is wholly on the boundary).
func interiorFacePairDistance(f1, f2 *topo.Face) float64 {
	u0, u1, v0, v1, ok := faceParamBox(f1)
	if !ok {
		return stdmath.Inf(1)
	}
	best := stdmath.Inf(1)
	for i := 0; i <= distGridU; i++ {
		for j := 0; j <= distGridV; j++ {
			u := u0 + (u1-u0)*float64(i)/float64(distGridU)
			v := v0 + (v1-v0)*float64(j)/float64(distGridV)
			best = stdmath.Min(best, alternatingFacePair(f1, f2, u, v))
		}
	}
	return best
}

// alternatingFacePair runs one alternating-projection descent seeded at f1(u,v): project the current
// f1 point onto f2, that foot back onto f1, and repeat. It returns the converged gap when both feet
// stay inside their trims, or +Inf the moment a foot leaves a trim (a boundary-dominated seed).
func alternatingFacePair(f1, f2 *topo.Face, u, v float64) float64 {
	p1 := f1.Geometry().PointAt(u, v)
	if !PointInFaceTrim(f1, p1) {
		return stdmath.Inf(1)
	}
	for i := 0; i < distAltIters; i++ {
		foot2, ok := projectIntoTrim(f2, p1)
		if !ok {
			return stdmath.Inf(1)
		}
		next, ok := projectIntoTrim(f1, foot2)
		if !ok {
			return stdmath.Inf(1)
		}
		if footConverged(p1, next) {
			return float64(next.DistanceTo(foot2))
		}
		p1 = next
	}
	foot2, ok := projectIntoTrim(f2, p1)
	if !ok {
		return stdmath.Inf(1)
	}
	return float64(p1.DistanceTo(foot2))
}

// projectIntoTrim projects p onto face f's surface and returns the foot only when it lies inside the
// face's trim; otherwise ok is false and the interior descent is abandoned.
func projectIntoTrim(f *topo.Face, p math.Point3) (math.Point3, bool) {
	_, _, foot := geom.ClosestPointOnSurface(f.Geometry(), p)
	if !PointInFaceTrim(f, foot) {
		return math.Point3{}, false
	}
	return foot, true
}

// footConverged reports whether two successive feet coincide to the relative floor — the alternating
// projection has reached a stationary pair.
func footConverged(a, b math.Point3) bool {
	move := float64(a.DistanceTo(b))
	return move <= distConvergeRel*(1+float64(b.AsVector().Length()))
}

// faceParamBox is the (u, v) box the face's trim actually occupies: the parameter extent of its
// boundary edges, so the interior seed grid covers the trimmed region even on an unbounded surface
// (a plane's footprint). A boundaryless face (whole sphere/torus) falls back to the surface's own
// parameter domain, which is bounded or periodic there. ok is false only when no bounded box exists.
func faceParamBox(f *topo.Face) (u0, u1, v0, v1 float64, ok bool) {
	box, filled := edgeParamExtent(f)
	if filled {
		return box[0], box[1], box[2], box[3], true
	}
	return surfaceParamDomain(f.Geometry())
}

// edgeParamExtent walks the face's boundary edges, inverts sampled edge points to the surface
// parameters, and returns the (u0,u1,v0,v1) bounds. filled is false for a boundaryless face.
func edgeParamExtent(f *topo.Face) (box [4]float64, filled bool) {
	surf := f.Geometry()
	box = [4]float64{stdmath.Inf(1), stdmath.Inf(-1), stdmath.Inf(1), stdmath.Inf(-1)}
	for _, e := range f.Edges() {
		lo, hi := e.Geometry().Domain()
		for i := 0; i <= distScanSamples; i++ {
			t := lo + (hi-lo)*float64(i)/float64(distScanSamples)
			u, v := surf.ParamAt(e.Geometry().PointAt(t))
			box[0], box[1] = stdmath.Min(box[0], u), stdmath.Max(box[1], u)
			box[2], box[3] = stdmath.Min(box[2], v), stdmath.Max(box[3], v)
			filled = true
		}
	}
	return box, filled
}

// surfaceParamDomain returns the surface's own parameter domain as a seed box, false when a direction
// is unbounded (a naked plane with no trim cannot be seeded and never reaches this path in practice).
func surfaceParamDomain(s geom.Surface) (u0, u1, v0, v1 float64, ok bool) {
	u0, u1 = s.UDomain()
	v0, v1 = s.VDomain()
	if stdmath.IsInf(u0, 0) || stdmath.IsInf(u1, 0) || stdmath.IsInf(v0, 0) || stdmath.IsInf(v1, 0) {
		return 0, 0, 0, 0, false
	}
	return u0, u1, v0, v1, true
}
