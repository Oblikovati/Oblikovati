// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cone-host RULING-edge arm — the cone-host trihedral-corner campaign, Slice CN2 (the crux;
// cone-host-corner-derivation.md §2 "Arm B"). Rounding the convex edge where a radial plane (one that
// CONTAINS the cone axis) meets the host cone is NOT a torus: the r-offset plane is parallel to the axis,
// so the ball-centre locus {offset plane} ∩ {offset cone A′=A+r/sinα·â} is a HYPERBOLA branch, and the
// constant-radius canal over it is a genuine canal surface (the first non-analytic — BSpline — arm through
// the fillet engine, what N7's "T9" was dropped for). It is built EXACTLY, with no SSI marching and no
// BSpline approximation (user directive 2026-07-19): closed-form hyperbola stations, exact plane/cone
// characteristic-circle feet, lofted homogeneously via the shared geom canal stack (geom.LoftCanalStations).
// The corner solve (CN3) still declines these cone cases at "corner face must be planar" — which fires
// BEFORE arm building (computeCorners precedes computeFillets) — so CN2 greens nothing and the arm is
// exercised only by direct construction/tests, never yet through a completed fillet.

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
}

// newConeCanalSpine builds the offset-plane frame and the trigonometric constants of the spine, or
// reports a near-cylinder (sinα below band: apex shift r/sinα blows up — a true cylinder host takes M5)
// or near-plane (cosα below band) cone, or a degenerate frame (plane normal parallel to the axis, i.e.
// not actually a ruling — unreachable once classifyConeArm has passed). Bands are model-relative (ADR-0042).
func newConeCanalSpine(co geom.Cone, nOut math.UnitVector3, r float64, res Resolution) (coneCanalSpine, coneArmReject) {
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
		radius: r, sinA: sinA, cosA: cosA, tanA: sinA / cosA,
	}, coneArmBuilt
}

// rhoAt is the ball-centre's perpendicular distance to the axis at station x_f: √(x_f²+r²).
func (s coneCanalSpine) rhoAt(xf float64) float64 { return stdmath.Hypot(xf, s.radius) }

// zetaAt is the ball-centre's axial height above the apex A at station x_f: r/sinα + ρ/tanα.
func (s coneCanalSpine) zetaAt(xf float64) float64 { return s.radius/s.sinA + s.rhoAt(xf)/s.tanA }

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
func buildConeCanalArm(e *topo.Edge, co geom.Cone, nOut math.UnitVector3, r float64, res Resolution) (geom.BSplineSurface, coneCanalSpine, coneArmReject) {
	spine, reason := newConeCanalSpine(co, nOut, r, res)
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
func (s coneCanalSpine) edgeXfSpan(e *topo.Edge, res Resolution) (lo, hi float64, reason coneArmReject) {
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
// axial height h above the apex: ρ = h·tanα − r·cosα (the ball's axis distance), x_f = √(ρ² − r²). fits
// is false when ρ < r (the endpoint is past the vertex, the ball cannot fit there) — the caller clamps
// that end to the vertex x_f = 0.
func (s coneCanalSpine) xfAtEndpoint(p math.Point3) (float64, bool) {
	h := float64(s.apex.VectorTo(p).Dot(s.axis))
	rhoNeed := h*s.tanA - s.radius*s.cosA
	if rhoNeed < s.radius {
		return 0, false
	}
	return stdmath.Sqrt(rhoNeed*rhoNeed - s.radius*s.radius), true
}

// sortedSpan returns the ordered pair (min, max) of a and b.
func sortedSpan(a, b float64) (lo, hi float64) {
	return stdmath.Min(a, b), stdmath.Max(a, b)
}

// canalStations bundles a chosen station set: the node coordinates x_f and their EXACT columns (ball
// centre + plane/cone feet). It is what resolveStations produces, the loft consumes, and the envelope-error
// measure re-reads — kept together so the node x_f (needed to bracket the closest-spine search) travels
// with the columns.
type canalStations struct {
	xfs                   []float64
	centers, feetA, feetB []math.Point3
}

// resolveStations chooses the station density that BOUNDS the between-station envelope error to the
// model-relative res.Weld() — the same weld the arm's at-station exactness is measured to (ADR-0042). It
// starts at canalArmStationsMin and doubles, each round placing curvature-weighted stations, lofting them,
// and MEASURING the actual mid-interval envelope error (maxEnvelopeError), until the error is within bound —
// returning the winning stations and their surface. If the cap canalArmStationsMax is reached still over
// bound the spine is genuinely unresolvable (coneArmRulingUnresolved), honest-rejected rather than lofted
// with a known gap (CN2 review replaced the old bare 24 that left D1's snout 2.1e-2 off the true envelope).
func (s coneCanalSpine) resolveStations(lo, hi float64, res Resolution) (canalStations, geom.BSplineSurface, coneArmReject) {
	for n := canalArmStationsMin; n <= canalArmStationsMax; n *= 2 {
		st, reason := s.stationsAt(lo, hi, n)
		if reason != coneArmBuilt {
			return canalStations{}, geom.BSplineSurface{}, reason
		}
		surf, err := geom.LoftCanalStations(st.centers, st.feetA, st.feetB, s.radius, res.Weld())
		if err != nil {
			return canalStations{}, geom.BSplineSurface{}, coneArmDegenerate
		}
		if s.maxEnvelopeError(surf, st) <= res.Weld() {
			return st, surf, coneArmBuilt
		}
	}
	return canalStations{}, geom.BSplineSurface{}, coneArmRulingUnresolved
}

// stationsAt builds the n+1 curvature-weighted stations across [lo, hi]: exact hyperbola centre and exact
// plane/cone feet per node, guarding each station's BAND ARC (plane foot → cone foot) against a canal
// self-intersection (fold). A cone foot on the axis or a folded band arc rejects (do-no-harm: never loft a
// malformed band). Nodes cluster near the hyperbola vertex (curvatureWeightedXf) rather than uniformly.
func (s coneCanalSpine) stationsAt(lo, hi float64, n int) (canalStations, coneArmReject) {
	xfs := s.curvatureWeightedXf(lo, hi, n)
	st := canalStations{xfs: xfs}
	st.centers = make([]math.Point3, n+1)
	st.feetA = make([]math.Point3, n+1)
	st.feetB = make([]math.Point3, n+1)
	for i, xf := range xfs {
		m := s.center(xf)
		coneT, ok := s.coneFoot(m)
		if !ok {
			return canalStations{}, coneArmDegenerate
		}
		fP := s.planeFoot(m)
		if s.bandArcMinRegularity(xf, m, fP, coneT) <= 0 {
			return canalStations{}, coneArmRulingFold
		}
		st.centers[i], st.feetA[i], st.feetB[i] = m, fP, coneT
	}
	return st, coneArmBuilt
}

// curvatureWeightedXf places the n+1 station nodes over [lo, hi] with density ∝ κ(x_f)^¼, so the cubic
// loft's O(h⁴) position error κ·h⁴ is equalized across intervals — clustering nodes at the hyperbola vertex
// (κ = cotα/r maximal) instead of wasting them on the smooth far span, so a modest count resolves the D1
// snout. Reduces to uniform where κ is constant. Nodes are the inverse-cumulative-weight images of an even
// partition (an arc-length-style remap of x_f).
func (s coneCanalSpine) curvatureWeightedXf(lo, hi float64, n int) []float64 {
	cum := s.curvatureWeightCumulative(lo, hi)
	total := cum[canalCurvatureWeightSamples]
	span := float64(canalCurvatureWeightSamples)
	out := make([]float64, n+1)
	for j := 0; j <= n; j++ {
		target := total * float64(j) / float64(n)
		out[j] = lo + (hi-lo)*invertCumulative(cum, target)/span
	}
	return out
}

// curvatureWeightCumulative is the running trapezoidal integral of κ(x_f)^¼ over
// canalCurvatureWeightSamples+1 uniform samples of [lo, hi]. κ = |curvatureVector| > 0 for every finite x_f
// (ζ” > 0 on the hyperbola), so the cumulative is strictly increasing and invertible — no floor needed.
func (s coneCanalSpine) curvatureWeightCumulative(lo, hi float64) []float64 {
	cum := make([]float64, canalCurvatureWeightSamples+1)
	prev := s.curvatureWeight(lo)
	for i := 1; i <= canalCurvatureWeightSamples; i++ {
		xf := lo + (hi-lo)*float64(i)/float64(canalCurvatureWeightSamples)
		w := s.curvatureWeight(xf)
		cum[i] = cum[i-1] + 0.5*(w+prev)
		prev = w
	}
	return cum
}

// curvatureWeight is κ(x_f)^¼, the node-density weight (canalCurvatureWeightExp equalizes the O(h⁴) error).
func (s coneCanalSpine) curvatureWeight(xf float64) float64 {
	return stdmath.Pow(float64(s.curvatureVector(xf).Length()), canalCurvatureWeightExp)
}

// invertCumulative returns the fractional sample index t∈[0,S] where the strictly-increasing running
// integral `cum` reaches `target`, linearly interpolating within the bracketing pair.
func invertCumulative(cum []float64, target float64) float64 {
	for i := 1; i < len(cum); i++ {
		if cum[i] >= target {
			return float64(i-1) + (target-cum[i-1])/(cum[i]-cum[i-1])
		}
	}
	return float64(len(cum) - 1)
}

// maxEnvelopeError is the between-station envelope error the derivation must bound: the max over interval
// MIDPOINTS (in v) and a u-sweep of |dist(surface point, exact spine) − r|. On the true canal every surface
// point is at distance r from its characteristic ball centre; the cubic v-interp between exact station
// columns deviates, and this measures that deviation with the exact spine machinery (CN2 review). At-station
// columns are exact by construction, so the error peaks mid-interval.
func (s coneCanalSpine) maxEnvelopeError(surf geom.BSplineSurface, st canalStations) float64 {
	vp := spineChordParams(st.centers)
	worst := 0.0
	for j := 0; j+1 < len(st.centers); j++ {
		vmid := 0.5 * (vp[j] + vp[j+1])
		worst = stdmath.Max(worst, s.intervalEnvelopeError(surf, vmid, st.xfs[j], st.xfs[j+1]))
	}
	return worst
}

// intervalEnvelopeError samples the band across u at the mid-v of one station interval and returns the
// largest |dist(point, exact spine) − r|. A mid-interval point's characteristic x_f lies between the two
// bracketing stations, so the closest-spine search is confined to [x_f low, x_f high].
func (s coneCanalSpine) intervalEnvelopeError(surf geom.BSplineSurface, vmid, xfA, xfB float64) float64 {
	lo, hi := sortedSpan(xfA, xfB)
	worst := 0.0
	for _, u := range canalEnvelopeUSamples() {
		p := surf.PointAt(u, vmid)
		worst = stdmath.Max(worst, stdmath.Abs(s.distanceToSpine(p, lo, hi)-s.radius))
	}
	return worst
}

// canalEnvelopeUSamples are the across-band u parameters the envelope error is probed at (the two feet, the
// mid arc, and the quarter points — enough to catch the worst deviation of the interpolated cross-section).
func canalEnvelopeUSamples() []float64 { return []float64{0, 0.25, 0.5, 0.75, 1} }

// distanceToSpine is the distance from p to the exact hyperbola spine — min over x_f∈[lo, hi] of
// |p − center(x_f)| — by golden-section search (the distance is unimodal on a one-interval bracket around
// the characteristic station). Reuses the exact center machinery; no marched geometry.
func (s coneCanalSpine) distanceToSpine(p math.Point3, lo, hi float64) float64 {
	return goldenSectionMin(func(xf float64) float64 { return float64(p.DistanceTo(s.center(xf))) }, lo, hi, canalSpineSearchIters)
}

// goldenSectionMin returns the minimum of a unimodal f over [lo, hi] after `iters` golden-section
// contractions (each shrinks the bracket by the golden ratio ≈0.618; 64 reaches machine precision).
func goldenSectionMin(f func(float64) float64, lo, hi float64, iters int) float64 {
	const invPhi = 0.6180339887498949 // 1/φ
	a, b := lo, hi
	c, d := b-invPhi*(b-a), a+invPhi*(b-a)
	fc, fd := f(c), f(d)
	for k := 0; k < iters; k++ {
		if fc < fd {
			b, d, fd = d, c, fc
			c = b - invPhi*(b-a)
			fc = f(c)
			continue
		}
		a, c, fc = c, d, fd
		d = a + invPhi*(b-a)
		fd = f(d)
	}
	return stdmath.Min(fc, fd)
}

// spineChordParams is the loft's v-parametrization: the normalized cumulative chord length of the station
// centres (P&T §9.2.1, matching geom.alphaParams(·, 1)), so v_j is where station j's exact column lives on
// the built surface. Shared by the envelope-error measure and the at-station exactness test.
func spineChordParams(centers []math.Point3) []float64 {
	cum := make([]float64, len(centers))
	for k := 1; k < len(centers); k++ {
		cum[k] = cum[k-1] + float64(centers[k-1].DistanceTo(centers[k]))
	}
	out := make([]float64, len(centers))
	for k := range out {
		out[k] = cum[k] / cum[len(centers)-1]
	}
	out[len(out)-1] = 1
	return out
}

// bandArcMinRegularity is the minimum tube-regularity factor 1 − r·(K·d̂) over the BAND ARC (the
// characteristic sub-arc the fillet actually spans, from the plane foot d̂ to the cone foot d̂), NOT the
// full tube: the full tube of a canal over the D1 vertex (κ·r = cotα > 1) folds on its concave down-axis
// side, but the band never reaches there (a full-tube check would spuriously reject D1). > 0 everywhere
// ⇒ the band arc is a regular strip of the envelope.
func (s coneCanalSpine) bandArcMinRegularity(xf float64, m, fP, coneT math.Point3) float64 {
	k := s.curvatureVector(xf)
	dA := unitBetween(m, fP)
	dB := unitBetween(m, coneT)
	minF := stdmath.Inf(1)
	for i := 0; i <= canalFoldArcSamples; i++ {
		d := slerpUnit(dA, dB, float64(i)/float64(canalFoldArcSamples))
		minF = stdmath.Min(minF, 1-s.radius*float64(k.Dot(d)))
	}
	return minF
}

// unitBetween is the unit direction from a to b (assumes a != b — the feet sit at radius r > 0 from m).
func unitBetween(a, b math.Point3) math.Vector3 {
	d, _ := math.UnitVector3FromVector(a.VectorTo(b))
	return d.AsVector()
}

// slerpUnit spherically interpolates between unit directions a and b (great-arc, constant angular speed),
// so the sampled d̂ sweeps the actual characteristic arc from the plane foot to the cone foot. Coincident
// directions (angle below canalSlerpMinAngle) return a unchanged.
func slerpUnit(a, b math.Vector3, t float64) math.Vector3 {
	cosw := stdmath.Max(-1, stdmath.Min(1, float64(a.Dot(b))))
	w := stdmath.Acos(cosw)
	if w < canalSlerpMinAngle {
		return a
	}
	sw := stdmath.Sin(w)
	return a.Scale(stdmath.Sin((1-t)*w) / sw).Add(b.Scale(stdmath.Sin(t*w) / sw))
}

// coneCanalArmFillet builds the exact canal arm on a convex-external Cone∧Plane RULING edge, carried in
// the same edgeFillet the straight-edge/torus paths emit plus the hyperbola spine descriptor (armCanalSpine)
// the corner weld reads. It honest-rejects a concave conical bore (material outside the cone; the ruling
// of a bore is convex yet needs A′ = A − r/sinα·â — a follow-on slice) or any build decline (α bands, no
// fit, fold), so coneArmEdge reports the cause via coneArmError (do-no-harm).
func coneCanalArmFillet(e *topo.Edge, co geom.Cone, pl geom.Plane, coneFace, planeFace *topo.Face, r float64, res Resolution) (edgeFillet, coneArmReject) {
	if sgn, ok := coneHostMaterialSign(e, co, coneFace); !ok || sgn <= 0 {
		return edgeFillet{}, coneArmConcaveBore
	}
	nOut, err := math.UnitVector3FromVector(outwardPlaneNormal(planeFace, pl))
	if err != nil {
		return edgeFillet{}, coneArmDegenerate
	}
	surf, spine, reason := buildConeCanalArm(e, co, nOut, r, res)
	if reason != coneArmBuilt {
		return edgeFillet{}, reason
	}
	faces := e.Faces()
	sp := spine
	return edgeFillet{a: faces[0], b: faces[1], edge: e, armSurface: surf, armCanalSpine: &sp}, coneArmBuilt
}

// coneCanalArmError names the ruling-edge (canal) build rejects — each carrying the offending radius and
// cone half-angle — mirroring coneArmSurfaceError for the torus arm.
func coneCanalArmError(reason coneArmReject, co geom.Cone, r float64) error {
	switch reason {
	case coneArmRulingNoFit:
		return fmt.Errorf("fillet: cannot round this Cone∧Plane RULING edge with radius %g — the rolling ball never fits "+
			"the picked span (cone half-angle %g): no hyperbola-spine station exists (tanα·(z_{A′}−z) < r everywhere)", r, co.HalfAngle)
	case coneArmRulingSpan:
		return fmt.Errorf("fillet: cannot round this Cone∧Plane RULING edge with radius %g — the fittable canal-spine span "+
			"collapses to a point (cone half-angle %g)", r, co.HalfAngle)
	case coneArmRulingUnresolved:
		return fmt.Errorf("fillet: cannot round this Cone∧Plane RULING edge with radius %g — the between-station envelope "+
			"error stays over the model-relative bound at the %d-station cap (cone half-angle %g): the hyperbola spine is "+
			"unresolvably high-curvature for this radius", r, canalArmStationsMax, co.HalfAngle)
	default: // coneArmRulingFold
		return fmt.Errorf("fillet: cannot round this Cone∧Plane RULING edge with radius %g — the constant-radius canal band "+
			"self-intersects at a station (irregular band arc, 1−κ·r·cosψ ≤ 0; cone half-angle %g)", r, co.HalfAngle)
	}
}
