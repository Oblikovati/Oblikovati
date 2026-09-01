// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// M5 Slice A (m5-curved-arm-derivation.md): the rolling-ball fillet on a Plane∧Cylinder edge is
// an exact torus or an exact cylinder — never a general canal surface — for the axis-aligned
// cases the corpus needs today. Which one it is turns entirely on the angle between the
// cylinder axis and the plane normal; classifyCurvedArm is that decision, split out so the
// (not-yet-written) arm builders can each stay a single-responsibility constructor for their
// one case.

// armKind is which exact rolling-ball arm surface fits a Plane∧Cylinder edge, decided by the
// axis-plane angle s = â·n̂_P (m5-curved-arm-derivation.md §"The three configurations"):
// axis ⊥ plane (|s|≈1) makes the edge a circle and the arm an exact torus (config i); axis ∥
// plane (s≈0) makes the edge a line and the arm an exact cylinder (config ii); anything oblique
// makes the edge an ellipse, which has no closed-form arm yet and is rejected (config iii,
// Slice B — the general canal/pipe surface, D4).
type armKind int

const (
	armRejected armKind = iota // oblique: ellipse edge, no closed-form arm (deferred, D4)
	armTorus                   // axis ⊥ plane: circle edge, exact torus arm (D2)
	armCylinder                // axis ∥ plane: line edge, exact cylinder arm (D3)
)

// armRejected's config-iii case (a plain geom.Cylinder wall obliquely capped, so the rim edge is a
// geom.EllipticalArc) is cluster W-F's F2 — complex/C1, complex/F1, simple/R7. Investigated but NOT
// built (out of scope for this pass; verified premise, not a newly-discovered gap): all three rims
// are OPEN arcs (StartVertex != EndVertex), not closed loops, so the plan is NOT to route through
// the closed-rim family (fillet_elliptic_rim_canal.go / fillet_rim.go — both closed-loop-only) but
// to add a THIRD armKind here that plugs into the SAME single-edge run-in/run-out+corner machinery
// armTorus/armCylinder already use (edgeFillet.armSurface is surface-agnostic; J6/J8 already prove a
// BSplineSurface flows through it unchanged).
//
// Derivation (no new closed form needed — the existing plane-cap elliptic-rim spine is ALREADY
// generic over the wall's ellipticity): synthesize a geom.EllipticalCylinder with
// MajorRadius=MinorRadius=cyl.Radius, Origin/AxisDir from cyl, ANY perpendicular Ref (degenerate
// when major=minor, so the choice is free) and feed it straight into
// ellipticRimSpine/newEllipticRimSpine/station() (fillet_elliptic_rim_spine.go) unchanged — a
// circular cylinder capped by an oblique plane is mathematically the SAME translational-sweep-wall
// construction the elliptic-cylinder-wall/plane-cap rim already solves in closed form, just with the
// ellipticity on the CAP side (the rim) instead of the WALL side. Unlike the cone-cap family
// (fillet_elliptic_cone_spine.go), a plane can never be exactly tangent to a cylinder along a
// ruling, so there is NO pinch/degenerate-root case to handle — every station is a plain linear
// solve, strictly simpler. Sample stations over the arc's own [u0,u1] (open, not 2π), loft with the
// ordinary geom.LoftCanalStations (not the Pinched variant), and terminate each end wherever the
// arc's own end vertex already sits (a plain run-in/run-out, no runout-plane imprint like B4/B8 —
// there is no host-tangency limit to taper into). Wire it in as a new armKind returned by
// classifyCurvedArm's default branch (gated on the rim edge actually being geom.EllipticalArc, so a
// genuinely-unhandled oblique case — if one exists — still declines). Per-face DRAWEXE
// reconciliation is mandatory before trusting any of the three cases; simple/R7 is a 16-edge
// multi-pick chain (only ONE of its edges is this oblique rim) — verify the chain interaction, not
// just the lone edge, before greening it.

// angArmClassifyCoef is k in ε_ang ≈ k·res.Weld()/R (§Numerical pitfalls): the classification
// band is an ANGULAR tolerance derived from the model's weld resolution divided by the edge's
// arc radius, never a bare constant — the same |s| slack that is negligible on a small cylinder
// is a visible misclassification on a huge one. k=3 sits mid-band of the derivation's k≈2..4.
const angArmClassifyCoef = 3

// classifyCurvedArm decides which exact rolling-ball arm surface fits the Plane∧Cylinder edge
// (cyl, pl): armTorus when the cylinder axis is (near-)perpendicular to the plane, armCylinder
// when it is (near-)parallel, armRejected when oblique or when cyl is degenerate (Radius ≤ 0).
// res scales the classification band to the model (ADR-0042) — see angArmClassifyCoef.
//
// Example: classifyCurvedArm(bossWall, topPlane, res) on a boss's top-rim edge (axis ⊥ top
// plane) returns armTorus.
func classifyCurvedArm(cyl geom.Cylinder, pl geom.Plane, res tol.Resolution) armKind {
	if cyl.Radius <= 0 {
		return armRejected
	}
	n, err := math.UnitVector3FromVector(pl.Normal())
	if err != nil {
		return armRejected
	}
	s := stdmath.Abs(cyl.AxisDir.Dot(n))
	epsAng := angArmClassifyCoef * res.Weld() / cyl.Radius
	switch {
	case s > 1-epsAng:
		return armTorus
	case s < epsAng:
		return armCylinder
	default:
		return armRejected
	}
}
