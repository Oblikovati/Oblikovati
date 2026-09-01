// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// WHICH SIDE OF THE CAP THE CLOSED RIM'S ROLLING BALL SITS ON — the assumption solveRim used to make.
//
// solveRim seated the ball by `inward := pl.Normal().Negate()`: the STORED plane normal, taken as
// evidence of the material side. It is not. A planar face carries a Reversed flag precisely because its
// stored normal may point INTO the material (kernel/ops/fillet.go's outwardPlaneNormal exists for that
// reason), and the arc engine was convicted of the identical assumption one slice ago: simple/W2 shipped
// an entire band into the void off a Reversed cap (.superpowers/sdd/w2h6-runout-report.md §1.2).
//
// On the CLOSED rim the same input is one fixture away. It does not ship a bad band on any corpus case
// today only because the tier-1 material-side probe happens to land OUTSIDE the body on both live
// Reversed-cap rims (simple/K1, simple/Z1), so resolveRim's cap-orientation ladder catches them. That is
// luck, not a guard: when the far side of a Reversed cap happens to carry material at the ball-centre
// radius — a spool, a plate under an overhanging head — the unguarded seat passes its own probe and the
// band is built on the void side of the cap, with its cylinder-tangent rail hanging outside the solid.
// TestReversedCapClosedRimSeatsOnTheMaterialSide builds exactly that solid.
//
// THE LAW, and it is the arc engine's law (w2h6-runout-report.md §1.2): the rolling ball is tangent to
// both surfaces and lies INSIDE the material iff the picked edge is CONVEX. So with MAT_in the
// material-inward direction across the cap and CVX = +1 convex / −1 concave, the centre sits at
//
//	capCentre + (CVX · MAT_in) · r
//
// and MAT_in comes from outwardPlaneNormal — the Reversed-aware normal, the one piece of evidence the
// stored normal is not.
//
// ★ AND THE VERIFICATION MUST BE SIGNED. An unsigned "is the ball centre inside the body?" is NOT
// sufficient — that was measured, not assumed: it ACCEPTS simple/H6's wrong seat, whose ball at radius 40,
// z = −60 is buried inside the lower cone and still completely wrong (w2h6-runout-report.md §9.4). So the
// seat is verified with the SHIPPED signed predicate — arcBallInMaterial + brep.PointInside, the pair
// solveArcBallSeat already uses — asking whether the centre is on the side the edge's own convexity
// demands, not merely whether it is somewhere in the solid.

// rimCapSeat is the verified seat solveRim builds its convex R−r frame on: the unit direction from the
// cap plane to the rolling-ball centre, and the band axis that frame implies (material-OUTWARD along the
// cap normal, the convention solveRim and rimWithCapOrientation already share).
type rimCapSeat struct {
	toCentre math.Vector3
	bandAxis math.Vector3
	// inMaterial is the SIGN the seat was verified under: true when the picked edge is convex (the ball
	// sits in the material), false when it is concave (the ball sits in the void). solveRim reads it as
	// its convex-tier gate — see the errConvexRimProbeFailed return there.
	inMaterial bool
}

// rimBallCapSeat derives the cap side from the Reversed-aware outward normal, signs it with the picked
// edge's convexity, and VERIFIES it with the signed material-side probe before solveRim will build on it.
// It returns errConvexRimProbeFailed (wrapped) when the seat cannot be verified, which is solveRim's own
// decline signal: resolveRim then hands the rim to rimWithCapOrientation, the sibling ladder that
// re-derives the WHOLE frame — centre, band axis and seam — from the true outward normal. Declining is
// deliberate: flipping only the centre while keeping a stored band axis would mirror the torus's own v
// parameterisation against its seam.
//
// Example:
//
//	seat, err := rimBallCapSeat(body, rimEdge, capFace, plane, capCentre, ref, cyl.Radius-r, r)
func rimBallCapSeat(b *topo.Body, e *topo.Edge, capF *topo.Face, pl geom.Plane,
	capCenter math.Point3, ref math.UnitVector3, majorR, r float64) (rimCapSeat, error) {
	inMaterial, err := arcBallInMaterial(e)
	if err != nil {
		return rimCapSeat{}, fmt.Errorf("%v: %w", err, errConvexRimProbeFailed)
	}
	seat, err := rimCapSeatFromOutwardNormal(capF, pl, inMaterial)
	if err != nil {
		return rimCapSeat{}, err
	}
	centre := capCenter.TranslateBy(seat.toCentre.Scale(r))
	if brep.PointInside(b, centre.TranslateBy(ref.AsVector().Scale(majorR))) != inMaterial {
		return rimCapSeat{}, fmt.Errorf(
			"fillet: rim cap seat at %v is not on the %s side the edge's convexity requires: %w",
			centre, materialSideName(inMaterial), errConvexRimProbeFailed)
	}
	return seat, nil
}

// rimCapSeatFromOutwardNormal applies the roll-sense law s = CVX·MAT_in to the cap's Reversed-aware
// outward normal. The band axis is taken from outwardPlaneNormal VERBATIM rather than from a normalized
// round-trip, so a cap whose stored normal already IS the outward normal reproduces solveRim's historic
// frame bit for bit (outwardPlaneNormal returns pl.Normal() itself on an un-Reversed face).
func rimCapSeatFromOutwardNormal(capF *topo.Face, pl geom.Plane, inMaterial bool) (rimCapSeat, error) {
	nOut := outwardPlaneNormal(capF, pl)
	outward, err := math.UnitVector3FromVector(nOut)
	if err != nil {
		return rimCapSeat{}, fmt.Errorf("fillet: degenerate rim cap normal %v: %w", pl.Normal(), errConvexRimProbeFailed)
	}
	toCentre := outward.Negate().AsVector() // MAT_in
	if !inMaterial {
		toCentre = outward.AsVector() // a concave rim's ball sits on the VOID side of the cap
	}
	return rimCapSeat{toCentre: toCentre, bandAxis: nOut, inMaterial: inMaterial}, nil
}
