// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// FR2 — closed-form spring curves and runout feet (far-runout-port-math.md §5).
//
// Each fillet arm touches its two host faces along a SPRING (contact) curve; the runout feet are the two
// crossings spring ∩ capping. Per ADR-4 the engine owns the feet: in FR3 it builds them here and hands
// them to intersectArmCapping. FR2 lands the closed forms and pins them to DRAWEXE (D5: the plane-spring
// foot matches OCCT's edge endpoint to 3.9e-5 — its own blend tolerance — the sphere-spring foot to 8.2e-8):
//
//   - Torus arm (sphere host + plane host) → two LATITUDE circles: the tube touches the host plane at the
//     tube angle where its axial offset reaches ±r (v_P), and the host sphere along the tangent-contact
//     latitude v_S = atan2(B, A) of the |P−O|² = R² identity (both are proper-fillet tangencies).
//   - Cylinder arm (two plane hosts) → two axis RULINGS at θ_i = atan2(−n̂_i·b̂, −n̂_i·r̂).
//
// Feet = spring ∩ capping, closed form and low degree (circle∩plane → atan2±arccos, ruling∩plane → linear),
// the nearer root to the far vertex kept (the nearerRoot precedent).

// armSprings returns an arm's two host-contact spring curves. hostA/hostB are the arm's two host surfaces
// (edgeFillet.a/.b in FR3). It takes the whole edgeFillet so a CANAL arm's springs can recover the exact
// hyperbola spine (armCanalSpine, keyed BEFORE the type switch — CN4b-1, the CN4a armStation(ef)
// precedent); every non-canal arm keeps the torus/cylinder switch byte-identically. Declines when the
// hosts are not a recognized fillet pairing or are not tangent to the arm — a wrong pairing must never
// fabricate a spring. r is the fillet radius (torus minor).
func armSprings(ef edgeFillet, hostA, hostB geom.Surface, r float64) ([2]geom.Curve3, bool) {
	if ef.armCanalSpine != nil {
		return canalArmSprings(*ef.armCanalSpine, ef.edge, r)
	}
	switch a := ef.armSurface.(type) {
	case geom.Torus:
		return torusArmSprings(a, hostA, hostB)
	case geom.Cylinder:
		return cylinderArmSprings(a, hostA, hostB)
	}
	_ = r
	return [2]geom.Curve3{}, false
}

// torusArmSprings builds a torus arm's two host-contact springs, ordered [non-plane, plane] so
// springsForHosts can map them by which host is the plane. Two host pairings ship: Sphere∧Plane (the FR2
// sphere slice) and Cylinder∧Plane (the P5 far-runout chain, Link 1). The non-plane spring is host-specific
// (sphere latitude / cylinder equator); the plane spring is the shared v_P latitude circle.
func torusArmSprings(t geom.Torus, hostA, hostB geom.Surface) ([2]geom.Curve3, bool) {
	if sp, pl, ok := spherePlaneHosts(hostA, hostB); ok {
		return torusSpherePlaneSprings(t, sp, pl)
	}
	if cyl, pl, ok := cylinderPlaneHosts(hostA, hostB); ok {
		return torusCylinderPlaneSprings(t, cyl, pl)
	}
	return [2]geom.Curve3{}, false
}

// torusSpherePlaneSprings builds the [sphere, plane] latitude circles (the unchanged FR2 sphere-slice path):
// the DRAWEXE far edge runs sphere foot → plane foot.
func torusSpherePlaneSprings(t geom.Torus, sp geom.Sphere, pl geom.Plane) ([2]geom.Curve3, bool) {
	tol := geom.ResolutionForPoints([]math.Point3{t.Center, sp.Center, pl.Origin}).Weld()
	sphereSpring, ok1 := torusSphereSpring(t, sp, tol)
	planeSpring, ok2 := torusPlaneSpring(t, pl, tol)
	if !ok1 || !ok2 {
		return [2]geom.Curve3{}, false
	}
	return [2]geom.Curve3{sphereSpring, planeSpring}, true
}

// torusCylinderPlaneSprings builds the [cylinder, plane] latitude circles for a torus arm on a Cylinder∧
// Plane pairing (Link 1, P5): the coaxial cylinder-host equator circle and the existing plane-host latitude
// circle. On P5 OCCT emits these as Circle R=50=R_h (cylinder host) and Circle R=45=R′ (plane host).
func torusCylinderPlaneSprings(t geom.Torus, cyl geom.Cylinder, pl geom.Plane) ([2]geom.Curve3, bool) {
	tol := geom.ResolutionForPoints([]math.Point3{t.Center, cyl.Origin, pl.Origin}).Weld()
	cylSpring, ok1 := torusCylinderSpring(t, cyl, tol)
	planeSpring, ok2 := torusPlaneSpring(t, pl, tol)
	if !ok1 || !ok2 {
		return [2]geom.Curve3{}, false
	}
	return [2]geom.Curve3{cylSpring, planeSpring}, true
}

// cylinderPlaneHosts identifies which host is the cylinder and which the plane (order-independent), the
// sibling of spherePlaneHosts for the torus-arm Cylinder∧Plane pairing.
func cylinderPlaneHosts(a, b geom.Surface) (geom.Cylinder, geom.Plane, bool) {
	if cyl, ok := a.(geom.Cylinder); ok {
		if pl, ok2 := b.(geom.Plane); ok2 {
			return cyl, pl, true
		}
	}
	if cyl, ok := b.(geom.Cylinder); ok {
		if pl, ok2 := a.(geom.Plane); ok2 {
			return cyl, pl, true
		}
	}
	return geom.Cylinder{}, geom.Plane{}, false
}

// torusCylinderSpring is the latitude circle where a coaxial torus arm touches its host cylinder (radius
// R_h). A constant-radius rolling-ball torus arm has a CIRCULAR ball-spine, and a plane∩cylinder spine is a
// circle only when coaxial — so the arm is coaxial with its host cylinder and tangency fixes cos v_C =
// (R_h − R′)/r = ±1 EXACTLY: the spring is the tube-equator latitude circle (centre = torus centre since
// sin v_C = 0, radius = R′±r = R_h, axis = torus axis; v_C=0 concave/bore R′=R_h−r, v_C=π convex/shaft
// R′=R_h+r). HOST-vs-CAPPING guard (load-bearing): only a genuine tangent host (|cos v_C|=1 within tol) is
// admitted — a cylinder with |cos v_C|<1 CUTS the tube in two latitude circles (a transversal capping, not a
// host) and declines rather than fabricating a mid-tube circle (torus-cyl-springs-feet-derivation Link 1).
func torusCylinderSpring(t geom.Torus, cyl geom.Cylinder, tol float64) (geom.Circle, bool) {
	axis := t.AxisDir.AsVector()
	hostAxis := cyl.AxisDir.AsVector()
	if stdmath.Abs(float64(axis.Dot(hostAxis))) < 1-sinFloor {
		return geom.Circle{}, false // host cylinder axis not ∥ torus axis: not a coaxial host
	}
	d := cyl.Origin.VectorTo(t.Center) // C − O_h
	da := float64(d.Dot(hostAxis))
	aPerp := float64(d.Sub(hostAxis.Scale(math.Scalar(da))).Length())
	if aPerp > tol {
		return geom.Circle{}, false // torus centre off the host axis (A⊥ > tol): no latitude-circle spring
	}
	off := cyl.Radius - t.MajorRadius // R_h − R′; a tangent host needs |R_h−R′| = r (i.e. |cos v_C| = 1)
	if stdmath.Abs(stdmath.Abs(off)-t.MinorRadius) > tol {
		return geom.Circle{}, false // |cos v_C| = |R_h−R′|/r ≠ 1: transversal capping cylinder, not a host
	}
	radius := t.MajorRadius + stdmath.Copysign(t.MinorRadius, off) // v_C=0 (off>0)→R+r, v_C=π (off<0)→R−r
	return geom.Circle{Center: t.Center, Normal: t.AxisDir, RefDir: t.Ref, Radius: radius}, true
}

// spherePlaneHosts identifies which host is the sphere and which the plane (order-independent).
func spherePlaneHosts(a, b geom.Surface) (geom.Sphere, geom.Plane, bool) {
	if sp, ok := a.(geom.Sphere); ok {
		if pl, ok2 := b.(geom.Plane); ok2 {
			return sp, pl, true
		}
	}
	if sp, ok := b.(geom.Sphere); ok {
		if pl, ok2 := a.(geom.Plane); ok2 {
			return sp, pl, true
		}
	}
	return geom.Sphere{}, geom.Plane{}, false
}

// torusPlaneSpring is the latitude circle where the tube is tangent to the host plane: the plane normal is
// ∥ the torus axis and its axial offset from the torus centre is ±r (tube extreme). Declines otherwise.
func torusPlaneSpring(t geom.Torus, pl geom.Plane, tol float64) (geom.Circle, bool) {
	axis := t.AxisDir.AsVector()
	n := pl.Normal()
	if stdmath.Abs(float64(axis.Dot(n))) < 1-sinFloor {
		return geom.Circle{}, false // plane normal not ∥ torus axis: not this fillet's host plane
	}
	a := float64(axis.Dot(t.Center.VectorTo(pl.Origin))) // signed axial distance centre → plane
	if stdmath.Abs(stdmath.Abs(a)-t.MinorRadius) > tol {
		return geom.Circle{}, false // plane not tangent to the tube
	}
	sinV := math.Clamp(a/t.MinorRadius, -1, 1)
	cosV := stdmath.Sqrt(stdmath.Max(1-sinV*sinV, 0))
	center := t.Center.TranslateBy(axis.Scale(math.Scalar(t.MinorRadius * sinV)))
	return geom.Circle{Center: center, Normal: t.AxisDir, RefDir: t.Ref, Radius: t.MajorRadius + t.MinorRadius*cosV}, true
}

// torusSphereSpring is the latitude circle where the tube is tangent to the host sphere: the double root of
// |P(u,v) − O|² = R_c², which collapses to v = atan2(B, A) with A = 2R′r, B = 2(d·â)r, d = C − O. The
// collapse to a single latitude (a full u-circle of contact) is ONLY valid when the sphere is COAXIAL with
// the torus axis — off-axis the |P−O|² has a cos(u−Ψ) term (far-runout-port-math §2) and there is no
// latitude-circle spring, so a transverse offset A⊥ > tol declines rather than fabricating a wrong spring.
func torusSphereSpring(t geom.Torus, sp geom.Sphere, tol float64) (geom.Circle, bool) {
	axis := t.AxisDir.AsVector()
	d := sp.Center.VectorTo(t.Center) // C − O_c
	da := float64(d.Dot(axis))
	aPerp := float64(d.Sub(axis.Scale(math.Scalar(da))).Length()) // A⊥: sphere-centre offset ⊥ the axis
	if aPerp > tol {
		return geom.Circle{}, false // off-axis sphere (A⊥ > tol): no latitude-circle spring, never fabricate
	}
	R, r := t.MajorRadius, t.MinorRadius
	a, b := 2*R*r, 2*da*r
	rhs := sp.Radius*sp.Radius - float64(d.LengthSquared()) - R*R - r*r
	amp := stdmath.Hypot(a, b)
	if amp < tol || stdmath.Abs(stdmath.Abs(rhs)-amp) > tol {
		return geom.Circle{}, false // no DOUBLE root (tangent contact): |rhs|≠amp ⇒ not this fillet's host sphere
	}
	v := stdmath.Atan2(b, a)
	if rhs < 0 {
		v += stdmath.Pi // the rhs=−amp tangency is the antipodal tube angle (atan2 alone is π-wrong here)
	}
	center := t.Center.TranslateBy(axis.Scale(math.Scalar(r * stdmath.Sin(v))))
	return geom.Circle{Center: center, Normal: t.AxisDir, RefDir: t.Ref, Radius: R + r*stdmath.Cos(v)}, true
}

// cylinderArmSprings builds the two axis rulings where the tube is tangent to each host, in the
// (hostA, hostB) order the caller maps back to (ef.a, ef.b). Each host contributes a straight ruling
// ∥ the arm axis: a PLANE host via the closed-form azimuth (cylinderPlaneRuling) and an
// EllipticalCylinder host (F4 vein) via the tangent foot ruling (cylinderEllipticRuling) — both are
// geom.Line, so springCapFoot crosses them with the capping plane through the SAME linePlaneFoot.
func cylinderArmSprings(c geom.Cylinder, hostA, hostB geom.Surface) ([2]geom.Curve3, bool) {
	r0, ok0 := cylinderHostRuling(c, hostA)
	r1, ok1 := cylinderHostRuling(c, hostB)
	if !ok0 || !ok1 {
		return [2]geom.Curve3{}, false
	}
	return [2]geom.Curve3{r0, r1}, true
}

// cylinderHostRuling is the straight contact ruling (∥ the arm axis) where the cylinder arm touches
// one host. A PLANE host keeps the byte-identical closed-form azimuth ruling; an EllipticalCylinder
// host (the elliptic-prism vein) returns the ruling through the arm↔wall tangent foot. Declines for
// any other host so a non-plane/non-elliptic pairing still floors honestly (do-no-harm).
func cylinderHostRuling(c geom.Cylinder, host geom.Surface) (geom.Curve3, bool) {
	switch h := host.(type) {
	case geom.Plane:
		return cylinderPlaneRuling(c, h)
	case geom.EllipticalCylinder:
		return cylinderEllipticRuling(c, h)
	}
	return nil, false
}

// cylinderEllipticRuling is the straight ruling where the cylinder arm is tangent to the elliptic wall:
// the arm and wall are coaxial-invariant translation surfaces (both ∥ the extrusion axis), so their
// tangency is a line ∥ the arm axis through the foot of the arm axis on the wall (the SAME generic
// point-inversion armRunoutFoot uses, so the spring rail and the contact rail land identically).
func cylinderEllipticRuling(c geom.Cylinder, ec geom.EllipticalCylinder) (geom.Line, bool) {
	_, _, foot := geom.ClosestPointOnSurface(ec, c.Origin)
	ln, err := geom.NewLine(foot, c.AxisDir.AsVector())
	return ln, err == nil
}

// cylinderPlaneRuling is the ruling where the cylinder touches host plane pl: azimuth θ = atan2(−n̂·b̂,
// −n̂·r̂) (the tube offset −r·n̂ reaches the host), a line through that contact point along the axis.
func cylinderPlaneRuling(c geom.Cylinder, pl geom.Plane) (geom.Line, bool) {
	n := pl.Normal()
	ref := c.Ref.AsVector()
	bin := c.AxisDir.Cross(c.Ref)
	theta := stdmath.Atan2(-float64(n.Dot(bin)), -float64(n.Dot(ref)))
	cos, sin := stdmath.Cos(theta), stdmath.Sin(theta)
	radial := ref.Scale(math.Scalar(cos)).Add(bin.Scale(math.Scalar(sin)))
	origin := c.Origin.TranslateBy(radial.Scale(math.Scalar(c.Radius)))
	ln, err := geom.NewLine(origin, c.AxisDir.AsVector())
	if err != nil {
		return geom.Line{}, false
	}
	return ln, true
}

// springCapFoot returns the foot where a spring curve crosses the capping face — the nearer root to the far
// vertex `near`. Plane cappings ship the FR2 sphere-slice forms (circle/ruling/canal); Cylinder cappings
// ship the Link-2 circle∩cylinder form (the torus-arm equator spring, P5). Sphere/cone cappings are §5
// follow-ons and decline.
func springCapFoot(spring geom.Curve3, capping geom.Surface, near math.Point3, res Resolution) (math.Point3, bool) {
	switch cap := capping.(type) {
	case geom.Plane:
		return springPlaneFoot(spring, cap, near, res)
	case geom.Cylinder:
		return springCylinderFoot(spring, cap, near, res)
	}
	return math.Point3{}, false
}

// springPlaneFoot crosses a spring with a capping PLANE (the unchanged FR2 sphere-slice path): circle∩plane
// (α·cos u + β·sin u = γ), ruling∩plane (linear), and the cone-canal spring's own crossing.
func springPlaneFoot(spring geom.Curve3, pl geom.Plane, near math.Point3, res Resolution) (math.Point3, bool) {
	switch s := spring.(type) {
	case geom.Circle:
		return circlePlaneFoot(s, pl, near, res)
	case geom.Line:
		return linePlaneFoot(s, pl)
	case coneCanalSpring:
		p, ok, _ := s.canalCapFoot(pl, near, res) // reason surfaced via springCapFootReasoned
		return p, ok
	}
	return math.Point3{}, false
}

// springCylinderFoot crosses a spring with a capping CYLINDER (O₂,â₂,R₂) — Link 2 of the torus∩cylinder
// far-runout chain. A torus arm's host-contact springs are CIRCLES (equator/latitude), so only the circle
// form is reachable; it keeps the nearer root to the far vertex (the nearerRoot precedent).
func springCylinderFoot(spring geom.Curve3, cyl geom.Cylinder, near math.Point3, res Resolution) (math.Point3, bool) {
	if c, ok := spring.(geom.Circle); ok {
		return circleCylinderFoot(c, cyl, near, res)
	}
	return math.Point3{}, false
}

// circlePlaneFoot solves α·cos u + β·sin u = γ for the circle-in-plane intersection and returns the root
// nearer the far vertex (α = a·m̂·r̂, β = a·m̂·b̂, γ = m̂·(o − Q); u = atan2(β,α) ± arccos(γ/√(α²+β²))).
func circlePlaneFoot(c geom.Circle, pl geom.Plane, near math.Point3, res Resolution) (math.Point3, bool) {
	n := pl.Normal()
	ref, bin := c.RefDir.AsVector(), c.Normal.Cross(c.RefDir)
	alpha := c.Radius * float64(n.Dot(ref))
	beta := c.Radius * float64(n.Dot(bin))
	gamma := float64(n.Dot(c.Center.VectorTo(pl.Origin)))
	mag := stdmath.Hypot(alpha, beta)
	if mag < res.Weld() || stdmath.Abs(gamma) > mag+res.Weld() {
		return math.Point3{}, false // circle ∥ plane, or the plane clears it: no crossing
	}
	base, off := stdmath.Atan2(beta, alpha), stdmath.Acos(math.Clamp(gamma/mag, -1, 1))
	return nearerCircleRoot(c, base+off, base-off, near), true
}

// nearerCircleRoot returns the point at whichever of the two circle angles is nearer the far vertex.
func nearerCircleRoot(c geom.Circle, u0, u1 float64, near math.Point3) math.Point3 {
	p0, p1 := c.PointAt(u0/(2*stdmath.Pi)), c.PointAt(u1/(2*stdmath.Pi))
	if p0.DistanceTo(near) <= p1.DistanceTo(near) {
		return p0
	}
	return p1
}

// linePlaneFoot returns the single crossing of a ruling with the capping plane (t = m̂·(o−q₀)/(m̂·d̂)).
func linePlaneFoot(l geom.Line, pl geom.Plane) (math.Point3, bool) {
	n := pl.Normal()
	denom := float64(n.Dot(l.Dir.AsVector())) // = sin(angle ruling↔plane): n̂, d̂ both unit
	if stdmath.Abs(denom) <= sinFloor {
		return math.Point3{}, false // ruling near-parallel to the plane (|n̂·d̂| ≤ sinFloor): no stable crossing
	}
	t := float64(n.Dot(l.Origin.VectorTo(pl.Origin))) / denom
	return l.PointAt(t), true
}
