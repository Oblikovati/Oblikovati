// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// WHERE THE ROLLING BALL SITS ON A CYLINDER/CAP ARC — the two senses resolveArcFillet used to ASSUME.
//
// A constant-radius blend on a cylinder/plane arc is a ball of radius r tangent to BOTH surfaces, so its
// centre is r off the cap plane and cylR∓r off the cylinder axis. Which way, on each of those two counts,
// is NOT free — and until this file the arc engine hard-coded one answer to each:
//
//	torusCenter = capCenter − pl.Normal()·r        (the STORED plane normal, not the outward one)
//	majorR      = cyl.Radius − r                   (the ball INSIDE the cylinder)
//
// Both are the convex-shaft special case, and the corpus contains two counter-examples that DRAWEXE 8.0.0
// convicts independently (both re-derived in .superpowers/sdd/w2h6-runout-report.md §2):
//
//   - simple/W2 — its cap face is Reversed, so the stored normal {0,1,0} points INTO the material and the
//     band was pushed to y = −0.2, wholesale into the void; and its cylinder is a GROOVE (material lies
//     OUTSIDE it), so the ball rides at cylR+r = 1.2. OCCT's own band is a torus at (3, 0.2, 0.9999),
//     R = 1.2, r = 0.2.
//   - simple/H6 — its cap plane's stored normal IS outward, but the picked arc is CONCAVE (the blend adds
//     material), so the ball sits on the VOID side of the cap and rides at cylR+r = 60. OCCT's own band is
//     a torus at (0, 0, −40), R = 60, r = 10. The previous slice's probe accepted H6's WRONG seat because
//     it asked only "is the ball centre inside the body?" — at (0,0,−60) the ball is buried in the lower
//     CONE, which is inside the body and still completely wrong. The predicate has to be signed by the
//     edge's convexity, which is what arcBallInMaterial does.
//
// The seat is therefore solved, not assumed: enumerate the four candidates in resolveRim's own ladder
// order and take the first whose ball centre sits where the edge's convexity says it must.

// arcBallSeat is one admissible seat for the rolling ball: which way the torus centre lies off the cap
// plane, and how far the ball centre rides from the cylinder axis.
type arcBallSeat struct {
	capSide math.Vector3 // unit, cap plane → torus centre
	majorR  float64      // ball-centre distance from the cylinder axis (cylR−r or cylR+r)
}

// arcBallSeats enumerates the four candidate seats in resolveRim's ladder order: the established
// convex-shaft derivation first (so a case that was already right stays byte-identical), then the R+r
// mirror, then the same pair on the far side of the cap. A non-positive majorR is not a seat — that is
// where the old `r >= cyl.Radius` rejection now lives, and it is now a CONVEX-tier fact rather than a
// blanket one: a groove or a concave corner admits r ≥ cylR perfectly well at cylR+r.
func arcBallSeats(nOut math.Vector3, cylR, r float64) []arcBallSeat {
	out := make([]arcBallSeat, 0, 4)
	for _, side := range []math.Vector3{nOut.Negate(), nOut} {
		for _, major := range []float64{cylR - r, cylR + r} {
			if major > 0 {
				out = append(out, arcBallSeat{capSide: side, majorR: major})
			}
		}
	}
	return out
}

// solveArcBallSeat returns the seat whose ball centre lies where the picked edge's own convexity says the
// rolling ball must be — inside the material for a convex edge (the blend REMOVES material), in the void
// for a concave one (it ADDS material). The probe point is the ball centre at the arc's own MIDPOINT
// azimuth: an end's radial lies on the side plane the arc terminates against, where inside/outside is a
// boundary coin-flip, and a sector solid has no material at an arbitrary azimuth at all.
//
// Example:
//
//	seat, err := solveArcBallSeat(body, arcEdge, capF, cyl, pl, capCentre, refMid, 0.2)
func solveArcBallSeat(b *topo.Body, e *topo.Edge, capF *topo.Face, cyl geom.Cylinder, pl geom.Plane,
	capCenter math.Point3, refMid math.UnitVector3, r float64) (arcBallSeat, error) {
	wantMaterial, err := arcBallInMaterial(e)
	if err != nil {
		return arcBallSeat{}, err
	}
	inside := brep.NewInsideQuery(b) // one prepared analytic query for up to four probes
	for _, s := range arcBallSeats(outwardPlaneNormal(capF, pl), cyl.Radius, r) {
		centre := capCenter.TranslateBy(s.capSide.Scale(r))
		if inside.Inside(centre.TranslateBy(refMid.AsVector().Scale(s.majorR))) == wantMaterial {
			return s, nil
		}
	}
	return arcBallSeat{}, fmt.Errorf("fillet: arc radius %g finds no rolling-ball seat on cylinder radius %g "+
		"(cap centre %v: no candidate at cylR∓r on either side of the cap holds the ball on the %s side)",
		r, cyl.Radius, capCenter, materialSideName(wantMaterial))
}

// arcBallInMaterial reports whether the rolling ball must sit INSIDE the material — true for a convex
// edge (the blend removes material), false for a concave one (it adds it). A tangent or unclassifiable
// edge is not a blend seed at all and is rejected rather than guessed.
func arcBallInMaterial(e *topo.Edge) (bool, error) {
	switch c := ClassifyEdgeConvexity(e); c {
	case EdgeConvex:
		return true, nil
	case EdgeConcave:
		return false, nil
	default:
		return false, fmt.Errorf("fillet: arc edge convexity is %v, need convex or concave to seat the "+
			"rolling ball (a tangent or non-manifold edge has no blend side)", c)
	}
}

// materialSideName names the side an unseatable ball was looked for on, for the decline message.
func materialSideName(inMaterial bool) string {
	if inMaterial {
		return "material"
	}
	return "void"
}

// arcMidRadial is the radial direction from the cylinder axis to the arc's own MIDPOINT — the azimuth
// every side-independent probe and frame question is asked at.
func arcMidRadial(e *topo.Edge, capCenter math.Point3, axis math.UnitVector3) (math.UnitVector3, error) {
	lo, hi := e.Geometry().Domain()
	mid := e.Geometry().PointAt((lo + hi) / 2)
	ref, err := math.UnitVector3FromVector(perpComponent(capCenter.VectorTo(mid), axis))
	if err != nil {
		return math.UnitVector3{}, fmt.Errorf("fillet: arc midpoint %v is on the cylinder axis — degenerate frame", mid)
	}
	return ref, nil
}
