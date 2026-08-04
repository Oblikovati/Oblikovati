// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Equal-radius parallel-axis Cylinder∧Cylinder miter arm (family B, e.g. P5's vertical seam edge).
// The two host cylinders share a radius and a direction; a rolling ball of radius r tangent to both
// has its centre on EACH host's ρ=R±r coaxial offset cylinder, and the two offset cylinders meet in a
// pair of rulings (lines parallel to the axis) — one of which is the arm axis. The offset SIGN per
// host is the discriminator the "|axis−sharedAxis| = R−r" distance test cannot see
// (curved-miter-closure-derivation.md §2): a bore wall offsets to R−r, a boss wall to R+r, so P5's
// shared bore (ρ=45) ∩ outer boss (ρ=55) yields the concave–convex arm axis (48.333,5.031) — NOT the
// symmetric R−r∩R−r branch (65,7.574), which sits on the wrong side of the shared cylinder and cannot
// close the miter seam to its host contact rails.

// equalParallelCylMiterArm builds the cylinder arm of an equal-radius parallel-axis Cylinder∧Cylinder
// miter edge (derivation D3, corrected per closure §2): each host's ball-centre locus is its own
// ρ=R±r coaxial offset cylinder, and the arm axis is the ruling of their intersection nearer the picked
// edge, radius r. conv is the discriminator: a CONVEX edge (P5's boss∩bore) offsets each wall R−ε·r
// (void side), a CONCAVE edge (P2/P3's fused-boss VALLEY) offsets each wall R+ε·r (tangent-external,
// both R+r). ok=false for non-equal radii, non-parallel axes, or when the offset cylinders do not meet.
func equalParallelCylMiterArm(e *topo.Edge, r float64, res Resolution, conv EdgeConvexity) (geom.Surface, bool) {
	faces := e.Faces()
	if len(faces) != 2 {
		return nil, false
	}
	cA, okA := faces[0].Geometry().(geom.Cylinder)
	cB, okB := faces[1].Geometry().(geom.Cylinder)
	if !okA || !okB {
		return nil, false
	}
	base, dir, ok := equalParallelArmRuling(e, cA, cB, r, res, conv)
	if !ok {
		return nil, false
	}
	arm, err := geom.NewCylinderWithRef(base, dir, base.VectorTo(e.StartVertex().Point().Midpoint(e.EndVertex().Point())), r)
	return arm, err == nil
}

// equalParallelArmRuling returns the arm-axis ruling (a base point and the shared axis direction) of
// two equal-radius parallel-axis cylinders' offset intersection, choosing the ruling nearer the picked
// edge's midpoint. Each host's offset radius is R∓ε·r with ε from cylinderHostRadialSign and the sign
// from conv (convex: R−ε·r, closure-§2 branch fix; concave valley: R+ε·r). ok=false when the radii/axes
// disqualify, a host normal is unreadable, or the offset circles miss.
func equalParallelArmRuling(e *topo.Edge, cA, cB geom.Cylinder, r float64, res Resolution, conv EdgeConvexity) (math.Point3, math.Vector3, bool) {
	d := cA.AxisDir.AsVector()
	tol := res.Weld() * cA.Radius
	if stdmath.Abs(float64(cA.AxisDir.Dot(cB.AxisDir)))-1 < -res.Weld() || stdmath.Abs(cA.Radius-cB.Radius) > tol {
		return math.Point3{}, math.Vector3{}, false // non-parallel axes or unequal radii — not this case
	}
	rhoA, okA := cylinderOffsetRadius(e, cA, r, conv)
	rhoB, okB := cylinderOffsetRadius(e, cB, r, conv)
	if !okA || !okB {
		return math.Point3{}, math.Vector3{}, false
	}
	plus, minus, ok := intersectCoplanarCircles(cA.Origin, rhoA, cB.Origin, rhoB, d, res)
	if !ok {
		return math.Point3{}, math.Vector3{}, false
	}
	return nearerRuling(e, plus, minus), d, true
}

// cylinderOffsetRadius is the ball-centre offset radius ρ of a rolling ball tangent to the cylinder host
// of e, ε = cylinderHostRadialSign (bore −1, boss +1). For a CONVEX miter edge ρ = R − ε·r (the void
// side, NEGATED vs the concave-arm +ε·r convention): P5's shared bore-side wall gives R−r=45 and its
// outer boss-side wall R+r=55 — the concave–convex branch (axis 48.333,5.031), the discriminator the
// |axis−sharedAxis|=R−r test cannot see (curved-miter-closure-derivation.md §2). For a CONCAVE miter
// edge (P2/P3's fused-boss valley) ρ = R + ε·r (tangent-external: both bosses ε=+1 give R+r=55, the ball
// riding the valley OUTSIDE both walls). Because the cylinder normal is exactly radial, ρ is exact.
// ok=false when the host face carries no readable outward normal.
func cylinderOffsetRadius(e *topo.Edge, cyl geom.Cylinder, r float64, conv EdgeConvexity) (float64, bool) {
	eps, ok := cylinderHostRadialSign(e, cyl)
	if !ok {
		return 0, false
	}
	if conv == EdgeConcave {
		return cyl.Radius + eps*r, true // concave fused-boss valley: both walls offset R+ε·r (P2/P3)
	}
	return cyl.Radius - eps*r, true // convex miter: void side (P5's bore R−r ∩ boss R+r)
}

// intersectCoplanarCircles returns the two intersection points of circles (c1,r1) and (c2,r2) that lie
// in a common plane with unit normal n — the general two-radii circle∩circle (closure §2 needs r1≠r2
// for the concave–convex branch). Any out-of-plane component of c1→c2 is stripped first, so callers may
// pass centres offset along n (e.g. cylinder origins at different axial heights: the returned points
// then seed rulings ∥ n). ok=false when the centres coincide in-plane (sep below the weld floor) or the
// circles miss (radicand<0).
func intersectCoplanarCircles(c1 math.Point3, r1 float64, c2 math.Point3, r2 float64, n math.Vector3, res Resolution) (math.Point3, math.Point3, bool) {
	w := c1.VectorTo(c2)
	wPerp := w.Sub(n.Scale(w.Dot(n)))
	sep := float64(wPerp.Length())
	u, err := math.UnitVector3FromVector(wPerp)
	if sep < res.Weld() || err != nil {
		return math.Point3{}, math.Point3{}, false
	}
	a := (sep*sep + r1*r1 - r2*r2) / (2 * sep)
	h2 := r1*r1 - a*a
	if h2 < 0 {
		return math.Point3{}, math.Point3{}, false
	}
	foot := c1.TranslateBy(u.AsVector().Scale(math.Scalar(a)))
	side := u.AsVector().Cross(n)
	h := stdmath.Sqrt(h2)
	return foot.TranslateBy(side.Scale(math.Scalar(h))), foot.TranslateBy(side.Scale(math.Scalar(-h))), true
}

// cylCylMiterArmEdge builds the exact cylinder arm of an equal-radius parallel-axis Cylinder∧Cylinder
// convex edge (family B, P5's vertical seam edge) so computeEdgeFillet no longer errors on it. It fires
// ONLY for that specific pairing (equal radii, parallel axes, offset rulings that meet); any other
// cyl∩cyl edge returns handled=false and keeps its former honest reject (do-no-harm). handled=true
// means this owns the edge and the returned edgeFillet carries the cylinder arm.
func cylCylMiterArmEdge(body *topo.Body, e *topo.Edge, p filletPick) (edgeFillet, bool) {
	faces := e.Faces()
	if len(faces) != 2 {
		return edgeFillet{}, false
	}
	_, okA := faces[0].Geometry().(geom.Cylinder)
	_, okB := faces[1].Geometry().(geom.Cylinder)
	if !okA || !okB || ClassifyEdgeConvexity(e) != EdgeConvex || p.varying() {
		return edgeFillet{}, false
	}
	arm, ok := equalParallelCylMiterArm(e, p.r0, ResolutionForBody(body), EdgeConvex)
	if !ok {
		return edgeFillet{}, false // not an equal-parallel pair — fall through to the honest reject
	}
	return curvedArmEdgeFillet(e, arm, true)
}
