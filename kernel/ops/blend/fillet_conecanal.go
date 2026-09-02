// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cone-host RULING-edge arm — the cone-host trihedral-corner campaign, Slice CN2 (the crux;
// cone-host-corner-derivation.md §2 "Arm B"). Rounding the convex edge where a radial plane (one that
// CONTAINS the cone axis) meets the host cone is NOT a torus: the r-offset plane is parallel to the axis,
// so the ball-centre locus {offset plane} ∩ {offset cone A′=A±r/sinα·â} is a HYPERBOLA branch, and the
// constant-radius canal over it is a genuine canal surface (the first non-analytic — BSpline — arm through
// the fillet engine, what N7's "T9" was dropped for). It is built EXACTLY, with no SSI marching and no
// BSpline approximation (user directive 2026-07-19): closed-form hyperbola stations, exact plane/cone
// characteristic-circle feet, lofted homogeneously via the shared geom canal stack (geom.LoftCanalStations).
//
// Handles BOTH host material sides (A2 wave, re-landed from a reverted session — see coneCanalSpine.
// apexSign): convex-external (material INSIDE the cone, A′ = A + r/sinα·â) and the concave BORE (material
// OUTSIDE, A′ = A − r/sinα·â — simple/I4's ruling). A bore ruling edge is edge-CONVEX (the ball rolls in
// the material exactly as on a boss), mirroring the sign the CAP-plane arm (coneArmFillet) already carries
// — the same insight the I1 cap-plane bore case greened. Derived from first principles (ball tangent to
// the cone from the OUTWARD normal n̂_out = −sinα·â + cosα·ê_r instead of the boss's INWARD
// n̂_in = sinα·â − cosα·ê_r) and cross-checked by re-deriving the shipped convex ζ(x_f) formula
// bit-for-bit from that same construction (geometry-math-advisor consultation, A2 wave).
//
// The corner solve (CN3/torus-corner) declines the SPHERE-corner cases at "corner face must be planar" —
// which fires BEFORE arm building (computeCorners precedes computeFillets) — for hosts that campaign has
// not yet built; I4's cone-host corner (CN5, R4 wave) already handles the concave sign, so I4 is exercised
// through a completed fillet, not just direct construction/tests.

const (
	// canalArmStationsMin floors the ADAPTIVE station count: the refinement starts here and doubles
	// (min, 2·, 4·, …) until the measured BETWEEN-station envelope error is ≤ the model-relative bound.
	// Each column is EXACT on the true envelope; the count controls only the cubic v-interpolation BETWEEN
	// stations. A smooth low-curvature arm (C2/C6) resolves within a doubling or two; the D1 snout — whose
	// hyperbola-vertex curvature κ = cotα/r blows up — needs several (CN2 review: the old bare 24 left an
	// uncontrolled 2.1e-2 gap at v≈0.995, right where CN5 must land the snout cap bit-exactly).
	canalArmStationsMin = 24
	// canalArmStationsMax caps the doubling. If the envelope error is still over bound here the spine is
	// genuinely unresolvable (coneArmRulingUnresolved) — honest-reject with the offending error, never loft
	// a band with a known gap (do-no-harm). D1 (the worst corpus case) resolves at 384, well inside the cap.
	canalArmStationsMax = 512
	// canalCurvatureWeightSamples discretizes the ∫κ^¼ cumulative used to PLACE the stations denser near the
	// hyperbola vertex (κ^¼ equalizes the O(h⁴) cubic-loft position error κ·h⁴ across intervals). It samples
	// the weight, not a tolerance — refinement stops on the MEASURED error, not on this count.
	canalCurvatureWeightSamples = 2000
	// canalCurvatureWeightExp is the ¼ power: station density ∝ κ^¼ equalizes the O(h⁴) cubic-interp error.
	canalCurvatureWeightExp = 0.25
	// canalSpineSearchIters is the golden-section iteration budget for the exact-spine closest-point the
	// envelope-error measure needs (per mid-interval sample; a small bracket converges to machine precision).
	canalSpineSearchIters = 64
	// canalFoldArcSamples is how many points the band-arc fold guard tests per station (from the plane
	// foot to the cone foot). The fold lobe of a canal over a high-curvature hyperbola vertex is on the
	// concave (down-axis) side the band never enters, so sampling the band arc alone is enough.
	canalFoldArcSamples = 8
	// canalSpanBand is k in the degenerate-span guard |x_f,hi − x_f,lo| < k·res.Weld(): a ruling whose
	// fittable hyperbola span collapses has no arm to loft.
	canalSpanBand = 4
	// canalSlerpMinAngle floors the great-arc interpolation angle (radians, dimensionless on unit
	// directions): below it the two feet directions coincide and slerp returns the first unchanged.
	canalSlerpMinAngle = 1e-9 // tol:angular — coincident foot-direction guard
)

// coneCanalSpine is the exact ball-centre HYPERBOLA of a convex-external Cone∧Plane ruling-edge fillet.
// In the r-offset-plane frame {axis â, in-plane transverse ê = â×n̂, plane material-outward normal n̂}
// the spine is m(x_f) = A + ζ(x_f)·â + x_f·ê − r·n̂ with ζ(x_f) = r/sinα + √(x_f²+r²)/tanα — PARAMETRIZED
// BY x_f (the free in-plane coordinate), never by the axial height ζ, whose dx_f/dζ→∞ at the hyperbola
// vertex bunches stations at the snout (D1; cone-host-corner-derivation.md §"Spine parametrization").
type coneCanalSpine struct {
	apex              math.Point3
	axis, ePerp, nOut math.Vector3 // {â, ê, n̂}: unit axis, unit in-plane transverse, unit plane material-outward normal
	radius            float64
	sinA, cosA, tanA  float64
	// apexSign is the material-side sign the CAP-plane torus arm (coneArmSurface) already carries: +1
	// convex-external (material INSIDE the cone, offset apex A′ = A + r/sinα·â) or −1 concave BORE
	// (material OUTSIDE, A′ = A − r/sinα·â). Re-landed from the reverted ring-machinery session (wave
	// round4-armlayer.md): derived from first principles (a ball tangent to the cone from the bore side,
	// offset along the OUTWARD normal n̂_out = −sinα·â + cosα·ê_r instead of the boss's INWARD
	// n̂_in = sinα·â − cosα·ê_r) and cross-checked by reproducing the shipped convex ζ(x_f) formula
	// bit-for-bit from that derivation. Only the ζ apex-shift term and the endpoint-inversion's r·cosα
	// term carry the sign; coneFoot's projection direction ĝ, planeFoot, rhoAt and the curvature formulas
	// are UNCHANGED (ĝ is perpendicular to both n̂_in and n̂_out by construction, and the curvature only
	// depends on ρ(x_f), whose apexSign-independent form cancels the constant ζ shift in ζ′, ζ″).
	apexSign float64
}

// newConeCanalSpine builds the offset-plane frame and the trigonometric constants of the spine, or
// reports a near-cylinder (sinα below band: apex shift r/sinα blows up — a true cylinder host takes M5)
// or near-plane (cosα below band) cone, or a degenerate frame (plane normal parallel to the axis, i.e.
// not actually a ruling — unreachable once classifyConeArm has passed). Bands are model-relative (ADR-0042).
// apexSign is the material-side sign (+1 convex-external / −1 concave bore, see coneCanalSpine.apexSign).
func newConeCanalSpine(co geom.Cone, nOut math.UnitVector3, apexSign, r float64, res tol.Resolution) (coneCanalSpine, coneArmReject) {
	sinA, cosA := stdmath.Sincos(co.HalfAngle)
	aband := coneAlphaBandCoef * res.Weld() / stdmath.Max(res.Size(), res.Weld())
	if sinA < aband {
		return coneCanalSpine{}, coneArmNearCylinder
	}
	if cosA < aband {
		return coneCanalSpine{}, coneArmNearPlane
	}
	axis := co.AxisDir.AsVector()
	ePerp, err := math.UnitVector3FromVector(axis.Cross(nOut.AsVector()))
	if err != nil {
		return coneCanalSpine{}, coneArmDegenerate // plane normal ∥ axis: not a ruling
	}
	return coneCanalSpine{
		apex: co.Apex, axis: axis, ePerp: ePerp.AsVector(), nOut: nOut.AsVector(),
		radius: r, sinA: sinA, cosA: cosA, tanA: sinA / cosA, apexSign: apexSign,
	}, coneArmBuilt
}

// rhoAt is the ball-centre's perpendicular distance to the axis at station x_f: √(x_f²+r²) — the SAME
// form for both the convex and concave-bore hosts (apexSign-independent: derived below).
func (s coneCanalSpine) rhoAt(xf float64) float64 { return stdmath.Hypot(xf, s.radius) }

// zetaAt is the ball-centre's axial height above the apex A at station x_f: apexSign·r/sinα + ρ/tanα
// (+r/sinα convex-external, −r/sinα concave bore — see coneCanalSpine.apexSign).
func (s coneCanalSpine) zetaAt(xf float64) float64 {
	return s.apexSign*s.radius/s.sinA + s.rhoAt(xf)/s.tanA
}

// center is the exact ball-centre m(x_f) on the hyperbola spine (in the r-offset plane, distance r from
// the radial plane on the material side).
func (s coneCanalSpine) center(xf float64) math.Point3 {
	off := s.axis.Scale(s.zetaAt(xf)).Add(s.ePerp.Scale(xf)).Sub(s.nOut.Scale(s.radius))
	return s.apex.TranslateBy(off)
}

// planeFoot is the radial-plane contact of the ball at m: f_P = m + r·n̂ (n̂ ⊥ the spine plane, so f_P is
// automatically on the characteristic circle). It lies exactly on the plane and at distance r from m.
func (s coneCanalSpine) planeFoot(m math.Point3) math.Point3 {
	return m.TranslateBy(s.nOut.Scale(s.radius))
}

// coneFoot is the meridian ruling contact of the ball at m: T = A + (w·ĝ)·ĝ with ĝ = cosα·â + sinα·ê_r
// (ê_r the outward radial direction of m). |T − m| = r EXACTLY on the hyperbola (proved in the
// derivation: ζ·sinα − ρ·cosα = r). ok is false only if m is on the axis (no meridian) — impossible on
// the spine, where ρ ≥ r > 0.
func (s coneCanalSpine) coneFoot(m math.Point3) (math.Point3, bool) {
	w := s.apex.VectorTo(m)
	h := w.Dot(s.axis)
	wperp := w.Sub(s.axis.Scale(h))
	rho := float64(wperp.Length())
	if rho == 0 {
		return math.Point3{}, false
	}
	er := wperp.Scale(1 / rho)
	g := s.axis.Scale(s.cosA).Add(er.Scale(s.sinA))
	return s.apex.TranslateBy(g.Scale(w.Dot(g))), true
}

// curvatureVector is the spine's curvature vector κ·N̂ at x_f (units 1/length), pointing to the concave
// side. The hyperbola is planar (in the offset plane), so with m'(x_f)=ζ'·â+ê and m”(x_f)=ζ”·â it is
// K = ζ”/(1+ζ'²)²·(â − ζ'·ê), ζ'=x_f/(ρ·tanα), ζ”=r²/(ρ³·tanα). At the vertex κ = 1/(r·tanα) (the
// semi-latus rectum), the source of the full-tube fold the band-arc guard must scope out.
func (s coneCanalSpine) curvatureVector(xf float64) math.Vector3 {
	rho := s.rhoAt(xf)
	zp := xf / (rho * s.tanA)
	zpp := s.radius * s.radius / (rho * rho * rho * s.tanA)
	scale := zpp / ((1 + zp*zp) * (1 + zp*zp))
	return s.axis.Sub(s.ePerp.Scale(zp)).Scale(scale)
}

// stationOf is the closed-form station of a corner centre C that lies ON the hyperbola spine: x_f =
// (C − A)·ê. ok is false when C is off the spine — its plane-normal offset is not −r (off the r-offset
// plane) or its axial height disagrees with ζ(x_f) — by more than weld·scale. The corner weld (CN4)
// reads this to place the great-circle setback; it is the canal case of the arm-station machinery.
func (s coneCanalSpine) stationOf(c math.Point3, scale, weld float64) (float64, bool) {
	w := s.apex.VectorTo(c)
	xf := float64(w.Dot(s.ePerp))
	if stdmath.Abs(float64(w.Dot(s.nOut))+s.radius) > weld*scale {
		return 0, false // C not on the r-offset plane
	}
	if stdmath.Abs(float64(w.Dot(s.axis))-s.zetaAt(xf)) > weld*scale {
		return 0, false // C's axial height disagrees with the hyperbola
	}
	return xf, true
}

// buildConeCanalArm builds the canal BSplineSurface of a convex-external Cone∧Plane ruling edge and the
// spine descriptor the weld needs, or a cause-named reject. nOut is the plane's material-outward unit
// normal (fixes the offset direction). Every station column is EXACT (closed-form hyperbola centre + exact
// plane/cone feet) and the band arc is fold-guarded; the only inter-station approximation is the cubic
// v-interpolation, whose envelope error resolveStations ADAPTIVELY refines to ≤ res.Weld() (curvature-weighted
// station placement + doubling, CN2 review) — NO marching, NO uncontrolled gap.
func buildConeCanalArm(e *topo.Edge, co geom.Cone, nOut math.UnitVector3, apexSign, r float64, res tol.Resolution) (geom.BSplineSurface, coneCanalSpine, coneArmReject) {
	spine, reason := newConeCanalSpine(co, nOut, apexSign, r, res)
	if reason != coneArmBuilt {
		return geom.BSplineSurface{}, coneCanalSpine{}, reason
	}
	lo, hi, reason := spine.edgeXfSpan(e, res)
	if reason != coneArmBuilt {
		return geom.BSplineSurface{}, coneCanalSpine{}, reason
	}
	_, surf, reason := spine.resolveStations(lo, hi, res)
	if reason != coneArmBuilt {
		return geom.BSplineSurface{}, coneCanalSpine{}, reason
	}
	return surf, spine, coneArmBuilt
}

// edgeXfSpan maps the picked ruling edge's two endpoints to their x_f stations (via the cone-foot axial
// height h_T = cotα·(r·cosα + ρ), inverted), clamping an endpoint past the hyperbola vertex (ball does
// not fit) to the vertex x_f=0 — the D1 snout terminus. It rejects when neither endpoint admits the ball
// (coneArmRulingNoFit) or the fittable span collapses (coneArmRulingSpan). The x_f sign follows the
// edge's side of the axis plane so the arm sits on the picked ruling.
func (s coneCanalSpine) edgeXfSpan(e *topo.Edge, res tol.Resolution) (lo, hi float64, reason coneArmReject) {
	mid := edgeMidpoint(e)
	sgn := 1.0
	if float64(s.apex.VectorTo(mid).Dot(s.ePerp)) < 0 {
		sgn = -1
	}
	x0, ok0 := s.xfAtEndpoint(e.StartVertex().Point())
	x1, ok1 := s.xfAtEndpoint(e.EndVertex().Point())
	if !ok0 && !ok1 {
		return 0, 0, coneArmRulingNoFit
	}
	lo, hi = sortedSpan(sgn*x0, sgn*x1)
	if hi-lo < canalSpanBand*res.Weld() {
		return 0, 0, coneArmRulingSpan
	}
	return lo, hi, coneArmBuilt
}

// xfAtEndpoint inverts the cone-foot axial height to the ball-centre station x_f for an edge endpoint at
// axial height h above the apex: ρ = h·tanα − apexSign·r·cosα (the ball's axis distance; the CONVEX
// tangent point sits on the near/apex side of the ball centre, −r·cosα, while the CONCAVE-bore tangent
// point sits on the far side, +r·cosα — apexSign carries the flip, see coneCanalSpine.apexSign),
// x_f = √(ρ² − r²). fits is false when ρ < r (the endpoint is past the vertex, the ball cannot fit
// there) — the caller clamps that end to the vertex x_f = 0.
func (s coneCanalSpine) xfAtEndpoint(p math.Point3) (float64, bool) {
	h := float64(s.apex.VectorTo(p).Dot(s.axis))
	rhoNeed := h*s.tanA - s.apexSign*s.radius*s.cosA
	if rhoNeed < s.radius {
		return 0, false
	}
	return stdmath.Sqrt(rhoNeed*rhoNeed - s.radius*s.radius), true
}

// sortedSpan returns the ordered pair (min, max) of a and b.
func sortedSpan(a, b float64) (lo, hi float64) {
	return stdmath.Min(a, b), stdmath.Max(a, b)
}
