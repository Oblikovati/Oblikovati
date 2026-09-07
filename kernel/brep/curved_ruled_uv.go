// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Ruled-side split in PARAMETER SPACE (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375). The OCCT-style
// arrangement approach for a full periodic ruled side (a cone frustum side OR a cylinder side) cut by a
// plane: instead of hand-building the kept loop and deriving its winding per topological case, split the
// side in its own (u, v) = (azimuth, axial-distance) space. A ruled side is straight along v, so the
// signed distance is LINEAR in v:
//
//	g(u, v) = a(u) + v·b(u),   a(u) = p + q·cos(u − u_n),   b(u) = s + t·cos(u − u_n)
//
// because the surface point is P(u,v) = base + v·â + rad(v)·r̂(u) with rad(v) = radSlope·v + radConst,
// so g = n·(base − O) + v·n·â + rad(v)·(n·r̂(u)) and n·r̂(u) = |n_r|·cos(u − u_n). A CONE has rad(v)=v·tanα
// (radConst 0 → q 0, t=tanα·|n_r|); a CYLINDER has rad(v)=R (radSlope 0 → t 0, q=R·|n_r|). The section is
// the single-valued v(u) = −a(u)/b(u) and the kept region {g<0} is a v-INTERVAL [lo(u), hi(u)] per u —
// every arrangement (arc-band, vertex-inside, oblique, within-band, clips-rim, tongue, the cylinder's
// axis-parallel flat) is one of two boundary walks: WRAPPING (the interval is non-empty at every azimuth
// → a two-loop band) or NON-WRAPPING (it empties over part of the seam → a single span), with the
// surface's own orientation inherited, no per-case winding. This is the sole splitter for both ruled
// sides; the bespoke cone and cylinder split families it replaced are gone.

// ruledUV is a periodic ruled side (cone or cylinder) expressed in (u, v): the surface frame and the
// linear coefficients of the signed distance g(u,v) = (p + q·cos(u−uN)) + v·(s + t·cos(u−uN)).
type ruledUV struct {
	base               math.Point3 // surface point at v=0 on the axis (cone apex, cylinder bottom centre)
	axis, ref, binor   math.Vector3
	radSlope, radConst float64 // rad(v) = radSlope·v + radConst (cone: tanα, 0; cylinder: 0, R)
	band               coneSideBand_
	p, q, s, t, uN     float64
	// seamU rotates the (u,v) parameter origin for the arrangement trim only: the artificial azimuth seam
	// (u=0≡2π) is moved to absolute azimuth seamU so it falls clear of the imprint's rim crossings (a
	// section arm grazing the seam otherwise breaks the arrangement, #1405). paramOf reports u relative to
	// seamU; point3/aU/bU add it back. The analytic walk leaves it 0, so its parameterisation is unchanged.
	seamU float64

	// General curved∩curved mode (#1403): when solidMode is set the side is cut not by a plane half-space
	// but by another SOLID's SSI imprint, so a band point is kept by 3D solid membership (keptBySolid) —
	// keep(op, isB, insideOther(point)) — instead of the linear plane predicate g(u,v)<0. The plane
	// coefficients (p,q,s,t,uN) are then unused; the geometry frame (base/axis/radSlope/…) drives point3.
	// insideOther is the other solid's point-membership oracle (analytic per primitive — a cone/cylinder
	// solid's faces are curved, so the planar ray-cast insideSolid does not apply).
	solidMode   bool
	solidOp     Op
	solidIsB    bool
	insideOther func(math.Point3) bool

	// pinched marks the equal-radius Steinmetz case (#1403): the SSI imprint self-intersects at the two
	// pinch points, so the kept region is two lobes that meet at those points, NOT a simple wrapping band.
	// The recogniser sets it authoritatively (it knows the imprint is the two crossing ellipses), so
	// wrapsAllU returns false — the lobes touch every azimuth but never form a band — and the boundary is
	// grouped into separate lobe faces. Equivalent to a net-Δu winding test on the traced loops, decided a
	// priori from the recogniser instead of rediscovered.
	pinched bool

	// seamHint pins the arrangement's artificial azimuth seam (overriding chooseSeamU) when the recogniser
	// knows the clear placement. The pinched Steinmetz imprint covers every azimuth, so chooseSeamU finds no
	// gap and would drop the seam near a pinch (shattering the degree-4 vertex); the recogniser sets the hint
	// to a lobe centre instead — π/2 from both pinches, splitting one lobe harmlessly across a seam the weld
	// rejoins (#1403).
	seamHint    float64
	hasSeamHint bool
}

// ruledUV satisfies uvSide: a singly-periodic surface whose v is the bounded axial band (#1406).
var _ uvSide = (*ruledUV)(nil)

// placeSeams moves the arrangement's artificial azimuth seam clear of the imprint (uvSide). A ruled side
// has only the one azimuth seam; v is bounded, so there is no tube seam to place.
func (c *ruledUV) placeSeams(imprint []geom.Curve3) { c.seamU = c.chooseSeamU(imprint) }

// vPeriodic reports that a ruled side's v (axial distance) does NOT wrap — only u does (uvSide).
func (c ruledUV) vPeriodic() bool { return false }

// uPeriodic reports that a ruled side's u (azimuth) DOES wrap (u=0≡2π), so the boundary welder folds the
// seam (uvSide). cutCylinderUV inherits this through embedding (#1591).
func (c ruledUV) uPeriodic() bool { return true }

// multiFace reports whether the kept region may be disconnected (uvSide): only the general curved∩curved
// cut (solidMode) can leave several faces (the two lens caps a rod punches in a fat cone); a plane half-space
// always leaves one connected region (#1403).
func (c ruledUV) multiFace() bool { return c.solidMode }

// assembleSegments samples the imprint and adds the rim+seam frame the arrangement subdivides (uvSide).
// For the pinched (Steinmetz) case each open arc is unwrapped per-arc (unwrapArcSegs) so an arc that merely
// TOUCHES the azimuth seam at a pinch endpoint is not fragmented by splitSeamCrossing — without it the
// wrapping-side lobe's arcs split at the seam and the lobe leaks into the surrounding region (#1403).
func (c ruledUV) assembleSegments(imprint []geom.Curve3) []uvSeg {
	var imp []uvSeg
	for _, cv := range imprint {
		segs := c.sampleImprintUV(cv)
		if c.pinched {
			segs = unwrapArcSegs(segs)
		}
		imp = append(imp, segs...)
	}
	return c.assembleBandSegments(imp)
}

// finalizeLoops drops the degenerate apex-pole rim loop from a kept cone face (uvSide; see dropApexLoop).
func (c ruledUV) finalizeLoops(loops []curvedLoop) []curvedLoop { return c.dropApexLoop(loops) }

// ruledMaterial wraps a ruled side's half-space predicate as a uvSide materialOf builder: a closure (not a
// bound method value) so the predicate reads the receiver AFTER trimByImprint has shifted its seam. A method
// value would freeze the pre-seam receiver and misclassify cells against an unshifted frame.
func ruledMaterial(c *ruledUV) func() materialPredicate {
	return func() materialPredicate { return c.halfSpaceMaterial() }
}

// aU returns a(u) = p + q·cos(u−uN), the v-independent part of the signed distance g(u,v)=a(u)+v·b(u). u is
// relative to the seam origin (seamU), so the absolute azimuth used against uN is u+seamU.
func (c ruledUV) aU(u float64) float64 { return c.p + c.q*stdmath.Cos(u+c.seamU-c.uN) }

// bU returns b(u) = s + t·cos(u−uN), the coefficient of v in the signed distance g(u,v)=a(u)+v·b(u).
func (c ruledUV) bU(u float64) float64 { return c.s + c.t*stdmath.Cos(u+c.seamU-c.uN) }

// vPinchTol is the axial-distance margin below which a kept interval counts as PINCHED (empty). A tongue
// pinches where the section meets a clamp rim (lo≈hi); when that azimuth lands exactly on a sample (a
// symmetric cut puts the pinch on u=0/π/2π) the section value equals the rim to within rounding, so a
// strict lo<hi flickers and breaks the span pairing. The margin makes the pinch read as empty either way.
// It sits well ABOVE the ~1e-14 rounding flicker yet two orders below the 1e-7 weld tolerance, so the span
// endpoint the bisection lands on (where the section sits vPinchTol inside the rim) still welds to the rim.
//
// tol:calibrated — this (u,v) arrangement margin is OCC-validated for the delicate tangent-limit cases
// (the parabola cone∩box, the symmetric pinch). It is NOT model-relativised under #1399: at a part scale
// of ~10 a size-scaled weld is an order of magnitude off and misclassifies the rim crossing, breaking the
// validated volumes. The split operates on the cone's own (azimuth, axial) frame, which is already ~O(R).
const vPinchTol = 1e-9

// keptV returns the kept (g<0) axial-distance interval [lo, hi] at azimuth u, clamped to the band, plus
// whether it is non-empty (thicker than vPinchTol). g=a(u)+v·b(u) is linear in v: when b>0 the kept side
// is v<v(u), when b<0 it is v>v(u), and when b≈0 the whole ruling is kept (a<0) or dropped (a≥0).
func (c ruledUV) keptV(u float64) (lo, hi float64, ok bool) {
	a, b := c.aU(u), c.bU(u)
	switch {
	case b > 1e-12:
		hi = c.band.vMax
		if v := -a / b; v < hi {
			hi = v
		}
		return c.band.vMin, hi, hi > c.band.vMin+vPinchTol
	case b < -1e-12:
		lo = c.band.vMin
		if v := -a / b; v > lo {
			lo = v
		}
		return lo, c.band.vMax, lo < c.band.vMax-vPinchTol
	default:
		return c.band.vMin, c.band.vMax, a < 0 // the plane is parallel to the ruling: whole column kept or dropped
	}
}

// point3 returns the surface point at (u, v): base + v·â + (radSlope·v+radConst)·r̂(u). u is relative to the
// seam origin (seamU), so the absolute azimuth on the surface frame is u+seamU.
func (c ruledUV) point3(u, v float64) math.Point3 {
	a := u + c.seamU
	radial := c.ref.Scale(math.Scalar(stdmath.Cos(a))).Add(c.binor.Scale(math.Scalar(stdmath.Sin(a))))
	rad := c.radSlope*v + c.radConst
	return c.base.TranslateBy(c.axis.Scale(math.Scalar(v))).TranslateBy(radial.Scale(math.Scalar(rad)))
}

// seamOrigin is the surface parameter of the chart's (0,0): a ruled side rotates only its azimuth
// origin, and its v IS the surface's own axial parameter (uvSide, ADR-0063).
func (c ruledUV) seamOrigin() math.Point2 { return math.P2(c.seamU, 0) }

// unwrapArcSegs makes one open imprint arc's (u,v) sampling CONTINUOUS in u — removing the 2π jump at a
// pinch endpoint that paramOf's [0,2π) branch introduces — and shifts the whole arc by whole turns so its
// mean azimuth lands in [0,2π). An arc running pinch-to-pinch then stays within one azimuth half (its
// endpoint sits exactly on the seam, 0 or 2π, rather than wrapping across it), so splitSeamCrossing leaves
// it whole and the lobe it bounds seals (#1403). Used only for the pinched Steinmetz arcs, whose endpoints
// touch the seam; the ordinary closed-loop imprints keep the raw branch for splitSeamCrossing to resolve.
func unwrapArcSegs(segs []uvSeg) []uvSeg {
	if len(segs) == 0 {
		return segs
	}
	us := make([]float64, len(segs)+1)
	us[0] = float64(segs[0].a.X)
	for i, s := range segs {
		us[i+1] = unwrapAzimuthNear(us[i], float64(s.b.X))
	}
	mean := 0.0
	for _, u := range us {
		mean += u
	}
	shift := turnsToCanonical(mean / float64(len(us)))
	out := make([]uvSeg, len(segs))
	for i, s := range segs {
		s.a = math.P2(math.Scalar(us[i]+shift), s.a.Y)
		s.b = math.P2(math.Scalar(us[i+1]+shift), s.b.Y)
		out[i] = s
	}
	return out
}

// turnsToCanonical returns the whole-turn shift (a multiple of 2π) that brings u into [0, 2π).
func turnsToCanonical(u float64) float64 {
	shift := 0.0
	for u+shift < 0 {
		shift += 2 * stdmath.Pi
	}
	for u+shift >= 2*stdmath.Pi {
		shift -= 2 * stdmath.Pi
	}
	return shift
}
