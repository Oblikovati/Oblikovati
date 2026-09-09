// SPDX-License-Identifier: GPL-2.0-only

package brep

import "oblikovati.org/kernel/geom"

// Curved boolean imprint (M2 Phase 1, Oblikovati/Oblikovati#1334). The imprint stage of the
// curved boolean asks, for a pair of faces, "where do their surfaces meet?". For the planar
// boolean that is plane∩plane → a line segment (imprint, boolean.go). Lifted to curved faces
// it is surface∩surface → the exact section: plane∩cylinder ⟂ axis → a circle, plane∩sphere →
// a circle, plane∩cone ⟂ axis → a circle, oblique plane∩cylinder → an ellipse, plane∩plane → a
// line, ruled∩quadric → a RuledQuadricArc, torus∩quadric → the harmonic loops.
//
// It is the ONE seam through which this package asks geom for a face pair's section, so every pairing
// that declines does so with the SAME named reason (Oblikovati/Oblikovati#3525). Before, each pairing
// called the intersector itself and several threw the reason away, which is how a torus pair, an
// ill-conditioned lane and a section that does not close all reached the user as one generic
// "no exact analytic path claims this configuration".

// curvedImprint returns the analytic intersection curve(s) of two faces' surfaces, the NAMED reason it
// refused, and whether the pair was handled in closed form. handled == false means no analytic solver
// applies (a torus against a torus, a NURBS face) or one applies and cannot use its own answer here —
// `why` says which, and the caller records it (recordSectionDecline) before it declines. handled ==
// true with an empty slice means the surfaces provably do not cross (parallel planes, a sphere clear of
// the plane, a tangent touch).
//
// Example — a cylinder side cut by a plane perpendicular to its axis yields one circle:
//
//	side, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
//	lid, _ := geom.NewPlane(math.P3(0, 0, 2.5), math.V3(0, 0, 1))
//	curves, why, ok := curvedImprint(side, lid, res) // ok, one geom.Circle, why=DeclineNone
//
// res is the model-relative coincidence scale (Oblikovati/Oblikovati#1399): the analytic
// clearance/grazing tests that decide exact-conic-vs-defer derive their tolerance from it, so
// the imprint classifies a µm or km copy of the same pair identically.
func curvedImprint(a, b geom.Surface, res geom.Resolution) ([]geom.Curve3, geom.SectionDecline, bool) {
	return geom.IntersectSurfacesAnalyticDeclining(a, b, res)
}

// sectionClosureGap is the widest endpoint gap of the curves an intersector returned: how far the
// worst of them is from coming back to where it started. It is the metric behind
// [geom.DeclineOpenSection], returned as a NUMBER rather than a bool so the recorded decline can name
// the offending value (the ground rules' exception-message rule) instead of only the gate.
//
// An OPEN curve out of the intersector is a partial answer, and a partial answer is refused — unlike
// the open arcs keepCrossingsOnTheWall itself produces, whose ends are rims this pipeline knows about.
func sectionClosureGap(curves []geom.Curve3) float64 {
	widest := 0.0
	for _, cv := range curves {
		lo, hi := cv.Domain()
		widest = max(widest, float64(cv.PointAt(lo).DistanceTo(cv.PointAt(hi))))
	}
	return widest
}

// declineOpenSection names a section whose curves do not close, or DeclineNone when they all do. It is
// the one place the closure scope of the closed-surface pairings is decided, so all three refuse the
// same thing for the same reason and report the same measured gap.
//
// THE CLASS SHOULD BE Weld(), AND IS NOT, and the reason is a defect elsewhere (ADR-0042; #3525,
// review round 1). By the rule the two points compared are one curve's OWN endpoints — one PointAt on
// one curve, the same computation, whose only spread is float noise — so Weld() is the class and Sew()
// (five orders looser: 1e-4·size against 1e-9·size) is for independent sources.
//
// It stays Sew() because the RESOLUTION these pairings hand in is degenerate. closedSurfaceRes reads
// geom.ResolutionForBox(faceLoopBox(sf)), and a BOUNDARY-LESS face — the bare ball and torus this
// pairing exists for — has no loops, so faceLoopBox returns the EMPTY box and the resolution collapses.
// Measured on two radius-2 spheres two apart: Weld() 1e-18, Sew() 1e-13, against the section circle's
// own endpoint noise of 4.2e-16. Weld() there is below the operands' float noise and refuses every
// exact sphere-pair section; Sew() admits it. Tightening the class is correct only after the
// resolution is derived from the model rather than from an empty box, which is its own change with its
// own measurement.
//
// A non-finite gap is OPEN, not "within tolerance": an UNBOUNDED curve (a plane∩plane line) has an
// infinite domain, its endpoint distance is not a number, and `NaN > tol` is false — so reading the
// comparison alone let the widest possible refusal through the narrowest gate.
func declineOpenSection(curves []geom.Curve3, res geom.Resolution) (float64, geom.SectionDecline) {
	gap := sectionClosureGap(curves)
	if !(gap <= res.Sew()) { // the negation is load-bearing: NaN must read as open
		return gap, geom.DeclineOpenSection
	}
	return gap, geom.DeclineNone
}

// islandSection is the seam PLUS the closure scope all three closed-surface pairings take: the pair's
// exact section, refused unless every crossing is an island on both charts. It exists because the three
// were the same seven lines three times over, and the one that drifted is how #3525's non-closing
// crossing came back as DeclineNone at two of them.
//
//	curves, why, ok := islandSection(sf.surface, rs.surface, res)
//	if !ok { recordSectionDecline(rec, why, sf, wf); return false }
func islandSection(a, b geom.Surface, res geom.Resolution) ([]geom.Curve3, sectionRefusal, bool) {
	curves, why, handled := curvedImprint(a, b, res)
	if !handled {
		return nil, refusal(why), false
	}
	if gap, open := declineOpenSection(curves, res); open != geom.DeclineNone {
		return nil, refusalf(open, "endpoint gap %g > sew %g", gap, res.Sew()), false
	}
	return curves, solved(), true
}
