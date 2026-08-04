// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// M5 Slice A, Task 5.2 (m5-weld-setback-retrim-derivation.md §A.2): the great-circle weld rails
// where each curved arm meets the corner sphere, plus the exact-G1 assertion. At the setback
// station the arm's generating ball-centre equals C, so the arm's terminal radius-r cross-section
// is a circle centred at C in the plane ⊥ spine-tangent — a GREAT CIRCLE of the corner sphere. That
// circle IS the shared weld curve (G0 exact), and along it both the arm normal and the sphere
// normal are (P−C)/r (G1 exact). This file emits that rail and certifies the G1 identity; it wires
// into nothing yet — T5.4 consumes it to bound the arm and sphere faces.

// railGreatCircleTol is the fractional tolerance (a multiple of the corner radius r) for the
// honest-reject that a built setback rail really is the corner sphere's great circle: its
// supporting-circle centre must lie within railGreatCircleTol·r of C and its radius within
// railGreatCircleTol·r of r. The rail is a great circle BY CONSTRUCTION (its three defining points
// are coplanar with C and all at distance r), so the residual is only the circumcircle solve's
// floating-point noise (~1e-12·r here). A fraction of r is scale-free, so ADR-0042's model-relative
// rule is honoured without threading a Resolution into this pure constructor; 1e-6 rejects a
// genuinely mis-built tangent point (off by ≫ machine-eps) yet never a valid corner.
const railGreatCircleTol = 1e-6

// g1RailSamples is how many points the G1 assertion checks along a rail. Five (both endpoints plus
// three interior) is ample: the normal error ‖C−m‖/r is constant along a valid rail (m≡C) and grows
// monotonically off it, so a coarse sample cannot miss a divergence a finer one would catch.
const g1RailSamples = 5

// curvedSetbackRail is the great-circle arc on the corner sphere between an arm's two host-tangent
// points — the shared curve welding that arm's setback end to the sphere (exact G0), along which
// both surface normals are (P−C)/r (exact G1). It builds the arc through the two tangent points and
// their great-circle bisector, then honest-rejects if the result is not the sphere's great circle
// (centre C, radius r) — a guard that catches a mis-built tangent point. Example:
//
//	rail, ok := curvedSetbackRail(w, w.arms[0])
//	if !ok { /* decline the weld — do-no-harm */ }
func curvedSetbackRail(w cornerWeld, arm armSetback) (geom.Arc3d, bool) {
	tA := endpointOf(w.center, w.radius, arm.railDir0)
	tB := endpointOf(w.center, w.radius, arm.railDir1)
	mid, ok := greatCircleBisector(w.center, w.radius, arm.railDir0, arm.railDir1)
	if !ok {
		return geom.Arc3d{}, false // antipodal rail directions — no unique great-circle midpoint
	}
	rail, err := geom.Arc3dByThreePoints(tA, mid, tB)
	if err != nil {
		return geom.Arc3d{}, false // collinear endpoints — degenerate rail (subtense 0 or π)
	}
	if !railIsGreatCircle(rail, w.center, w.radius) {
		return geom.Arc3d{}, false // supporting circle not centred at C / radius ≠ r — mis-built tangent point
	}
	return rail, true
}

// greatCircleBisector is the midpoint of the great-circle arc between the two rail directions:
// C + r·unit(d0 + d1). Because unit(d0+d1) is a combination of d0 and d1, this point lies in the
// plane through C spanned by them, so {T_A, midpoint, T_B} are coplanar with C — the circle through
// them is the sphere's great circle. Errors when d0 and d1 are antipodal (their sum is zero).
func greatCircleBisector(c math.Point3, r float64, d0, d1 math.UnitVector3) (math.Point3, bool) {
	bisector, err := math.UnitVector3FromVector(d0.AsVector().Add(d1.AsVector()))
	if err != nil {
		return math.Point3{}, false
	}
	return endpointOf(c, r, bisector), true
}

// railIsGreatCircle is the honest-reject guard: the built rail's supporting circle must be centred
// at C with radius r (within railGreatCircleTol·r), i.e. it is a great circle of the corner sphere.
func railIsGreatCircle(rail geom.Arc3d, center math.Point3, r float64) bool {
	tol := railGreatCircleTol * r
	if rail.Center.DistanceTo(center) > tol {
		return false
	}
	return stdmath.Abs(rail.Radius-r) <= tol
}

// curvedRailG1 samples the rail and asserts the weld is G1: at each point the arm normal (the canal
// identity (P−m)/r, m = the arm's moving ball-centre) and the sphere normal (P−C)/r coincide within
// res.Weld(). Their difference is exactly (C−m)/r — the solved-station error — so this bites only
// when the arm's ball-centre drifts off C (e.g. a mis-solved or perturbed arm). Example:
//
//	if !curvedRailG1(arm.arm, rail, w.center, w.radius, res) { /* not G1 — reject the weld */ }
func curvedRailG1(arm geom.Surface, rail geom.Arc3d, center math.Point3, r float64, res Resolution) bool {
	tol := res.Weld() // unit-normal difference is dimensionless — the relative resolution, not r-scaled
	for i := 0; i < g1RailSamples; i++ {
		p := rail.PointAt(float64(i) / float64(g1RailSamples-1))
		m, ok := armBallCenter(arm, p)
		if !ok {
			return false // arm ball-centre undefined at this sample (P on the spine — degenerate)
		}
		armN := m.VectorTo(p).Scale(1 / r)         // canal identity: n_arm = (P − m)/r
		sphereN := center.VectorTo(p).Scale(1 / r) // sphere normal:  n_sphere = (P − C)/r
		if armN.Sub(sphereN).Length() > tol {
			return false
		}
	}
	return true
}

// armBallCenter is the arm surface's moving ball-centre m generating point P — the nearest point on
// the arm's spine (cylinder: the axis foot; torus: the major-circle point). The arm's outward normal
// at P is (P−m)/tubeRadius; at the setback station m equals the corner centre C for every rail point.
func armBallCenter(surf geom.Surface, p math.Point3) (math.Point3, bool) {
	switch s := surf.(type) {
	case geom.Cylinder:
		return cylinderBallCenter(s, p), true
	case geom.Torus:
		return torusBallCenter(s, p)
	default:
		return math.Point3{}, false // only torus / cylinder arms carry a rolling-ball spine
	}
}

// cylinderBallCenter is the foot of the perpendicular from P onto the cylinder arm's axis line.
func cylinderBallCenter(c geom.Cylinder, p math.Point3) math.Point3 {
	axis := c.AxisDir.AsVector()
	return c.Origin.TranslateBy(axis.Scale(c.Origin.VectorTo(p).Dot(axis)))
}

// torusBallCenter is the nearest point on the torus arm's major (spine) circle to P: C_t + ρ·radial,
// radial = the unit in-plane direction from the torus centre toward P. Errors when P projects onto
// the torus axis (the radial direction is undefined there).
func torusBallCenter(t geom.Torus, p math.Point3) (math.Point3, bool) {
	axis := t.AxisDir.AsVector()
	d := t.Center.VectorTo(p)
	inPlane := d.Sub(axis.Scale(d.Dot(axis)))
	radial, err := math.UnitVector3FromVector(inPlane)
	if err != nil {
		return math.Point3{}, false
	}
	return t.Center.TranslateBy(radial.AsVector().Scale(t.MajorRadius)), true
}
