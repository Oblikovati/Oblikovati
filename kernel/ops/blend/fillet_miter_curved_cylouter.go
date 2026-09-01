// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// Concave curved-miter seam bottom for a CYLINDER torus-outer host (M3/M9) — the MIRROR of the plane
// branch's host roles. For P5 (convex) the shared face is a Cylinder and the torus arm's outer host is
// a Plane, so miterSeamBottomPlane builds sBot on a plane contact circle. For M3/M9 (concave base
// cove) the shared face is the box-top Plane and the torus arm's outer host is the cylinder WALL, so
// sBot lives on the torus↔host-cylinder CONTACT CIRCLE instead. The two branches are additive: the
// dispatcher in fillet_miter_curved.go keys on torOuter's surface type, so P5's convex seam is
// byte-identical (curved-miter-seam-recon.md §1; DRAWEXE-validated M3 sBot=(20,15,105)).

// miterSeamBottomCyl is sBot for a CYLINDER torus-outer host (M3/M9). The torus arm is coaxial with
// that host cylinder, so it is tangent to it along the CONTACT CIRCLE: centre tor.Center, radius
// hostCyl.Radius, in the plane through the centre ⟂ tor.AxisDir. sBot is the point of that circle at
// distance r (the arm radius) from the cyl-ARM axis line nearest the corner vertex vp — the physical
// branch (curved-miter-seam-derivation.md §1d). ok=false when the host is not the torus's coaxial
// cylinder, or the contact circle never reaches distance r from the arm axis (no crossing).
func miterSeamBottomCyl(arms curvedMiterArms, hostCyl geom.Cylinder, vp math.Point3, res opstol.Resolution) (math.Point3, bool) {
	if !torusCoaxialWithHost(arms.tor, hostCyl, res) {
		return math.Point3{}, false
	}
	e1 := arms.tor.Ref.AsVector()
	e2 := arms.tor.AxisDir.AsVector().Cross(e1)
	r := arms.cyl.Radius
	axisPt, axisDir := arms.cyl.Origin, arms.cyl.AxisDir.AsVector()
	membership := func(a float64) float64 {
		p := contactCirclePoint(arms.tor.Center, hostCyl.Radius, e1, e2, a)
		return distToLineSq(p, axisPt, axisDir) - r*r
	}
	return nearestCircleRoot(membership, arms.tor.Center, hostCyl.Radius, e1, e2, vp)
}

// torusCoaxialWithHost reports whether host is the torus's coaxial outer cylinder: parallel axes and
// the torus centre lying on the host axis (within the model-relative weld). The cylinder-outer sBot
// branch fires ONLY for such a host — an unrelated cylinder outer face keeps flooring (do-no-harm).
func torusCoaxialWithHost(tor geom.Torus, host geom.Cylinder, res opstol.Resolution) bool {
	if stdmath.Abs(float64(tor.AxisDir.Dot(host.AxisDir)))-1 < -res.Weld() {
		return false
	}
	tol := res.Weld() * host.Radius
	return distToLineSq(tor.Center, host.Origin, host.AxisDir.AsVector()) <= tol*tol
}

// nearestCircleRoot returns the contact-circle point (centre c, radius rho, in-plane orthonormal axes
// e1,e2) at a sign change of membership nearest vp — the physical sBot branch. ok=false when membership
// never changes sign (the circle stays wholly inside or outside distance r; no crossing).
func nearestCircleRoot(membership func(float64) float64, c math.Point3, rho float64, e1, e2 math.Vector3, vp math.Point3) (math.Point3, bool) {
	best, found := math.Point3{}, false
	for _, a := range circleSignChanges(membership) {
		p := contactCirclePoint(c, rho, e1, e2, a)
		if !found || p.DistanceTo(vp) < best.DistanceTo(vp) {
			best, found = p, true
		}
	}
	return best, found
}

// circleSignChanges returns the angles in [0,2π) where membership changes sign, sampling the circle and
// bisecting each bracket — the contact-circle∩arm-tube crossings (the quartic membership has up to 4
// transversal roots, well-separated for the coaxial cove geometry). Tangencies (no sign change) are
// skipped as non-physical: the physical sBot is a transversal crossing.
func circleSignChanges(membership func(float64) float64) []float64 {
	const steps = 240
	var out []float64
	prev := membership(0)
	for i := 1; i <= steps; i++ {
		a0 := 2 * stdmath.Pi * float64(i-1) / float64(steps)
		a1 := 2 * stdmath.Pi * float64(i) / float64(steps)
		cur := membership(a1)
		if prev == 0 || prev*cur < 0 {
			out = append(out, bisectAxial(membership, a0, a1))
		}
		prev = cur
	}
	return out
}

// contactCirclePoint is the point at angle a on the circle centre c, radius rho, with in-plane
// orthonormal axes e1,e2: c + rho·(cos a·e1 + sin a·e2).
func contactCirclePoint(c math.Point3, rho float64, e1, e2 math.Vector3, a float64) math.Point3 {
	radial := e1.Scale(math.Scalar(stdmath.Cos(a))).Add(e2.Scale(math.Scalar(stdmath.Sin(a))))
	return c.TranslateBy(radial.Scale(math.Scalar(rho)))
}

// distToLineSq is the squared distance from p to the line through a along the unit direction d.
func distToLineSq(p, a math.Point3, d math.Vector3) float64 {
	w := a.VectorTo(p)
	perp := w.Sub(d.Scale(w.Dot(d)))
	return float64(perp.Dot(perp))
}
