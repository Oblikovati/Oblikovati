// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The SPAN-AREA criterion spliceCornerBiteChain picks the bitten corner with.
//
// WHAT THE PICK MEANS. A bite chain cuts a host face's ring into two spans; the splice REMOVES the span
// bounding the smaller region on that host and keeps the other. "Smaller" is only well defined as a
// surface AREA on the host.
//
// WHY A NEWELL AREA IS NOT THAT AREA. |newellNormal|/2 is the area of the ring's PROJECTION onto its own
// best-fit plane. On a planar host that IS the surface area, exactly. On a curved host it is the shadow:
// always short, and short by a factor each span sets for ITSELF — a span wrapping a full quarter of a
// cylinder loses 1 − 2/π ≈ 36 % of its developed width, one wrapping a few degrees loses almost nothing.
// Two spans therefore shrink by DIFFERENT factors, so the projected comparison can invert one the true
// areas decide the other way (falsified verbatim in fillet_chain_span_area_test.go). complex/D8's two
// curved hosts measure the distortion at 5.9 % / 9.7 %, carried today only by a 6.9× pick margin.
//
// THE CRITERION. The region's true area in the face's OWN metric chart, by Green's theorem:
//
//	A = ∫∫_D √(EG−F²) du dv = ∮_{∂D} M(u,v) dv,   M(u,v) = ∫_{u₀}^{u} |P_u × P_v|(t,v) dt
//
// (√(EG−F²) = |P_u × P_v| by Lagrange's identity, so no first-form assembly is needed). The contour is
// the chart image of the SAME sampled point ring the Newell form used, so the two criteria differ only
// in the metric, never in the sampling.
//
// WHERE IT IS EXACT. On a plane it is the Newell area, taken verbatim (below), so no planar pick moves.
// On a cylinder and on a cone |P_u × P_v| does not depend on u, which makes the inner integrand constant
// and M a polynomial of degree ≤ 2 along a chart edge — inside both quadrature rules' exactness degree,
// so the value is exact to rounding. Those are the developable hosts, where "developed area" is a
// theorem rather than an approximation. On a sphere, torus or NURBS host M is transcendental along an
// edge and the outer rule is a Gauss–Legendre approximation converging spectrally in the sample count;
// it is still the true metric, never the shadow.

// spanAreaGaussX/spanAreaGaussW are the 4-point Gauss–Legendre nodes and weights on [0,1] (exact through
// degree 7). Used for both the contour integral along a chart edge and the potential's inner sweep.
var (
	spanAreaGaussX = [4]float64{0.06943184420297371, 0.33000947820757187, 0.6699905217924281, 0.9305681557970262}
	spanAreaGaussW = [4]float64{0.1739274225687269, 0.3260725774312731, 0.3260725774312731, 0.1739274225687269}
)

// spanAreaUPanels is the number of composite panels the potential's inner u-sweep is split into. One
// panel is already exact on every u-independent metric (plane, cylinder, cone, sphere, torus); the
// panels buy accuracy only on a host whose metric varies ALONG u, i.e. a general NURBS patch.
const spanAreaUPanels = 8

// uvPoint is a point in a face's own parameter chart, with periodic directions UNWRAPPED (so a ring
// crossing a cylinder's 2π seam stays a connected polygon instead of leaping the whole domain).
type uvPoint struct{ u, v float64 }

// developedSpanArea is the true area the sampled point ring encloses ON surf — the quantity
// spliceCornerBiteChain's smaller-span pick actually means. A planar host (or an unknown one) falls back
// to the Newell area, which is exact there; every other host is measured in its own metric chart.
//
//	developedSpanArea(round, ring) // 462.84 on complex/D8's bitten corner, vs 435.53 projected
func developedSpanArea(surf geom.Surface, ring []math.Point3) float64 {
	if len(ring) < 3 {
		return 0
	}
	if _, planar := surf.(geom.Plane); planar || surf == nil {
		return float64(newellNormal(ring).Length()) / 2
	}
	return metricChartArea(surf, unwrappedParamRing(surf, ring))
}

// unwrappedParamRing maps a 3D point ring into surf's parameter chart, unwrapping each periodic
// direction against its predecessor so consecutive samples never jump a whole period at the seam.
func unwrappedParamRing(surf geom.Surface, ring []math.Point3) []uvPoint {
	uPeriod, vPeriod := wrapPeriod(surf.UDomain()), wrapPeriod(surf.VDomain())
	out := make([]uvPoint, 0, len(ring))
	for _, p := range ring {
		u, v := surf.ParamAt(p)
		if len(out) > 0 {
			prev := out[len(out)-1]
			u, v = unwrapParam(u, prev.u, uPeriod), unwrapParam(v, prev.v, vPeriod)
		}
		out = append(out, uvPoint{u, v})
	}
	return out
}

// wrapPeriod returns a parameter direction's wrap period, or 0 when it does not wrap. The Surface
// contract (kernel/geom/surface.go) fixes [0, 2π] as the domain of every direction that runs AROUND an
// axis, and that is exactly the set ParamAt wraps its answer into.
func wrapPeriod(lo, hi float64) float64 {
	if lo == 0 && stdmath.Abs(hi-2*stdmath.Pi) <= 1e-12 {
		return 2 * stdmath.Pi
	}
	return 0
}

// unwrapParam shifts x by whole periods so it lands within half a period of ref — the standard phase
// tessellate.Unwrap, applied per sample so a ring's chart image stays connected across the seam.
func unwrapParam(x, ref, period float64) float64 {
	if period <= 0 {
		return x
	}
	return x - period*stdmath.Round((x-ref)/period)
}

// metricChartArea is the surface area bounded by a closed polygon in surf's parameter chart, evaluated
// as the Green's-theorem contour integral ∮ M dv of the metric potential (see the file header). The sign
// carries the polygon's orientation, which the criterion does not care about, so the magnitude is
// returned.
func metricChartArea(surf geom.Surface, poly []uvPoint) float64 {
	if len(poly) < 3 {
		return 0
	}
	uRef, sum := poly[0].u, 0.0
	for i, p := range poly {
		q := poly[(i+1)%len(poly)]
		if q.v == p.v {
			continue // dv = 0 contributes exactly zero — and a sampled rim arc is hundreds of such edges
		}
		sum += (q.v - p.v) * meanPotentialAlongEdge(surf, p, q, uRef)
	}
	return stdmath.Abs(sum)
}

// meanPotentialAlongEdge is ∫₀¹ M(u(s), v(s)) ds along the straight chart edge p→q — the outer rule of
// metricChartArea's contour integral.
func meanPotentialAlongEdge(surf geom.Surface, p, q uvPoint, uRef float64) float64 {
	var sum float64
	for k, s := range spanAreaGaussX {
		u, v := p.u+s*(q.u-p.u), p.v+s*(q.v-p.v)
		sum += spanAreaGaussW[k] * metricPotential(surf, u, v, uRef)
	}
	return sum
}

// metricPotential is M(u,v) = ∫_{uRef}^{u} |P_u × P_v|(t, v) dt, the u-antiderivative of the area
// element at fixed v, by composite Gauss–Legendre over spanAreaUPanels panels.
func metricPotential(surf geom.Surface, u, v, uRef float64) float64 {
	h := (u - uRef) / spanAreaUPanels
	var sum float64
	for panel := range spanAreaUPanels {
		lo := uRef + float64(panel)*h
		for k, s := range spanAreaGaussX {
			du, dv := surf.DerivativesAt(lo+s*h, v)
			sum += spanAreaGaussW[k] * float64(du.Cross(dv).Length())
		}
	}
	return sum * h
}
