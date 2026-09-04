// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
)

// The POLE contour of a cap (M48/C3, ADR-0062).
//
// A cap — a sphere face bounded by one latitude circle — could not be integrated at all. Its single
// rim wraps u and returns to its own v, so greenFormFor picks the ∮ −P du reduction with the
// antiderivative's lower limit at minLoopV, which IS the rim's own latitude: the integrand vanishes
// identically on the only contour there is, and the region came out as zero. A hemisphere of radius 5
// declined where the whole sphere measured 314.1593 exactly, so every body with a cap in it was
// measured by TESSELLATION instead.
//
// The contour is incomplete, not the reduction. A cap's boundary in the parameter rectangle is its rim
// AND the line at the pole, which is one point in space but a full 2π of parameter. OCCT stores that
// line explicitly, as a degenerate edge carrying a pcurve, precisely so a cap's wire is a closed
// contour; sphereFaceUV already adds the same segment when it charts a sphere for the boolean. Here it
// is supplied to the integral.
//
// It cannot come through edgeGreen: the pole's curve is a point, and duvdtAt reads no direction from a
// degenerate parametrization. Its contribution is the boundary form evaluated along the pole line
// directly, which is an ordinary one-dimensional quadrature in u.

// capPoleContour is the ∮ −P du contribution of the line at the pole that closes a cap's contour, and
// the far limit it runs at. ok=false when the face is not a cap: more than one rim (a band, whose two
// rims close each other), no rim at all, or a reduction this form does not apply to.
func capPoleContour[T quadTerms[T]](s geom.Surface, loops []faceLoop, form greenAxis, at pointEval[T]) (T, bool) {
	var zero T
	rim, ok := soleWrappingRim(loops)
	if !ok || form.dv {
		return zero, false
	}
	vHi, ok := polarVLimit(s)
	if !ok {
		return zero, false
	}
	// The FAR limit, always the high one, so the enclosed region is a region of the surface rather
	// than a choice: which of the two caps the FACE is stays faceHoldsEnclosedRegion's question.
	uStart := rim.edges[0].samples[0].u
	span := -rim.netU // the pole line runs back the way the rim came, closing the contour
	along := func(u float64) T { return integrateV(at, form.base, vHi, u).scale(-1) }
	return integrateSeeded(along, uStart, uStart+span, fullDomainSeed), true
}

// polarVLimit is the surface's high v limit when the surface DEGENERATES there — a sphere's pole, where
// the whole azimuth is one point. A torus's v limit is a period, not a pole: it closes onto itself, so
// there is no line to close a contour with and this declines. The question is asked of the geometry
// rather than of the surface kind.
func polarVLimit(s geom.Surface) (float64, bool) {
	vLo, vHi := s.VDomain()
	uLo, uHi := s.UDomain()
	if !allFinite(vLo, vHi, uLo, uHi) || vHi <= vLo {
		return 0, false
	}
	a, b := s.PointAt(uLo, vHi), s.PointAt(uLo+(uHi-uLo)/3, vHi)
	c := s.PointAt(uLo+2*(uHi-uLo)/3, vHi)
	span := float64(s.PointAt(uLo, vLo).DistanceTo(s.PointAt(uLo, vHi)))
	tol := polarCollapseRel * stdmath.Max(span, 1)
	if float64(a.DistanceTo(b)) > tol || float64(a.DistanceTo(c)) > tol {
		return 0, false // the v limit is a circle, not a point
	}
	return vHi, true
}

// polarCollapseRel is how small the spread of the v-limit's own points must be, relative to the
// surface's v extent, for that limit to BE a point.
const polarCollapseRel = 1e-9 // tol:parametric — pole collapse, relative to the surface's v extent

// soleWrappingRim returns the face's one seam-wrapping loop, when it has exactly one and that loop
// travels in u. A band has two and needs no pole; a contractible face has none.
func soleWrappingRim(loops []faceLoop) (faceLoop, bool) {
	if wrappingLoopCount(loops) != 1 {
		return faceLoop{}, false
	}
	for _, fl := range loops {
		if !loopWraps(fl) {
			continue
		}
		if closeUV(fl.netU, 0, 0, 0) || !closeUV(fl.netV, 0, 0, 0) {
			return faceLoop{}, false // it wraps v, or both: not a latitude rim
		}
		if len(fl.edges) == 0 || len(fl.edges[0].samples) == 0 {
			return faceLoop{}, false
		}
		return fl, true
	}
	return faceLoop{}, false
}

// capInteriorUV is the probe point for a cap: between its rim and the pole the contour closes at, at
// the rim's own start azimuth. It names the same region capPoleContour measured, which is what lets
// faceHoldsEnclosedRegion compare the FACE against it.
func capInteriorUV(s geom.Surface, loops []faceLoop) (u, v float64, ok bool) {
	rim, isCap := soleWrappingRim(loops)
	if !isCap {
		return 0, 0, false
	}
	vHi, ok := polarVLimit(s)
	if !ok {
		return 0, 0, false
	}
	first := rim.edges[0].samples[0]
	rimV := first.v
	if stdmath.Abs(vHi-rimV) <= 0 {
		return 0, 0, false // the rim is AT the pole: no interior between them
	}
	return first.u, (rimV + vHi) / 2, true
}
