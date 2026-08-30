// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Per-face analytic integration for mass properties (M48/C3). Two topological paths, selected
// by whether the face is trimmed — never by surface kind:
//
//   - a boundary-less face (whole sphere/torus) integrates over its full parameter rectangle;
//   - a bounded face reduces each area/volume/moment integrand g to a boundary line integral
//     ∫∫_D g du dv = ∮_∂D Q dv, Q(u,v)=∫_{u0}^{u} g(s,v) ds (Green's theorem), evaluated over
//     the face's uv loops. This is exact for planar faces (the integrands are polynomial) and
//     spectrally accurate for the analytic curved surfaces.
//
// Every 1-D integral is adaptive Gauss–Legendre so the same code serves a polynomial plane and
// a periodic cylinder/cone/sphere/torus without a per-type branch.

const (
	quadOrder      = 8     // Gauss points per adaptive cell — exact to degree 15
	quadDepth      = 24    // max adaptive-subdivision depth
	quadRelTol     = 1e-11 // tol:numeric — adaptive-quadrature relative convergence (dimensionless)
	quadAbsTol     = 1e-13 // tol:numeric — adaptive-quadrature absolute-floor convergence
	edgeUVSamples  = 48    // points sampled along each edge to reconstruct its uv boundary curve
	fullDomainSeed = 4     // initial cells per axis for a boundary-less face's full-domain integral
)

// fullDomainTerms integrates g over a boundary-less face's entire finite parameter rectangle by
// a nested adaptive rule (outer in v of an inner adaptive integral in u). An unbounded domain
// cannot be integrated this way and declines (ok=false).
func fullDomainTerms(s geom.Surface) (massTerms, bool) {
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	if !allFinite(uLo, uHi, vLo, vHi) {
		return massTerms{}, false
	}
	outer := func(v float64) massTerms {
		return integrateU(s, uLo, uHi, v)
	}
	return integrateSeeded(outer, vLo, vHi, fullDomainSeed), true
}

// integrateU adaptively integrates every surface integrand along the line at fixed v, u∈[a,b].
func integrateU(s geom.Surface, a, b, v float64) massTerms {
	eval := func(u float64) massTerms { return integrandsAt(s, u, v) }
	return integrateAdaptive(eval, a, b, quadDepth)
}

// integrateSeeded splits [a,b] into `cells` equal pieces and adaptively integrates each — the
// seed spreads the first refinement so a smooth periodic integrand is resolved from the start.
func integrateSeeded(eval func(float64) massTerms, a, b float64, cells int) massTerms {
	if cells < 1 {
		cells = 1
	}
	h := (b - a) / float64(cells)
	var acc massTerms
	for c := range cells {
		acc = acc.add(integrateAdaptive(eval, a+float64(c)*h, a+float64(c+1)*h, quadDepth))
	}
	return acc
}

// integrateAdaptive integrates a massTerms-valued function over [a,b], subdividing until a
// single Gauss cell agrees with its two-cell refinement within tolerance (or the depth cap).
func integrateAdaptive(eval func(float64) massTerms, a, b float64, depth int) massTerms {
	whole := gaussCellTerms(eval, a, b)
	mid := (a + b) / 2
	refined := gaussCellTerms(eval, a, mid).add(gaussCellTerms(eval, mid, b))
	if depth <= 0 || termsConverged(whole, refined) {
		return refined
	}
	return integrateAdaptive(eval, a, mid, depth-1).add(integrateAdaptive(eval, mid, b, depth-1))
}

// gaussCellTerms applies the fixed-order Gauss–Legendre rule to one cell.
func gaussCellTerms(eval func(float64) massTerms, a, b float64) massTerms {
	nodes, weights := geom.GaussLegendre(quadOrder)
	half, mid := (b-a)/2, (a+b)/2
	var acc massTerms
	for i, x := range nodes {
		acc = acc.add(eval(mid + half*x).scale(weights[i]))
	}
	return acc.scale(half)
}

// termsConverged reports whether the coarse and refined estimates agree to tolerance on every
// component (mixed absolute/relative, so a component that is legitimately ~0 does not stall).
func termsConverged(coarse, refined massTerms) bool {
	c := [11]float64{coarse.vol, coarse.mx, coarse.my, coarse.mz, coarse.cxx, coarse.cyy, coarse.czz, coarse.cxy, coarse.cyz, coarse.czx, coarse.area}
	r := [11]float64{refined.vol, refined.mx, refined.my, refined.mz, refined.cxx, refined.cyy, refined.czz, refined.cxy, refined.cyz, refined.czx, refined.area}
	for i := range c {
		if stdmath.Abs(c[i]-r[i]) > quadAbsTol+quadRelTol*stdmath.Abs(r[i]) {
			return false
		}
	}
	return true
}

// greenTerms integrates a bounded face by Green's theorem over its uv loops. Each loop edge is
// integrated ALONG ITS TRUE CURVE (Gauss in the edge parameter, not chord-wise), so a circular
// edge on a planar cap contributes exactly rather than as an inscribed polygon. The region is
// oriented positive by the outer loop's uv shoelace sign, so the enclosed measure is positive
// regardless of the stored loop orientation or the surface's uv handedness.
func greenTerms(f *topo.Face) (massTerms, bool) {
	s := f.Geometry()
	loops, ok := buildFaceLoops(s, f)
	if !ok {
		return massTerms{}, false
	}
	u0 := minLoopU(loops)
	sigma := outerOrientation(loops)
	var total massTerms
	for _, fl := range loops {
		for _, le := range fl.edges {
			total = total.add(edgeGreen(s, le, u0))
		}
	}
	return total.scale(sigma), true
}

// arcSample is one point on an edge: its curve parameter t and the face-surface (u, v) it maps
// to, globally unwrapped so the sample sequence is continuous across a periodic seam.
type arcSample struct{ t, u, v float64 }

// loopEdge is one edge of a face loop with its 3-D curve and a reference sample sequence used to
// seed periodic unwrapping during the true-curve integration.
type loopEdge struct {
	curve   geom.Curve3
	samples []arcSample
	uPeriod float64
	vPeriod float64
}

// faceLoop is a face boundary loop reconstructed in parameter space.
type faceLoop struct {
	edges []loopEdge
	outer bool
}

// buildFaceLoops reconstructs every loop of a bounded face as continuous-in-uv edge sequences.
func buildFaceLoops(s geom.Surface, f *topo.Face) ([]faceLoop, bool) {
	uPeriod := periodOf(s.UDomain())
	vPeriod := periodOf(s.VDomain())
	out := make([]faceLoop, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		fl, ok := buildLoopEdges(s, l, uPeriod, vPeriod)
		if !ok {
			return nil, false
		}
		out = append(out, fl)
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// buildLoopEdges walks a loop's edge uses in order and samples each edge's curve, inverting every
// sample onto the surface (ParamAt is exact on-surface for the analytic surfaces) and unwrapping
// periodic coordinates continuously across edges.
func buildLoopEdges(s geom.Surface, l *topo.Loop, uPeriod, vPeriod float64) (faceLoop, bool) {
	fl := faceLoop{outer: l.IsOuter()}
	pu, pv, have := 0.0, 0.0, false
	for _, use := range l.EdgeUses() {
		le, ok := sampleLoopEdge(s, use, uPeriod, vPeriod, &pu, &pv, &have)
		if !ok {
			return faceLoop{}, false
		}
		fl.edges = append(fl.edges, le)
	}
	return fl, true
}

// sampleLoopEdge samples one edge use in traversal order, inverting each point onto the surface
// and unwrapping it continuously from the running (*pu, *pv) cursor shared across the loop.
func sampleLoopEdge(s geom.Surface, use *topo.EdgeUse, uPeriod, vPeriod float64, pu, pv *float64, have *bool) (loopEdge, bool) {
	c := use.Edge().Geometry()
	lo, hi := c.Domain()
	if !allFinite(lo, hi) {
		return loopEdge{}, false // an unbounded edge cannot bound a finite trim
	}
	t0, t1 := lo, hi
	if use.Reversed() {
		t0, t1 = hi, lo
	}
	le := loopEdge{curve: c, uPeriod: uPeriod, vPeriod: vPeriod}
	for k := 0; k <= edgeUVSamples; k++ {
		t := t0 + (t1-t0)*float64(k)/float64(edgeUVSamples)
		u, v := s.ParamAt(c.PointAt(t))
		if *have {
			u = unwrapPeriodic(u, *pu, uPeriod)
			v = unwrapPeriodic(v, *pv, vPeriod)
		}
		le.samples = append(le.samples, arcSample{t: t, u: u, v: v})
		*pu, *pv, *have = u, v, true
	}
	return le, true
}

// edgeGreen returns ∮ Q dv along one edge, summed over its reference segments; each segment is
// integrated on the true curve with a fixed Gauss rule (the reference is dense enough that a
// segment is a smooth, seam-free arc).
func edgeGreen(s geom.Surface, le loopEdge, u0 float64) massTerms {
	var acc massTerms
	for k := 0; k+1 < len(le.samples); k++ {
		a, b := le.samples[k], le.samples[k+1]
		if a.t == b.t {
			continue
		}
		acc = acc.add(segmentGreen(s, le, a, b, u0))
	}
	return acc
}

// segmentGreen integrates ∫ Q(u(t),v(t))·v′(t) dt over one edge segment [a.t, b.t] on the true
// curve. u, v come from ParamAt (unwrapped to the linearly-interpolated reference so the branch
// is right), and v′ is a central difference of the surface v-parameter along the curve.
func segmentGreen(s geom.Surface, le loopEdge, a, b arcSample, u0 float64) massTerms {
	nodes, weights := geom.GaussLegendre(6)
	half, mid := (b.t-a.t)/2, (a.t+b.t)/2
	var acc massTerms
	for i, x := range nodes {
		t := mid + half*x
		frac := (t - a.t) / (b.t - a.t)
		seedU := a.u + (b.u-a.u)*frac
		seedV := a.v + (b.v-a.v)*frac
		u, v := uvOnCurve(s, le, t, seedU, seedV)
		vp := dvdt(s, le, t, b.t-a.t, seedV)
		q := integrateU(s, u0, u, v)
		acc = acc.add(q.scale(weights[i] * vp))
	}
	return acc.scale(half)
}

// uvOnCurve inverts the curve point at t onto the surface, choosing the periodic branch nearest
// the interpolated reference (u, v) seed.
func uvOnCurve(s geom.Surface, le loopEdge, t, seedU, seedV float64) (u, v float64) {
	ru, rv := s.ParamAt(le.curve.PointAt(t))
	return unwrapPeriodic(ru, seedU, le.uPeriod), unwrapPeriodic(rv, seedV, le.vPeriod)
}

// dvdt is the rate of the surface v-parameter along the curve at t, by central difference (δ a
// small fraction of the segment span), each sample unwrapped to the seed so no seam jump leaks in.
func dvdt(s geom.Surface, le loopEdge, t, span, seedV float64) float64 {
	d := stdmath.Abs(span) * 1e-4 // tol:numeric — central-difference step as a fraction of the param span
	if d == 0 {
		return 0
	}
	_, vpRaw := s.ParamAt(le.curve.PointAt(t + d))
	_, vmRaw := s.ParamAt(le.curve.PointAt(t - d))
	vp := unwrapPeriodic(vpRaw, seedV, le.vPeriod)
	vm := unwrapPeriodic(vmRaw, seedV, le.vPeriod)
	return (vp - vm) / (2 * d)
}

// outerOrientation returns +1 if the outer loop is already CCW in uv, −1 otherwise, so the
// region integral comes out with positive area measure.
func outerOrientation(loops []faceLoop) float64 {
	for _, fl := range loops {
		if fl.outer {
			if loopSignedArea(fl) < 0 {
				return -1
			}
			return 1
		}
	}
	return 1
}

// loopSignedArea is the shoelace signed area of a loop's reference polyline (positive when CCW).
func loopSignedArea(fl faceLoop) float64 {
	pts := loopPolyline(fl)
	var a float64
	for i := range pts {
		p, q := pts[i], pts[(i+1)%len(pts)]
		a += p.u*q.v - q.u*p.v
	}
	return a / 2
}

// loopPolyline flattens a loop's edge samples into one uv polyline (drops shared-vertex repeats).
func loopPolyline(fl faceLoop) []arcSample {
	var pts []arcSample
	for _, le := range fl.edges {
		for _, sp := range le.samples {
			if n := len(pts); n > 0 && closeUV(pts[n-1].u, pts[n-1].v, sp.u, sp.v) {
				continue
			}
			pts = append(pts, sp)
		}
	}
	return pts
}

// minLoopU is the smallest u over all loop samples — the shared lower limit u0 for the inner
// antiderivative, kept finite by taking it from the (bounded) trim rather than the surface.
func minLoopU(loops []faceLoop) float64 {
	min := stdmath.Inf(1)
	for _, fl := range loops {
		for _, le := range fl.edges {
			for _, sp := range le.samples {
				if sp.u < min {
					min = sp.u
				}
			}
		}
	}
	return min
}

// periodOf reports a coordinate's period: 2π for a direction that wraps the axis (domain
// exactly [0, 2π]), 0 for a non-periodic direction (which is never unwrapped).
func periodOf(lo, hi float64) float64 {
	if lo == 0 && stdmath.Abs(hi-2*stdmath.Pi) < 1e-9 { // tol:parametric — periodic-domain span compare
		return 2 * stdmath.Pi
	}
	return 0
}

// unwrap returns the period-multiple of raw nearest prev, giving a continuous coordinate across
// a periodic seam. A zero period (non-periodic) passes raw through unchanged.
func unwrapPeriodic(raw, prev, period float64) float64 {
	if period == 0 {
		return raw
	}
	return raw + period*stdmath.Round((prev-raw)/period)
}

// closeUV reports whether two uv points coincide to a tight parameter tolerance.
func closeUV(u1, v1, u2, v2 float64) bool {
	return stdmath.Abs(u1-u2) < 1e-9 && stdmath.Abs(v1-v2) < 1e-9 // tol:parametric — uv coincidence
}

// allFinite reports whether every value is finite (guards an unbounded parameter rectangle).
func allFinite(xs ...float64) bool {
	for _, x := range xs {
		if stdmath.IsInf(x, 0) || stdmath.IsNaN(x) {
			return false
		}
	}
	return true
}
