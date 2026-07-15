// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
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
func classifyCurvedArm(cyl geom.Cylinder, pl geom.Plane, res Resolution) armKind {
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
