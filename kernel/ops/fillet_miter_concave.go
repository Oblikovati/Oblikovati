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
// torusArmSurface's R−ε·r (a boss-cap rim, ε=+1) or concaveTorusArmSurface's R+ε·r (a reentrant cove),
// with ε from cylinderHostRadialSign(e, cyl) — the SAME per-wall boss/notch sign the W-DH wave wired into
// the plain concave rim arm (M5: ε=+1 boss cove byte-identical, ε=−1 notch cove). Earlier this arm ALWAYS
// took the boss sign (ε=+1 hardcoded on both branches, "bore-miter is a later slice"), which is wrong on
// simple/W3, W4's boss-NOTCH corner (a slot cut INTO a cylindrical boss, not a boss-base cove): DRAWEXE's
// own torus arm there is Radii 1.2/0.2 = R+r on host R=1, i.e. ε=−1, while the hardcoded-ε=+1 build shipped
// R−r=0.8 — a wrong analytic surface that still tessellated and welded (watertight, silently ~5% short in
// AREA). conv is still the convex/concave discriminator; ε is orthogonal to it (a concave-classified EDGE
// can sit on a boss OR a notch wall, and so can a convex one — cylinderHostRadialSign reads the wall, not
// the edge). ok=false when ε cannot be read (a degenerate edge midpoint/normal) — the do-no-harm floor.
func curvedMiterTorusArm(e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, outwardN math.UnitVector3, r float64, res Resolution, conv EdgeConvexity) (geom.Surface, bool) {
	eps, ok := cylinderHostRadialSign(e, cyl)
	if !ok {
		return nil, false
	}
	if conv == EdgeConcave {
		if t, ok := concaveTorusArmSurface(cyl, pl, outwardN, r, eps, res); ok {
			return t, true
		}
		return nil, false
	}
	if t, ok := torusArmSurface(cyl, pl, outwardN, r, eps, res); ok {
		return t, true
	}
	return nil, false
}

// curvedMiterCylinderArm builds the cylinder arm of an axis-parallel Cylinder∧Plane miter LINE edge
// (config ii) on the correct material side: the concave branch already reads ε off the wall via
// concaveArmOffsetRadius/cylinderHostRadialSign (O2's two arms on the receded shared cylinder); the
// convex branch now does too (cylinderArmSurface's R−ε·r), the same boss/notch generalization
// curvedMiterTorusArm above carries — a convex LINE edge on a notch wall needs ρ=R+r exactly as the
// circle-edge torus arm does. conv is the convex/concave discriminator; ok=false when ε cannot be read.
func curvedMiterCylinderArm(e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, outwardN math.UnitVector3, r float64, res Resolution, conv EdgeConvexity) (geom.Surface, bool) {
	if conv == EdgeConcave {
		if c, ok := concaveCylinderArmSurface(e, cyl, pl, outwardN, r, res); ok {
			return c, true
		}
		return nil, false
	}
	eps, ok := cylinderHostRadialSign(e, cyl)
	if !ok {
		return nil, false
	}
	if c, ok := cylinderArmSurface(e, cyl, pl, outwardN, r, eps); ok {
		return c, true
	}
	return nil, false
}

// concaveTorusArmSurface builds the exact torus arm of a rolling-ball fillet of radius r on a CONCAVE
// (reentrant) circle edge where cylinder cyl meets plane pl with the axis ⊥ the plane — the concave dual
// of torusArmSurface (concave-sphere-cone-arm-derivation.md sign convention). BOTH offsets flip vs the
// convex builder: the major radius R−r → R+ε·r (the ball centre recedes into the void — AWAY from the
// axis on a BOSS wall, ε=+1, or TOWARD it inside a NOTCH/BORE wall, ε=−1) and the plane offset
// −n̂·r → +n̂·r (the ball sits r into the VOID, so the fillet ADDS the cove). ε is cylinderHostRadialSign's
// n_C·r̂ read — the SAME per-wall sign the concave line arm (concaveArmOffsetRadius) has always used; the
// boss case (ε=+1) is byte-identical to the historical R+r arm. The minor radius stays r and the
// NewTorusWithRef frame is byte-identical in form. Returns false for a degenerate torus frame or a notch
// spindle (ε=−1 with r ≥ R: the ball-centre circle collapses onto the axis — DRAWEXE M5 ground truth:
// the notch-ceiling cove arm is maj=R−r=25, the corner a plain r-sphere on that spine).
//
// Example: concaveTorusArmSurface(bossWall{R:50}, basePlane{z:0,n̂:+ẑ}, +ẑ, 5, +1, res) → torus centre
// (0,0,5), axis ẑ, major 55, minor 5 (the O2 base-rim arm, OCCT Radii 55 5 = R+r).
func concaveTorusArmSurface(cyl geom.Cylinder, pl geom.Plane, outwardN math.UnitVector3, r, eps float64, res Resolution) (geom.Torus, bool) {
	majorR := cyl.Radius + eps*r // ball centre r into the void: outside a boss wall (+r), inside a notch wall (−r)
	if majorR < armSpindleBand*res.Weld() {
		return geom.Torus{}, false // notch spindle (R−r reaches the axis) or degenerate frame guard
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
