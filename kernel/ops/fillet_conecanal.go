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
	// canalArmStations is the number of exact hyperbola stations sampled across the arm span (giving
	// canalArmStations+1 cross-section columns). Each column is EXACT on the true envelope; the count
	// controls only the cubic v-interpolation BETWEEN stations, so a smooth hyperbola arc needs few.
	canalArmStations = 24
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

// buildConeCanalArm builds the exact canal BSplineSurface of a convex-external Cone∧Plane ruling edge and
// the spine descriptor the weld needs, or a cause-named reject. nOut is the plane's material-outward unit
// normal (fixes the offset direction). It samples exact hyperbola stations across the picked edge's fit
// span, builds each cross-section from the exact plane/cone feet, guards the band arc against a canal
// fold, and lofts homogeneously via the shared geom canal stack — NO marching, NO approximation.
func buildConeCanalArm(e *topo.Edge, co geom.Cone, nOut math.UnitVector3, r float64, res Resolution) (geom.BSplineSurface, coneCanalSpine, coneArmReject) {
	spine, reason := newConeCanalSpine(co, nOut, r, res)
	if reason != coneArmBuilt {
		return geom.BSplineSurface{}, coneCanalSpine{}, reason
	}
	lo, hi, reason := spine.edgeXfSpan(e, res)
	if reason != coneArmBuilt {
		return geom.BSplineSurface{}, coneCanalSpine{}, reason
	}
	centers, feetA, feetB, reason := spine.sampleStations(lo, hi)
	if reason != coneArmBuilt {
		return geom.BSplineSurface{}, coneCanalSpine{}, reason
	}
	surf, err := geom.LoftCanalStations(centers, feetA, feetB, r, res.Weld())
	if err != nil {
		return geom.BSplineSurface{}, coneCanalSpine{}, coneArmDegenerate
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

// sampleStations samples canalArmStations+1 exact hyperbola stations uniformly in x_f across [lo, hi],
// returning the ball-centres and the exact plane/cone feet, after guarding each station's BAND ARC
// (plane foot → cone foot) against a canal self-intersection (fold). A cone foot on the axis or a folded
// band arc rejects (do-no-harm: never loft a malformed band).
func (s coneCanalSpine) sampleStations(lo, hi float64) (centers, feetA, feetB []math.Point3, reason coneArmReject) {
	n := canalArmStations
	centers = make([]math.Point3, n+1)
	feetA = make([]math.Point3, n+1)
	feetB = make([]math.Point3, n+1)
	for i := 0; i <= n; i++ {
		xf := lo + (hi-lo)*float64(i)/float64(n)
		m := s.center(xf)
		coneT, ok := s.coneFoot(m)
		if !ok {
			return nil, nil, nil, coneArmDegenerate
		}
		fP := s.planeFoot(m)
		if s.bandArcMinRegularity(xf, m, fP, coneT) <= 0 {
			return nil, nil, nil, coneArmRulingFold
		}
		centers[i], feetA[i], feetB[i] = m, fP, coneT
	}
	return centers, feetA, feetB, coneArmBuilt
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
	default: // coneArmRulingFold
		return fmt.Errorf("fillet: cannot round this Cone∧Plane RULING edge with radius %g — the constant-radius canal band "+
			"self-intersects at a station (irregular band arc, 1−κ·r·cosψ ≤ 0; cone half-angle %g)", r, co.HalfAngle)
	}
}
