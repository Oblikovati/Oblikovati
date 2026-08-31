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
// a periodic cylinder/cone/sphere/torus without a per-type branch. The engine is GENERIC over the
// term type: the same quadrature and Green reduction integrate both the divergence-flux mass terms
// (volume/inertia) and the area-weighted surface moments the congruence signature needs (#3449),
// so there is one adaptive engine, not two.

const (
	quadOrder      = 8     // Gauss points per adaptive cell — exact to degree 15
	quadDepth      = 24    // max adaptive-subdivision depth
	quadRelTol     = 1e-11 // tol:numeric — adaptive-quadrature relative convergence (dimensionless)
	quadAbsTol     = 1e-13 // tol:numeric — adaptive-quadrature absolute-floor convergence
	edgeUVSamples  = 48    // points sampled along each edge to reconstruct its uv boundary curve
	fullDomainSeed = 4     // initial cells per axis for a boundary-less face's full-domain integral
)

// quadTerms is any component-wise accumulator the adaptive engine can integrate: it adds, scales,
// and reports when a coarse estimate agrees with its refinement. Both massTerms (divergence flux)
// and areaTerms (surface moments) satisfy it, so one engine serves both families (#3449).
type quadTerms[T any] interface {
	add(T) T
	scale(float64) T
	converged(T) bool
}

// pointEval maps one surface parameter point to its integrand contribution — integrandsAt for the
// flux mass terms, areaIntegrandsAt for the surface moments.
type pointEval[T any] func(u, v float64) T

// faceRegion integrates a face's region terms: a boundary-less face over its full parameter
// rectangle, a bounded face by Green's theorem over its uv loops. Both use `at` as the integrand.
func faceRegion[T quadTerms[T]](f *topo.Face, at pointEval[T]) (T, bool) {
	if len(f.Loops()) == 0 {
		return fullDomainTerms(f.Geometry(), at)
	}
	return greenTerms(f, at)
}

// fullDomainTerms integrates g over a boundary-less face's entire finite parameter rectangle by
// a nested adaptive rule (outer in v of an inner adaptive integral in u). An unbounded domain
// cannot be integrated this way and declines (ok=false).
func fullDomainTerms[T quadTerms[T]](s geom.Surface, at pointEval[T]) (T, bool) {
	var zero T
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	if !allFinite(uLo, uHi, vLo, vHi) {
		return zero, false
	}
	outer := func(v float64) T {
		return integrateU(at, uLo, uHi, v)
	}
	return integrateSeeded(outer, vLo, vHi, fullDomainSeed), true
}

// integrateU adaptively integrates every surface integrand along the line at fixed v, u∈[a,b].
func integrateU[T quadTerms[T]](at pointEval[T], a, b, v float64) T {
	eval := func(u float64) T { return at(u, v) }
	return integrateAdaptive(eval, a, b, quadDepth)
}

// integrateV adaptively integrates every surface integrand along the line at fixed u, v∈[a,b].
func integrateV[T quadTerms[T]](at pointEval[T], a, b, u float64) T {
	eval := func(v float64) T { return at(u, v) }
	return integrateAdaptive(eval, a, b, quadDepth)
}

// integrateSeeded splits [a,b] into `cells` equal pieces and adaptively integrates each — the
// seed spreads the first refinement so a smooth periodic integrand is resolved from the start.
func integrateSeeded[T quadTerms[T]](eval func(float64) T, a, b float64, cells int) T {
	if cells < 1 {
		cells = 1
	}
	h := (b - a) / float64(cells)
	var acc T
	for c := range cells {
		acc = acc.add(integrateAdaptive(eval, a+float64(c)*h, a+float64(c+1)*h, quadDepth))
	}
	return acc
}

// integrateAdaptive integrates a term-valued function over [a,b], subdividing until a single Gauss
// cell agrees with its two-cell refinement within tolerance (or the depth cap).
func integrateAdaptive[T quadTerms[T]](eval func(float64) T, a, b float64, depth int) T {
	whole := gaussCellTerms(eval, a, b)
	mid := (a + b) / 2
	refined := gaussCellTerms(eval, a, mid).add(gaussCellTerms(eval, mid, b))
	if depth <= 0 || whole.converged(refined) {
		return refined
	}
	return integrateAdaptive(eval, a, mid, depth-1).add(integrateAdaptive(eval, mid, b, depth-1))
}

// gaussCellTerms applies the fixed-order Gauss–Legendre rule to one cell.
func gaussCellTerms[T quadTerms[T]](eval func(float64) T, a, b float64) T {
	nodes, weights := geom.GaussLegendre(quadOrder)
	half, mid := (b-a)/2, (a+b)/2
	var acc T
	for i, x := range nodes {
		acc = acc.add(eval(mid + half*x).scale(weights[i]))
	}
	return acc.scale(half)
}

// greenTerms integrates a bounded face by Green's theorem over its uv loops. Each loop edge is
// integrated ALONG ITS TRUE CURVE (Gauss in the edge parameter, not chord-wise), so a circular
// edge on a planar cap contributes exactly rather than as an inscribed polygon.
//
// Each loop is oriented for the positive-measure region by loopRegionSigns, so the result is
// ∫∫_D g over the true region on the +normal side; the caller then applies (or, for the unsigned
// surface moments, omits) the outward sense so a hole/inner wall contributes correctly.
func greenTerms[T quadTerms[T]](f *topo.Face, at pointEval[T]) (T, bool) {
	var zero T
	s := f.Geometry()
	loops, ok := buildFaceLoops(s, f)
	if !ok {
		return zero, false
	}
	form, ok := greenFormFor(loops)
	if !ok {
		return zero, false
	}
	signs := loopRegionSigns(loops)
	var enclosed T
	for i, fl := range loops {
		var lt T
		for _, le := range fl.edges {
			lt = lt.add(edgeGreen(s, le, form, at))
		}
		enclosed = enclosed.add(lt.scale(signs[i]))
	}
	return faceSideOfRegion(f, loops, enclosed, at)
}

// faceSideOfRegion returns the face's own terms given the terms of the region its loops ENCLOSE.
// The two differ only on a closed surface, where the face may be the complement of that region (see
// analytic_face_region.go); the complement's terms are the whole surface's minus the enclosed
// region's, never the enclosed region's negation.
func faceSideOfRegion[T quadTerms[T]](f *topo.Face, loops []faceLoop, enclosed T, at pointEval[T]) (T, bool) {
	full, closed := fullDomainTerms(f.Geometry(), at)
	if !closed {
		return enclosed, true // an OPEN surface's complement is unbounded: the bounded side is the face
	}
	holds, certain := faceHoldsEnclosedRegion(f, loops)
	if !certain {
		var zero T
		return zero, false // the side cannot be certified: decline rather than integrate a guess
	}
	if holds {
		return enclosed, true
	}
	return full.add(enclosed.scale(-1)), true
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

// faceLoop is a face boundary loop reconstructed in parameter space. It carries no outer/hole
// flag: loopRegionSigns derives the role from the loops' own nesting, because topo.Loop.IsOuter is
// not well defined on a closed surface. netU/netV are how far the unwrapped walk travelled overall —
// zero on a loop that closes in the plane, a whole number of periods on one that wraps a seam.
type faceLoop struct {
	edges      []loopEdge
	netU, netV float64
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
// periodic coordinates continuously across edges. It records the walk's NET travel so greenTerms can
// pick the Green form the loop admits; a loop that closes in neither parameter is refused there.
func buildLoopEdges(s geom.Surface, l *topo.Loop, uPeriod, vPeriod float64) (faceLoop, bool) {
	var fl faceLoop
	pu, pv, have := 0.0, 0.0, false
	for _, use := range l.EdgeUses() {
		le, ok := sampleLoopEdge(s, use, uPeriod, vPeriod, &pu, &pv, &have)
		if !ok {
			return faceLoop{}, false
		}
		fl.edges = append(fl.edges, le)
	}
	if len(fl.edges) == 0 || len(fl.edges[0].samples) == 0 {
		return faceLoop{}, false
	}
	first := fl.edges[0].samples[0]
	fl.netU, fl.netV = pu-first.u, pv-first.v
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

// edgeGreen returns the boundary integral along one edge, summed over its reference segments; each
// segment is integrated on the true curve with a fixed Gauss rule (the reference is dense enough
// that a segment is a smooth, seam-free arc).
func edgeGreen[T quadTerms[T]](s geom.Surface, le loopEdge, form greenAxis, at pointEval[T]) T {
	var acc T
	for k := 0; k+1 < len(le.samples); k++ {
		a, b := le.samples[k], le.samples[k+1]
		if a.t == b.t {
			continue
		}
		acc = acc.add(segmentGreen(s, le, a, b, form, at))
	}
	return acc
}

// segmentGreen integrates the chosen boundary form over one edge segment [a.t, b.t] on the true
// curve. Each node's u, v come from ParamAt, unwrapped to the linearly-interpolated reference so the
// periodic branch is right.
func segmentGreen[T quadTerms[T]](s geom.Surface, le loopEdge, a, b arcSample, form greenAxis, at pointEval[T]) T {
	nodes, weights := geom.GaussLegendre(6)
	half, mid := (b.t-a.t)/2, (a.t+b.t)/2
	var acc T
	for i, x := range nodes {
		t := mid + half*x
		frac := (t - a.t) / (b.t - a.t)
		n := greenNode{t: t, span: b.t - a.t, seedU: a.u + (b.u-a.u)*frac, seedV: a.v + (b.v-a.v)*frac}
		n.u, n.v = uvOnCurve(s, le, t, n.seedU, n.seedV)
		acc = acc.add(greenIntegrand(s, le, form, at, n).scale(weights[i]))
	}
	return acc.scale(half)
}

// greenNode is one Gauss node on an edge: its curve parameter and segment span, the unwrapped
// surface parameters there, and the seeds the derivative estimates unwrap against.
type greenNode struct{ t, span, u, v, seedU, seedV float64 }

// greenIntegrand is one node's contribution: Q(u,v)·v′ for the u-antiderivative form, or −P(u,v)·u′
// for the v-antiderivative one. The two are the conjugate halves of the same identity — ∫∫ g du dv
// equals ∮ Q dv and equals −∮ P du — so a face uses whichever its loops can close.
func greenIntegrand[T quadTerms[T]](s geom.Surface, le loopEdge, form greenAxis, at pointEval[T], n greenNode) T {
	if form.dv {
		return integrateU(at, form.base, n.u, n.v).scale(dvdt(s, le, n.t, n.span, n.seedV))
	}
	return integrateV(at, form.base, n.v, n.u).scale(-dudt(s, le, n.t, n.span, n.seedU))
}

// greenAxis names which conjugate reduction a face's loops carry, with the antiderivative's lower
// limit: u0 for the ∮ Q dv form, v0 for the ∮ −P du one.
type greenAxis struct {
	dv   bool
	base float64
}

// greenFormFor picks the reduction the loops admit. ∮ Q dv needs every loop to come back to its
// starting u, ∮ −P du needs every loop to come back to its starting v. A BAND on a periodic surface
// — the wall of a drilled bore, a zone on a torus — closes in only one of them: its walk crosses the
// seam a whole number of times, so the other coordinate ends a period or more away and that
// reduction cannot close (Oblikovati/Oblikovati#3453 — the analytic integral used to decline the
// most ordinary body in CAD, a plate with a hole in it). The u-form is preferred, so every face that
// closes both ways integrates exactly as it did before this split.
func greenFormFor(loops []faceLoop) (greenAxis, bool) {
	if loopsReturnTo(loops, func(fl faceLoop) float64 { return fl.netU }) {
		return greenAxis{dv: true, base: minLoopU(loops)}, true
	}
	if loopsReturnTo(loops, func(fl faceLoop) float64 { return fl.netV }) {
		return greenAxis{dv: false, base: minLoopV(loops)}, true
	}
	return greenAxis{}, false
}

// loopsReturnTo reports whether every loop's net travel in the coordinate `net` selects is zero.
func loopsReturnTo(loops []faceLoop, net func(faceLoop) float64) bool {
	for _, fl := range loops {
		if !closeUV(net(fl), 0, 0, 0) {
			return false
		}
	}
	return true
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

// dudt is dvdt for the u-parameter — the rate of the surface u-parameter along the curve at t.
func dudt(s geom.Surface, le loopEdge, t, span, seedU float64) float64 {
	d := stdmath.Abs(span) * 1e-4 // tol:numeric — central-difference step as a fraction of the param span
	if d == 0 {
		return 0
	}
	upRaw, _ := s.ParamAt(le.curve.PointAt(t + d))
	umRaw, _ := s.ParamAt(le.curve.PointAt(t - d))
	up := unwrapPeriodic(upRaw, seedU, le.uPeriod)
	um := unwrapPeriodic(umRaw, seedU, le.uPeriod)
	return (up - um) / (2 * d)
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

// minLoopV is minLoopU for the v-coordinate — the shared lower limit v0 of the ∮ −P du form.
func minLoopV(loops []faceLoop) float64 {
	min := stdmath.Inf(1)
	for _, fl := range loops {
		for _, le := range fl.edges {
			for _, sp := range le.samples {
				if sp.v < min {
					min = sp.v
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
