// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// Mixed-sense trihedral corner patch on a CURVED host (corner-blend-weld Slice-1a, M8). A trihedral
// vertex where 1 CONVEX + 2 CONCAVE rolling-ball fillets meet is NOT a sphere corner — no single ball
// is tangent to the shared curved host at both R−r (the convex arm) and R+r (a concave arm)
// simultaneously (proved for M8: 20 vs 30 from the boss axis). OCCT builds it as an analytic 2r-TORUS
// corner (axis = the convex fillet axis, major R=2r, minor r), the exact curved-host lift of the planar
// box-corner mechanism (fillet_corner_torus.go, K9/M2/L6). This file derives that torus + its four
// bounding weld arcs from the incident arms; the derivation is DRAWEXE-verified to the digit
// (geometry-math-advisor, slice1a-report.md §"M8 mixed-sense corner-patch"):
//
//	M8: C=(55,30.635,105), axis vertical, major 10 minor 5; ρ̂=(−0.25,−0.968); F_A=(52.5,20.955,105).
//
// The two deltas vs the planar mechanism: (i) the common-perpendicular reference for the centre C is the
// PLANAR concave band (a real line spine) — a curved concave band has a circle spine, no line; (ii) the
// curved concave band's spine foot is the nearest point on its ball-centre CIRCLE (F_A = O_boss+(R+r)ρ̂)
// and its wall contact normal is the radial ρ̂, not a constant plane normal.

// curvedMixedCornerTol is the model-relative slack (× the host radius, ADR-0042) for the R=2r pivot
// gate: |dist(C, cove circle) − 2r| below this admits the analytic torus corner; above it the corner is
// a non-2r (BSpline) mixed corner (N4/O1/H7) and this path DECLINES so it is never mis-modelled.
const curvedMixedCornerTol = 1e-6

// curvedMixedCorner is the fully solved M8-class corner: the analytic 2r-torus patch, its centre C, the
// two radii, and the four bounding weld arcs — cove (shared with the R+r concave torus arm), inner
// equator (shared with the convex cylinder arm), bandB (shared with the planar cylinder arm), and top
// (radius 2r on the shared host plane, shared with its retrim). Every arc is built here ONCE so both the
// patch and the neighbour it welds to read the byte-identical curve (assembleBody welds by shared points).
type curvedMixedCorner struct {
	torus    geom.Torus
	center   math.Point3 // C — on the convex arm axis, at the concave-spine height
	r, major float64     // r (minor) and R=2r (major)
	arcCove  geom.Arc3d  // (a) tube arc radius r about F_A — welds to the R+r cove torus arm
	arcInner geom.Arc3d  // (b) inner-equator arc radius R−r=r about the axis — welds to the convex cyl arm
	arcBandB geom.Arc3d  // (c) tube arc radius r about F_B — welds to the planar concave cyl arm
	arcTop   geom.Arc3d  // (d) top-contact arc radius R=2r about the axis on the host plane — welds to its retrim
}

// curvedMixedArms is the three classified arms of a mixed-sense curved-host corner: the convex pivot
// (a geom.Cylinder arm), the curved concave band (a geom.Torus R+r cove arm) with its host cylinder, and
// the planar concave band (a geom.Cylinder arm whose two hosts are planes — the line-spine reference).
type curvedMixedArms struct {
	convex geom.Cylinder // the pivot: L_cvx = the ball-centre axis
	cove   geom.Torus    // the R+r concave torus arm (major = R_host+r)
	boss   geom.Cylinder // the curved host wall the cove arm rolls outside of (radius R_host)
	planar geom.Cylinder // band B: the planar concave arm's ball-centre line (spine)
	top    geom.Plane    // the host plane both concave bands contact (the top-contact plane)
	topOut math.Vector3  // top material-outward unit normal
}

// solveCurvedMixedCorner derives the 2r-torus corner patch + its four weld arcs from the classified
// arms, or ok=false when the corner is not this analytic-torus class (a degenerate pivot radius, a
// centre off the convex axis, or a cove circle whose foot is not at 2r — the BSpline mixed corners
// N4/O1/H7). It is the curved-host sibling of solveMixedTorusCorner. Example:
//
//	c, ok := solveCurvedMixedCorner(arms, r, res); if ok { /* c.torus is the M8 corner patch */ }
func solveCurvedMixedCorner(arms curvedMixedArms, r float64, res tol.Resolution) (curvedMixedCorner, bool) {
	axis := arms.convex.AxisDir.AsVector()
	spineB := arms.planar
	c, footB := closestPointsBetweenLines(arms.convex.Origin, axis, spineB.Origin, spineB.AxisDir.AsVector())
	major := c.DistanceTo(footB)
	if stdmath.Abs(major-2*r) > curvedMixedCornerTol*arms.boss.Radius {
		return curvedMixedCorner{}, false // not the 2r rolling pivot — a BSpline mixed corner (N4/O1/H7)
	}
	rho, ok := cornerRadialDir(c, arms.boss, res)
	if !ok {
		return curvedMixedCorner{}, false // C on the boss axis (spindle) — ρ̂ undefined
	}
	fA := c.TranslateBy(rho.Scale(major)) // nearest point on the cove ball-centre circle (F_A = C + R·ρ̂)
	if !coveFootOnCircle(fA, arms.cove, res) {
		return curvedMixedCorner{}, false // F_A not on the R+r cove circle — inconsistent curved corner
	}
	return assembleCurvedMixedCorner(c, footB, fA, rho, arms, r, major)
}

// cornerRadialDir is ρ̂: the unit radial from the boss axis toward the corner centre C (the direction the
// convex-arm axis and C both sit along, R_host−r out from the boss). ok=false when C is on the axis
// (dist ≤ the model weld band, a spindle already screened upstream) so ρ̂ is undefined.
func cornerRadialDir(c math.Point3, boss geom.Cylinder, res tol.Resolution) (math.Vector3, bool) {
	foot := footOnLine(c, boss.Origin, boss.AxisDir.AsVector())
	rho, err := math.UnitVector3FromVector(foot.VectorTo(c))
	if err != nil || foot.DistanceTo(c) < armSpindleBand*res.Weld() {
		return math.Vector3{}, false
	}
	return rho.AsVector(), true
}

// coveFootOnCircle re-verifies F_A lies on the cove arm's ball-centre circle (centre = cove torus
// centre, radius = its major R+r, in the plane ⊥ the axis) within the model-relative tolerance — the
// curved-host validity gate the planar mechanism gets for free from its line foot (derivation §pitfalls).
func coveFootOnCircle(fA math.Point3, cove geom.Torus, res tol.Resolution) bool {
	axis := cove.AxisDir.AsVector()
	d := cove.Center.VectorTo(fA)
	if stdmath.Abs(d.Dot(axis)) > res.Weld()*cove.MajorRadius {
		return false // F_A off the cove circle plane
	}
	inPlane := d.Sub(axis.Scale(d.Dot(axis))).Length()
	return stdmath.Abs(float64(inPlane)-cove.MajorRadius) < res.Weld()*cove.MajorRadius
}

// assembleCurvedMixedCorner builds the torus and the four weld arcs once C, the two feet and ρ̂ are
// solved. σ̂ = unit(F_B−C) is the second concave direction; the u-span (angle ρ̂→σ̂) need not be 90°
// (M8 is 104.5°), so every arc goes through its on-circle bisector midpoint (arcMidOnCircle / the tube
// bisector) rather than a hard-coded quarter.
func assembleCurvedMixedCorner(c, footB, fA math.Point3, rho math.Vector3, arms curvedMixedArms, r, major float64) (curvedMixedCorner, bool) {
	axis := arms.convex.AxisDir.AsVector()
	sigma, err := math.UnitVector3FromVector(c.VectorTo(footB))
	if err != nil {
		return curvedMixedCorner{}, false
	}
	tor, err := geom.NewTorus(c, axis, major, r)
	if err != nil {
		return curvedMixedCorner{}, false
	}
	nTop := arms.topOut
	qInnerA := c.TranslateBy(rho.Scale(r))                                    // C + r·ρ̂ — on the convex cylinder AND the boss wall
	qInnerB := c.TranslateBy(sigma.AsVector().Scale(r))                       // C + r·σ̂
	qBotA := fA.TranslateBy(nTop.Scale(-r))                                   // F_A − r·n̂_top — on the host plane
	qBotB := footB.TranslateBy(nTop.Scale(-r))                                // F_B − r·n̂_top
	cBot := c.TranslateBy(nTop.Scale(-arms.top.Origin.VectorTo(c).Dot(nTop))) // C dropped to the host plane (top-contact circle centre)
	arcs, ok := curvedMixedArcs(c, cBot, fA, footB, qInnerA, qInnerB, qBotA, qBotB, r, major)
	if !ok {
		return curvedMixedCorner{}, false
	}
	return curvedMixedCorner{torus: tor, center: c, r: r, major: major,
		arcCove: arcs[0], arcInner: arcs[1], arcBandB: arcs[2], arcTop: arcs[3]}, true
}

// curvedMixedArcs builds the four bounding quarter-arcs (a,b,c,d) through their on-circle bisector
// midpoints. (a)/(c) are tube arcs radius r about the spine feet; (b) is the inner equator radius R−r=r
// about the axis (centre C); (d) is the top contact radius R=2r about the axis on the host plane
// (centre cBot). Any arc constructor decline (collinear triple) returns ok=false — do-no-harm.
func curvedMixedArcs(c, cBot, fA, footB, qInnerA, qInnerB, qBotA, qBotB math.Point3, r, major float64) ([4]geom.Arc3d, bool) {
	specs := []struct{ from, mid, to math.Point3 }{
		{qInnerA, tubeArcMid(fA, qInnerA, qBotA, r), qBotA},              // (a) cove tube
		{qInnerA, arcMidOnCircle(c, qInnerA, qInnerB, major-r), qInnerB}, // (b) inner equator
		{qInnerB, tubeArcMid(footB, qInnerB, qBotB, r), qBotB},           // (c) band-B tube
		{qBotA, arcMidOnCircle(cBot, qBotA, qBotB, major), qBotB},        // (d) top contact
	}
	var out [4]geom.Arc3d
	for i, s := range specs {
		arc, err := geom.Arc3dByThreePoints(s.from, s.mid, s.to)
		if err != nil {
			return [4]geom.Arc3d{}, false
		}
		out[i] = arc
	}
	return out, true
}

// tubeArcMid is the on-circle midpoint of the 90° tube arc from→to on the minor circle (centre = the
// tube centre F, radius r): F pushed r along the bisector of the two contact radial directions. The
// sibling of arcMidOnCircle for a tube (minor) arc rather than a major-circle arc.
func tubeArcMid(f, from, to math.Point3, r float64) math.Point3 {
	bis := f.VectorTo(from).Add(f.VectorTo(to))
	return f.TranslateBy(probe.Unit(bis).Scale(r))
}
