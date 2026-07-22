// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Concave analytic miter-arm recognition (advances OCCT blend/simple M3/M9/O2/P2/P3 past
// solveCurvedMiter's former convex-only arm gate). Where the CONVEX miter arm offsets the rolling ball
// into the void with the R−r sign, the CONCAVE (reentrant) miter arm sits in the material valley with
// the R+r sign — the exact mirror of the shipped concaveCylinderArmCandidates / concaveSphereArmSurface
// / concaveConeArmSurface conventions (both offsets flipped OUTWARD, ball centre into the VOID). The
// DISCRIMINANT is ClassifyEdgeConvexity, threaded through miterEdgeArmSurface — never a hardcoded flip;
// the surface TYPE is unchanged (still an exact geom.Torus / geom.Cylinder), only the offset sign moves.
// This greens nothing directly: it lets the 5 concave-rim cases BUILD their arms and flow to the next
// stage (curved seam → multi-corner weld → piece-B shared-face retrim), where they still floor.

// curvedMiterTorusArm builds the torus arm of a Cylinder∧Plane miter edge on the correct material side:
// the convex R−r torusArmSurface (a boss-cap rim) or the concave R+r concaveTorusArmSurface (a reentrant
// base cove). conv is the discriminator, so the convex corpus (P2/P3's top rim, B3) keeps the
// byte-identical R−r arm and only a concave rim (M3/M9/O2) takes R+r.
func curvedMiterTorusArm(cyl geom.Cylinder, pl geom.Plane, outwardN math.UnitVector3, r float64, res Resolution, conv EdgeConvexity) (geom.Surface, bool) {
	if conv == EdgeConcave {
		if t, ok := concaveTorusArmSurface(cyl, pl, outwardN, r, res); ok {
			return t, true
		}
		return nil, false
	}
	if t, ok := torusArmSurface(cyl, pl, outwardN, r, res); ok {
		return t, true
	}
	return nil, false
}

// curvedMiterCylinderArm builds the cylinder arm of an axis-parallel Cylinder∧Plane miter LINE edge
// (config ii) on the correct material side: the convex R−r cylinderArmSurface or the concave R+ε·r
// concaveCylinderArmSurface (O2's two arms on the receded shared cylinder). conv is the discriminator.
func curvedMiterCylinderArm(e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, outwardN math.UnitVector3, r float64, res Resolution, conv EdgeConvexity) (geom.Surface, bool) {
	if conv == EdgeConcave {
		if c, ok := concaveCylinderArmSurface(e, cyl, pl, outwardN, r, res); ok {
			return c, true
		}
		return nil, false
	}
	if c, ok := cylinderArmSurface(e, cyl, pl, outwardN, r); ok {
		return c, true
	}
	return nil, false
}

// concaveTorusArmSurface builds the exact torus arm of a rolling-ball fillet of radius r on a CONCAVE
// (reentrant) circle edge where cylinder cyl meets plane pl with the axis ⊥ the plane — the concave dual
// of torusArmSurface (concave-sphere-cone-arm-derivation.md sign convention). BOTH offsets flip vs the
// convex builder: the major radius R−r → R+r (the ball centre recedes AWAY from the axis, into the void
// outside the wall) and the plane offset −n̂·r → +n̂·r (the ball sits r into the VOID, so the fillet ADDS
// the cove). The minor radius stays r and the NewTorusWithRef frame is byte-identical in form. Returns
// false only for a degenerate torus frame (R+r > 0 always holds, so there is no spindle reject).
//
// Example: concaveTorusArmSurface(bossWall{R:50}, basePlane{z:0,n̂:+ẑ}, +ẑ, 5, res) → torus centre
// (0,0,5), axis ẑ, major 55, minor 5 (the O2 base-rim arm, OCCT Radii 55 5 = R+r).
func concaveTorusArmSurface(cyl geom.Cylinder, pl geom.Plane, outwardN math.UnitVector3, r float64, res Resolution) (geom.Torus, bool) {
	majorR := cyl.Radius + r // offset the ball AWAY from the axis, into the void (convex builder: R − r)
	if majorR < armSpindleBand*res.Weld() {
		return geom.Torus{}, false // defensive: R+r > 0 always, but keep the frame guard
	}
	n := outwardN
	outward := n.AsVector()
	center := projectOntoPlane(cyl.Origin, pl).TranslateBy(outward.Scale(r)) // r into the VOID (convex: −n̂·r)
	tor, err := geom.NewTorusWithRef(center, n.AsVector(), cyl.Ref.AsVector(), majorR, r)
	return tor, err == nil
}

// concaveCylinderArmSurface builds the exact cylinder arm of a rolling-ball fillet of radius r on a
// CONCAVE axis-parallel Cylinder∧Plane LINE edge — the concave dual of cylinderArmSurface, REUSING the
// shipped concave conventions: concaveArmOffsetRadius (ρ = R + ε·r, ε = cylinderHostRadialSign) and
// concaveArmRulingBases (the offset plane pushed +r into the VOID). Of the two P_r∩C_ρ rulings it keeps
// the one nearer the picked edge (nearerRuling), matching the convex miter builder's disambiguation.
// Returns false on the shipped concave spindle/clearance reject or a degenerate arm frame (do-no-harm).
func concaveCylinderArmSurface(e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, outwardN math.UnitVector3, r float64, res Resolution) (geom.Cylinder, bool) {
	rho, err := concaveArmOffsetRadius(e, cyl, r, res)
	if err != nil {
		return geom.Cylinder{}, false
	}
	plus, minus, ok := concaveArmRulingBases(cyl, pl, outwardN, rho, r)
	if !ok {
		return geom.Cylinder{}, false
	}
	base := nearerRuling(e, plus, minus)
	arm, err := geom.NewCylinderWithRef(base, cyl.AxisDir.AsVector(), outwardN.AsVector(), r)
	return arm, err == nil
}
