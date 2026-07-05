// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Shared accept/decline gate for the partial curved-on-planar KIND (#1591, ADR-0049 D-b). Both the partial
// boss (JoinPartialBoss) and the edge scallop (CutEdgeScallop) recognise the SAME contact: a cylinder tool
// meeting a planar face perpendicular to its axis, whose base circle CLIPS the face boundary. This is the
// characterization that routes a contact to the planeUV path instead of the strictly-interior fast-path
// (DrillThroughHole / JoinCylindricalBoss handle the circle wholly inside one face); factoring it here keeps
// the two assemblers from each re-deriving the "planar ∧ perpendicular ∧ pierced-but-not-clean" test.

// planarCapPerpTo returns face f's plane when f is planar and its normal is (anti)parallel to the tool axis ua
// — a cap/seat the perpendicular tool could sit flush on or drill through. ok=false otherwise.
func planarCapPerpTo(f curvedFace, ua math.Vector3) (geom.Plane, bool) {
	pl, isPlane := f.surface.(geom.Plane)
	if !isPlane || stdmath.Abs(float64(unit(pl.Normal()).Dot(ua))) < 1-1e-7 {
		return geom.Plane{}, false
	}
	return pl, true
}

// circleClipsCap reports whether the tool circle of radius `radius` centred at `center` CLIPS face f's
// boundary — pierced (centre inside the face) but NOT clean (the rim crosses out). This is the partial-vs-
// interior discriminator: pierced && clean is the strictly-interior fast-path's job, so the partial planeUV
// path takes only pierced && !clean.
func circleClipsCap(center math.Point3, radius float64, f curvedFace, pl geom.Plane) bool {
	pierced, clean := circleVsCap(center, radius, f, pl)
	return pierced && !clean
}
