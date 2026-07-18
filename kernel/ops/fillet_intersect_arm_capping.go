// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// FR2 — the intersectArmCapping PORT (far-runout-engine-architecture.md ADR-3; far-runout-port-math.md).
//
// The runout trim = armSurface ∩ capping restricted to the material sub-arc between the two feet, EXACT
// on both surfaces and ANALYTIC-ON-THE-ARM (a point is always arm.PointAt(chart) — shared-edge identity,
// tessellator honesty; never a bare polyline). Two pairings ship for the sphere slice and are pinned to
// DRAWEXE (far-runout-port-math §1/§3):
//
//   - Torus ∩ Plane — the spiric of Perseus via geom.SpiricArc/TorusSectionCoeffs (D5/D9/E4's meridian
//     arms on the cap plane; residual 6.83e-7 = OCCT's own approximation floor);
//   - Cylinder ∩ Plane — the exact ellipse via geom.IntersectSurfacesAnalytic (oblique caps).
//
// Torus∩Sphere (closed-form u(v), §2) and Torus/Cyl ∩ Cone/Cyl (feet-bracketed Newton, §4) are derived
// and DRAWEXE-validated but need an analytic-on-arm section-curve geom sibling this slice does not add
// (the census: D5/D9/E4 have no such capping); they clean-decline (do-no-harm) until a cone/torus family
// exercises them. The feet themselves are supplied by the engine (FR3, ADR-4) — closed-form via
// armSprings + springCapFoot (fillet_arm_springs.go); the port consumes them and never re-fits.

// intersectArmCapping dispatches on the arm surface type; each pairing restricts arm ∩ capping to the
// oriented sub-arc feet[0]→feet[1]. r is the fillet radius (carried for the §4 Newton pairings, which are
// not exercised by this slice). Returns (nil, false) — the do-no-harm floor — for any un-shipped pairing
// or on any existence/branch/grazing decline (far-runout-port-math §6, "numerical pitfalls").
func intersectArmCapping(arm, capping geom.Surface, feet [2]math.Point3, r float64, res Resolution) (geom.Curve3, bool) {
	_ = r
	switch a := arm.(type) {
	case geom.Torus:
		return torusCappingTrim(a, capping, feet) // the torus path builds its own scale-local tolerance
	case geom.Cylinder:
		return cylinderCappingTrim(a, capping, feet, res)
	}
	return nil, false // only torus/cylinder arms carry a rolling-ball section
}

// torusCappingTrim routes a torus arm's capping. Only ∩Plane (the spiric) ships for the sphere slice;
// ∩Sphere/∩Cone/∩Cylinder are the un-exercised §2/§4 follow-ons and clean-decline.
func torusCappingTrim(t geom.Torus, capping geom.Surface, feet [2]math.Point3) (geom.Curve3, bool) {
	if pl, ok := capping.(geom.Plane); ok {
		return torusPlaneTrim(t, pl, feet)
	}
	return nil, false
}

// torusPlaneTrim is the D5 workhorse: the plane cuts the torus in a spiric of Perseus (geom.SpiricArc,
// both attitudes) and the trim is the branch through BOTH feet, restricted to their tube-angle interval.
// The feet are inverted onto the arm chart (far-runout-port-math §0) and certified on it; the branch is
// selected by the endpoint certificate (§6.1 — the same-sign match rejects the mirror oval); existence
// and grazing between the feet are guarded (§6, pitfalls). Analytic-on-the-arm by SpiricArc construction.
func torusPlaneTrim(t geom.Torus, pl geom.Plane, feet [2]math.Point3) (geom.Curve3, bool) {
	tol := armCapTol(feet[0], feet[1], t.Center, pl.Origin)
	v0, ok0 := torusChartV(t, feet[0], tol)
	v1, ok1 := torusChartV(t, feet[1], tol)
	if !ok0 || !ok1 {
		return nil, false // a foot not on the arm is an upstream bug (§0) — decline with do-no-harm
	}
	phi, m, k, c := geom.TorusSectionCoeffs(t, pl)
	if m*(t.MajorRadius+t.MinorRadius) <= tol {
		return nil, false // plane ⊥ torus axis (M→0): the latitude-circle emitter is a §1 follow-on
	}
	if !spiricBandOK(t, m, k, c, v0, v1, tol) {
		return nil, false
	}
	branch, ok := selectSpiricBranch(t, phi, m, k, c, v0, v1, feet, tol)
	if !ok {
		return nil, false // 0 or ≥2 branches through both feet: geometric ambiguity, decline (§6.4)
	}
	return geom.SpiricArc{Torus: t, Phi: phi, M: m, K: k, C: c, Branch: branch, V0: v0, V1: v1}, true
}

// torusChartV inverts a point onto the torus chart (far-runout-port-math §0) and returns its tube angle v,
// certifying |P(chart) − p| ≤ tol so a foot off the arm declines rather than snapping.
func torusChartV(t geom.Torus, p math.Point3, tol float64) (float64, bool) {
	axis := t.AxisDir.AsVector()
	binormal := axis.Cross(t.Ref.AsVector())
	q := t.Center.VectorTo(p)
	za := float64(q.Dot(axis))
	perp := q.Sub(axis.Scale(math.Scalar(za)))
	u := stdmath.Atan2(float64(perp.Dot(binormal)), float64(perp.Dot(t.Ref.AsVector())))
	v := stdmath.Atan2(za, float64(perp.Length())-t.MajorRadius)
	if float64(t.PointAt(u, v).DistanceTo(p)) > tol {
		return 0, false
	}
	return v, true
}

// selectSpiricBranch enumerates the ± arccos branches and keeps the one whose section point at BOTH feet's
// tube angles hits BOTH feet within tol (far-runout-port-math §6.1 — the same-sign endpoint certificate,
// 11 orders of discrimination on D5). Exactly one survivor is required; 0 or 2 ⇒ ambiguity ⇒ decline.
func selectSpiricBranch(t geom.Torus, phi, m, k, c, v0, v1 float64, feet [2]math.Point3, tol float64) (float64, bool) {
	branch, count := 0.0, 0
	for _, br := range [2]float64{1, -1} {
		d0 := float64(spiricPoint(t, phi, m, k, c, br, v0).DistanceTo(feet[0]))
		d1 := float64(spiricPoint(t, phi, m, k, c, br, v1).DistanceTo(feet[1]))
		if d0 <= tol && d1 <= tol {
			branch, count = br, count+1
		}
	}
	if count != 1 {
		return 0, false
	}
	return branch, true
}

// spiricPoint evaluates the spiric section at tube angle v on the given branch, ON the torus — it reuses
// geom.SpiricArc.UAt(v) (the single-valued u(v) = Φ ± arccos w) so the port shares the arc's exact math.
func spiricPoint(t geom.Torus, phi, m, k, c, branch, v float64) math.Point3 {
	sa := geom.SpiricArc{Torus: t, Phi: phi, M: m, K: k, C: c, Branch: branch}
	return t.PointAt(sa.UAt(v), v)
}

// spiricBandOK is the existence + grazing guard (far-runout-port-math §6 pitfalls): the section must reach
// the arm across the WHOLE tube interval (|w(v)| ≤ 1 at both feet and every interior extremum) and neither
// foot may sit at a section extreme (√(1−w²)·M·ρ > tol — the spring not tangent to the capping there).
func spiricBandOK(t geom.Torus, m, k, c, v0, v1, tol float64) bool {
	majR, r := t.MajorRadius, t.MinorRadius
	for _, v := range bandCheckAngles(k, c, majR, r, v0, v1) {
		w, rho := spiricW(m, k, c, majR, r, v), majR+r*stdmath.Cos(v)
		if stdmath.Abs(w) > 1+tol/(m*rho) {
			return false // capping does not cut the arm between the feet: no trim
		}
	}
	for _, v := range [2]float64{v0, v1} {
		w, rho := spiricW(m, k, c, majR, r, v), majR+r*stdmath.Cos(v)
		if stdmath.Sqrt(stdmath.Max(1-w*w, 0))*m*rho <= tol {
			return false // degenerate foot: spring ∥ capping (grazing runout)
		}
	}
	return true
}

// spiricW is w(v) = (K − C·r·sin v) / (M·(R + r·cos v)), the arccos argument of the spiric branch.
func spiricW(m, k, c, majR, r, v float64) float64 {
	cv, sv := stdmath.Cos(v), stdmath.Sin(v)
	return (k - c*r*sv) / (m * (majR + r*cv))
}

// bandCheckAngles returns the tube angles to test for existence: the two feet plus every interior extremum
// of w in the open interval (far-runout-port-math §6 — w′=0 ⇔ k·sin v − c·R·cos v − c·r = 0, closed form).
func bandCheckAngles(k, c, majR, r, v0, v1 float64) []float64 {
	lo, hi := stdmath.Min(v0, v1), stdmath.Max(v0, v1)
	angles := []float64{v0, v1}
	for _, e := range spiricExtrema(k, c, majR, r) {
		for _, cand := range [3]float64{e, e - 2*stdmath.Pi, e + 2*stdmath.Pi} {
			if cand > lo && cand < hi {
				angles = append(angles, cand)
			}
		}
	}
	return angles
}

// spiricExtrema solves k·sin v − c·R·cos v = c·r (the w′=0 condition) in closed form, returning its ≤2
// roots (empty when the amplitude is degenerate or the equation has no real solution).
func spiricExtrema(k, c, majR, r float64) []float64 {
	amp := stdmath.Hypot(k, c*majR)
	if amp == 0 {
		return nil
	}
	s := c * r / amp
	if stdmath.Abs(s) > 1 {
		return nil
	}
	delta, base := stdmath.Atan2(-c*majR, k), stdmath.Asin(s)
	return []float64{base - delta, stdmath.Pi - base - delta}
}

// armCapTol is the port's scale-invariant tolerance (ADR-0042): the model-relative weld built from the
// feet, the arm centre, and the capping origin, so a µm or km copy of the runout classifies identically.
func armCapTol(a, b, c, d math.Point3) float64 {
	return geom.ResolutionForPoints([]math.Point3{a, b, c, d}).Weld()
}

// cylinderCappingTrim routes a cylinder arm's capping. Only ∩Plane (the oblique ellipse) ships;
// ∩Sphere/∩Cone/∩Cylinder are the un-exercised §2/§4 follow-ons and clean-decline.
func cylinderCappingTrim(cyl geom.Cylinder, capping geom.Surface, feet [2]math.Point3, res Resolution) (geom.Curve3, bool) {
	if pl, ok := capping.(geom.Plane); ok {
		return cylinderPlaneTrim(cyl, pl, feet, res)
	}
	return nil, false
}

// cylinderPlaneTrim is the oblique cylinder cap: the exact ellipse (geom.IntersectSurfacesAnalytic, minor
// radius R_a, major R_a/|m̂·â|, major axis = the cylinder axis projected into the plane — matched to OCCT
// in far-runout-port-math §3) restricted to the sub-arc between the feet. A perpendicular cap yields a
// circle, which is the perpendicular fast-path's domain (farCrossSectionArc), so the port declines it.
func cylinderPlaneTrim(cyl geom.Cylinder, pl geom.Plane, feet [2]math.Point3, res Resolution) (geom.Curve3, bool) {
	curves, ok := geom.IntersectSurfacesAnalytic(pl, cyl, res)
	if !ok || len(curves) != 1 {
		return nil, false // an axis-parallel cap yields a line pair, not a single section conic
	}
	if ell, isEll := curves[0].(geom.EllipseFull); isEll {
		return ellipseTrim(ell, feet, armCapTol(feet[0], feet[1], cyl.Origin, pl.Origin))
	}
	return nil, false // Circle (perpendicular cap): the perpendicular fast-path's job, not the port's
}

// ellipseTrim restricts a full section ellipse to the minor sub-arc between the feet (reusing the setback
// ellipse helpers ellipseSubArc/ellipseAngleOf), certifying each foot lies on the ellipse first. The full
// material-side host certificate (§6.3) lands in FR3 where the engine supplies the arm's hosts; the
// oriented feet + minor arc are the sub-arc — a shape the sphere slice never exercises (cyl caps are perp).
func ellipseTrim(e geom.EllipseFull, feet [2]math.Point3, tol float64) (geom.Curve3, bool) {
	if !footOnEllipse(e, feet[0], tol) || !footOnEllipse(e, feet[1], tol) {
		return nil, false // a foot off the section declines rather than projecting
	}
	return ellipseSubArc(e, feet[0], feet[1], false)
}

// footOnEllipse certifies p lies on the ellipse to tol (invert to its angle, re-evaluate, compare).
func footOnEllipse(e geom.EllipseFull, p math.Point3, tol float64) bool {
	return float64(ellipsePointAtAngle(e, ellipseAngleOf(e, p)).DistanceTo(p)) <= tol
}
