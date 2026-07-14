// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// extractRunout tiles an S1-shaped DOUBLE-interference runout hole into THREE valence-4 coons4
// RailLoops — the topology OCCT's BRepBlend produces here, measured from the DRAWEXE oracle
// (architecture note s1-runout-topology.md). The hole is a single hexagon whose six sides
// alternate {plane-A curve, fillet ¼-circle, feature-B arc}; it cannot close as one coons4, so we
// split it with two internal seams into a CENTRAL patch (bridging the two feature walls) and two
// flanking patches (RIGHT/LEFT) that each reach out to a fillet cut.
//
// Continuity is load-bearing (coons4Provider only builds a G1 ribbon for a Cont==G1 side with a
// non-nil Adjacent): the fillet ¼-circles are G1 to the fillet cylinder and the plane-A curves are
// G1 to host plane A (real analytic hosts available now); the feature-arc sides and the two
// internal seams are G0 with a nil Adjacent (a feature-wall G1 ribbon would invert the patch —
// advisor pitfall 4 — and a fill-to-fill G1 seam is a coupled multi-patch solve deferred to M3).
//
// It returns ok=false for anything that is NOT this double-interference case (a single or absent
// imprint, a footprint whose conic centre/radius can't be read, or a seam abscissa that falls
// outside a feature circle), and honest-rejects a mis-tiled loop via the flat-lune guard. This
// task is UNWIRED — Task 10 connects it — so its only obligation is to emit the three loops.
func extractRunout(region runoutRegion, ef edgeFillet, res Resolution) ([]RailLoop, bool) {
	loops, _, ok := extractRunoutTiled(region, ef, res)
	return loops, ok
}

// extractRunoutTiled is extractRunout with the resolved runoutTiling also returned, so the closure
// path (buildRunoutHostsAndWalls) can reconstruct the host planes and split the boss walls against
// the SAME corners the loops were tiled from (bit-identical shared corners ⇒ watertight, Task 10b).
func extractRunoutTiled(region runoutRegion, ef edgeFillet, res Resolution) ([]RailLoop, runoutTiling, bool) {
	tl, ok := resolveTiling(region, ef)
	if !ok {
		return nil, runoutTiling{}, false
	}
	loops := make([]RailLoop, 0, 3)
	for _, build := range []func() (RailLoop, bool){tl.centralLoop, tl.rightLoop, tl.leftLoop} {
		loop, ok := build()
		if !ok || !loopWellFormed(loop, res.Weld()) {
			return nil, runoutTiling{}, false
		}
		loops = append(loops, loop)
	}
	return loops, tl, true
}

// runoutTiling is extractRunout's resolved geometry: the fillet cylinder, the two host planes and
// their imprints (feature A = the narrower boss on plane A, feature B = the wider boss on plane B
// that cuts the fillet), the two fillet-cut spine stations, and the eight distinct hexagon/seam
// corners. Every corner is computed ONCE here so a shared corner is bit-identical across loops
// (watertight union), and the loop builders below only assemble curves between them.
type runoutTiling struct {
	cyl            geom.Cylinder
	planeA, planeB geom.Plane
	aImp, bImp     runoutImprint
	cutR, cutL     float64     // fillet-cut spine stations (+x, -x)
	faR, fbR       math.Point3 // fillet∩planeA / ∩planeB at +x
	fbL, faL       math.Point3 // ∩planeB / ∩planeA at -x
	sbR, sbL       math.Point3 // seam bottoms (on feature-A footprint, plane A)
	stR, stL       math.Point3 // seam tops (on feature-B footprint, plane B)
	caL, caR       math.Point3 // feature-A footprint crossings of the receded band (plane A, x-ordered)
}

// resolveTiling classifies the two features, derives the free seam abscissa, and computes the
// eight corners. The seam sits halfway between a fillet cut and the hole's symmetry plane (the
// midpoint of feature-B's cut interval) — a derived, model-relative placement, NOT the oracle's
// hard-coded x=3.38 (the fill is area-validated, so the interior seam is free; s1-runout-topology.md).
func resolveTiling(region runoutRegion, ef edgeFillet) (runoutTiling, bool) {
	aImp, bImp, aCut, bCut, ok := classifyFeatures(region, ef.cyl)
	if !ok {
		return runoutTiling{}, false
	}
	lo, hi := spineInterval(bCut, ef.cyl)
	center, half := (lo+hi)/2, (hi-lo)/2
	t := runoutTiling{
		cyl: ef.cyl, planeA: aImp.plane, planeB: bImp.plane,
		aImp: aImp, bImp: bImp, cutR: hi, cutL: lo,
	}
	if !t.resolveCorners(aCut, bCut, center+half/2, center-half/2) {
		return runoutTiling{}, false
	}
	t.caL, t.caR = xOrdered(aCut.pMinus, aCut.pPlus)
	return t, true
}

// xOrdered returns (p,q) sorted so the first has the smaller X — the left/right convention the host-A
// notch + flat-fill triangles use for the feature-A crossings (caL left, caR right).
func xOrdered(a, b math.Point3) (lo, hi math.Point3) {
	if a.X <= b.X {
		return a, b
	}
	return b, a
}

// resolveCorners fills the four fillet-cut corners (exact cylinder∩plane contacts) and the four
// seam corners (on the feature footprints at the two seam spine stations). ok=false when a seam
// station lands outside a feature circle (seamPointOnFeature declines).
func (t *runoutTiling) resolveCorners(aCut, bCut imprintCut, seamR, seamL float64) bool {
	t.faR = filletContact(t.cyl, t.planeA, t.cutR)
	t.fbR = filletContact(t.cyl, t.planeB, t.cutR)
	t.fbL = filletContact(t.cyl, t.planeB, t.cutL)
	t.faL = filletContact(t.cyl, t.planeA, t.cutL)
	var ok [4]bool
	t.sbR, ok[0] = seamPointOnFeature(t.aImp, aCut, t.cyl, seamR)
	t.sbL, ok[1] = seamPointOnFeature(t.aImp, aCut, t.cyl, seamL)
	t.stR, ok[2] = seamPointOnFeature(t.bImp, bCut, t.cyl, seamR)
	t.stL, ok[3] = seamPointOnFeature(t.bImp, bCut, t.cyl, seamL)
	return ok[0] && ok[1] && ok[2] && ok[3]
}

// classifyFeatures labels the region's two imprints: the one whose footprint cuts the WIDER span
// of the fillet spine is feature B (it interrupts the quarter-cylinder and owns the top arc); the
// narrower is feature A (its footprint owns the central bottom arc and its host carries the two
// plane-A boundary curves). It declines a region that is not exactly the two-imprint case.
func classifyFeatures(region runoutRegion, cyl geom.Cylinder) (aImp, bImp runoutImprint, aCut, bCut imprintCut, ok bool) {
	if len(region.imprints) != 2 || len(region.cuts) != 2 {
		return runoutImprint{}, runoutImprint{}, imprintCut{}, imprintCut{}, false
	}
	if cutSpan(region.cuts[0], cyl) >= cutSpan(region.cuts[1], cyl) {
		return region.imprints[1], region.imprints[0], region.cuts[1], region.cuts[0], true
	}
	return region.imprints[0], region.imprints[1], region.cuts[0], region.cuts[1], true
}

// cutSpan is a cut's axial extent along the fillet spine — the ruler classifyFeatures ranks the
// two bosses by (the wider one is the fillet-cutting feature B).
func cutSpan(cut imprintCut, cyl geom.Cylinder) float64 {
	lo, hi := spineInterval(cut, cyl)
	return hi - lo
}

// centralLoop bridges the two feature walls: featureA arc (bottom) → seam_right → featureB arc
// (top) → seam_left. All four sides are G0 with a nil Adjacent — a pure-position Coons patch whose
// admissibility is the plain-fill NoFold, not a ribbon.
func (t runoutTiling) centralLoop() (RailLoop, bool) {
	bottom, ok0 := featureSubArc(t.aImp, t.sbL, t.sbR)
	top, ok1 := featureSubArc(t.bImp, t.stR, t.stL)
	if !ok0 || !ok1 {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: bottom, Cont: G0},
		{Curve: internalSeam(t.sbR, t.stR), Cont: G0},
		{Curve: top, Cont: G0},
		{Curve: internalSeam(t.stL, t.sbL), Cont: G0},
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}

// rightLoop reaches from the central patch out to the +x fillet cut: plane-A curve (G1→plane A) →
// fillet ¼-circle (G1→fillet cylinder) → feature-B arc portion (G0) → seam_right (G0, the same
// endpoints centralLoop's seam_right carries, traversed the other way so both loops stay closed).
func (t runoutTiling) rightLoop() (RailLoop, bool) {
	arc, ok0 := armSectionArc(t.cyl, t.planeA, t.planeB, t.cutR)
	fb, ok1 := featureSubArc(t.bImp, t.fbR, t.stR)
	if !ok0 || !ok1 {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: planeARunoutCurve(t.sbR, t.faR), Adjacent: t.planeA, Cont: G1},
		{Curve: arc, Adjacent: t.cyl, Cont: G1},
		{Curve: fb, Cont: G0},
		{Curve: internalSeam(t.stR, t.sbR), Cont: G0},
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}

// leftLoop is rightLoop mirrored to the -x fillet cut: plane-A curve → seam_left → feature-B arc
// portion → fillet ¼-circle. The cycle is ordered so consecutive sides share endpoints.
func (t runoutTiling) leftLoop() (RailLoop, bool) {
	arc, ok0 := armSectionArc(t.cyl, t.planeB, t.planeA, t.cutL)
	fb, ok1 := featureSubArc(t.bImp, t.stL, t.fbL)
	if !ok0 || !ok1 {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: planeARunoutCurve(t.faL, t.sbL), Adjacent: t.planeA, Cont: G1},
		{Curve: internalSeam(t.sbL, t.stL), Cont: G0},
		{Curve: fb, Cont: G0},
		{Curve: arc, Adjacent: t.cyl, Cont: G1},
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}

// armSectionArc is the fillet cross-section quarter-circle at spine station `spine`: the arc on the
// fillet cylinder from its contact with plane `first` to its contact with plane `second`, through
// the 45° bisector point. The contacts are the feet of the perpendiculars from the cross-section's
// axis point to each host plane (each at distance = Radius, since the fillet is tangent to both) —
// the same Arc3dByThreePoints construction fillet_faces.go uses at the corner arcs, re-stationed.
// Swapping first/second yields the arc in the reverse orientation (used by the two flank loops).
func armSectionArc(cyl geom.Cylinder, first, second geom.Plane, spine float64) (geom.Arc3d, bool) {
	axisPt := cyl.Origin.TranslateBy(cyl.AxisDir.AsVector().Scale(spine))
	c1 := projectOntoPlane(axisPt, first)
	c2 := projectOntoPlane(axisPt, second)
	bis := axisPt.VectorTo(c1).Add(axisPt.VectorTo(c2))
	l := bis.Length()
	if l < arcBisectorTiny*cyl.Radius {
		return geom.Arc3d{}, false // contacts near-antipodal: the bisector midpoint is ill-defined
	}
	mid := axisPt.TranslateBy(bis.Scale(cyl.Radius / l))
	arc, err := geom.Arc3dByThreePoints(c1, mid, c2)
	return arc, err == nil
}

// featureSubArc is the sub-arc of an imprint's footprint circle from `from` to `to` (both already
// on the circle), built through the circle point on the angular bisector of the two — exact for the
// sub-180° arcs this hexagon uses (feature footprints reconstructed by footprintConic, no fitting).
func featureSubArc(im runoutImprint, from, to math.Point3) (geom.Arc3d, bool) {
	c, r, ok := footprintConic(im.footprintEdge)
	if !ok {
		return geom.Arc3d{}, false
	}
	bis := c.VectorTo(from).Add(c.VectorTo(to))
	l := bis.Length()
	if l < arcBisectorTiny*r {
		return geom.Arc3d{}, false // endpoints near-antipodal on the footprint circle
	}
	mid := c.TranslateBy(bis.Scale(r / l))
	arc, err := geom.Arc3dByThreePoints(from, mid, to)
	return arc, err == nil
}

// featureMajorArc is the MAJOR (>180°) sub-arc of an imprint's footprint circle from `from` to `to`,
// built through the point ANTIPODAL to featureSubArc's bisector midpoint — the piece that wraps the far
// side of the boss, where featureSubArc's minor arc would cut straight across it.
func featureMajorArc(im runoutImprint, from, to math.Point3) (geom.Arc3d, bool) {
	c, r, ok := footprintConic(im.footprintEdge)
	if !ok {
		return geom.Arc3d{}, false
	}
	bis := c.VectorTo(from).Add(c.VectorTo(to))
	l := bis.Length()
	if l < arcBisectorTiny*r {
		return geom.Arc3d{}, false // endpoints near-antipodal: the major/minor split is ill-defined
	}
	mid := c.TranslateBy(bis.Scale(-r / l)) // antipodal to featureSubArc's midpoint → the major side
	arc, err := geom.Arc3dByThreePoints(from, mid, to)
	return arc, err == nil
}

// hostSideArc is the footprint sub-arc from `from` to `to` that stays on the imprint's HOST side (away
// from the fillet band): the minor (featureSubArc) arc when ITS midpoint is host-side, else the major
// one. It reuses onHostSide's signed band test (fillet_runout_imprint.go) so the host-plane notch and
// the split boss wall trace the SAME boundary the boss footprint had before the fillet.
func hostSideArc(im runoutImprint, from, to math.Point3) (geom.Arc3d, bool) {
	minor, ok := featureSubArc(im, from, to)
	if ok && onHostSide(im, minor.PointAt(0.5)) {
		return minor, true
	}
	return featureMajorArc(im, from, to)
}

// planeARunoutCurve is the fill's boundary on host plane A between a seam-bottom and a fillet-cut
// corner. Both endpoints lie in plane A, so the fill can be G1-tangent to the (flat) host along it;
// its exact shape is a free, area-validated boundary (like the seam), emitted here as the straight
// in-plane segment. A curved OCCT-faithful trace is an M3/Task-10 area-oracle refinement.
func planeARunoutCurve(seamBottom, filletCut math.Point3) geom.LineSegment {
	return geom.NewLineSegment(seamBottom, filletCut)
}

// internalSeam is the free-placement decomposition curve shared by a flank patch and the central
// patch. It is a straight segment; Adjacent stays nil and Cont G0 (a fill-to-fill G1 seam is a
// coupled solve deferred to M3). Built from the SAME two corner values on both loops so the union
// is watertight — the flank stores it reversed to keep its own cycle closed.
func internalSeam(from, to math.Point3) geom.LineSegment {
	return geom.NewLineSegment(from, to)
}

// filletContact is the point where the fillet cylinder touches `plane` at spine station `spine`:
// the projection of the cross-section's axis point onto the plane (the tangency foot, at radius).
func filletContact(cyl geom.Cylinder, plane geom.Plane, spine float64) math.Point3 {
	axisPt := cyl.Origin.TranslateBy(cyl.AxisDir.AsVector().Scale(spine))
	return projectOntoPlane(axisPt, plane)
}

// seamPointOnFeature is the point on an imprint's footprint circle at spine station `spine`, on the
// edgeward side (toward the fillet band). It solves the circle at abscissa a = spine − centre-spine
// as centre + a·axis + √(r²−a²)·edgeward; ok=false when |a| ≥ r (the seam station falls outside the
// feature circle, i.e. a mis-derived seam abscissa) so extractRunout honest-rejects.
func seamPointOnFeature(im runoutImprint, cut imprintCut, cyl geom.Cylinder, spine float64) (math.Point3, bool) {
	c, r, ok := footprintConic(im.footprintEdge)
	if !ok {
		return math.Point3{}, false
	}
	a := spine - spineParam(c, cyl)
	if a*a >= r*r {
		return math.Point3{}, false
	}
	w, ok := edgewardNormal(im, cyl, c, cut)
	if !ok {
		return math.Point3{}, false
	}
	h := stdmath.Sqrt(r*r - a*a)
	return c.TranslateBy(cyl.AxisDir.AsVector().Scale(a)).TranslateBy(w.Scale(h)), true
}

// edgewardNormal is the unit in-plane direction on an imprint's host, perpendicular to the fillet
// spine, pointing toward the fillet band (the edge). It is host-normal × spine-axis, sign-oriented
// to agree with the direction from the footprint centre to a band crossing (cut.pMinus, which is on
// the band). ok=false if the spine lies in the host normal (degenerate, no in-plane perpendicular).
func edgewardNormal(im runoutImprint, cyl geom.Cylinder, center math.Point3, cut imprintCut) (math.Vector3, bool) {
	u, err := math.UnitVector3FromVector(im.plane.Normal().Cross(cyl.AxisDir.AsVector()))
	if err != nil {
		return math.Vector3{}, false
	}
	w := u.AsVector()
	if center.VectorTo(cut.pMinus).Dot(w) < 0 {
		w = w.Negate()
	}
	return w, true
}

// projectOntoPlane returns the orthogonal projection of p onto plane (its metric-nearest point).
func projectOntoPlane(p math.Point3, plane geom.Plane) math.Point3 {
	n := plane.Normal()
	d := plane.Origin.VectorTo(p).Dot(n) / n.Dot(n)
	return p.TranslateBy(n.Scale(-d))
}

// loopWellFormed gates a tiled loop: valence 4, closed within the model-relative weld, and every
// corner non-degenerate (the flat-lune guard). Any failure means a mis-tile ⇒ honest-reject.
func loopWellFormed(l RailLoop, weld float64) bool {
	if l.Valence() != 4 || !l.Closed(weld) {
		return false
	}
	return noFlatLuneCorner(l, weld)
}

// noFlatLuneCorner is the flat-lune guard (advisor pitfall 1): at every corner the two consecutive
// sides must enclose a non-degenerate triangle (area ≥ weld²). A corner below that threshold means
// two consecutive sides are collinear on a shared carrier plane — the degeneracy that collapsed the
// original single-quad guess to a lune. weld is model-relative (ADR-0042), never a bare epsilon.
func noFlatLuneCorner(l RailLoop, weld float64) bool {
	n := len(l.Sides)
	for i := range l.Sides {
		prev := curveStart(l.Sides[(i+n-1)%n].Curve)
		cur := curveStart(l.Sides[i].Curve)
		next := curveStart(l.Sides[(i+1)%n].Curve)
		if triangleArea(prev, cur, next) < weld*weld {
			return false
		}
	}
	return true
}

// triangleArea is the area of triangle (a,b,c): ½·|(a−b)×(c−b)|.
func triangleArea(a, b, c math.Point3) float64 {
	return 0.5 * b.VectorTo(a).Cross(b.VectorTo(c)).Length()
}

// arcBisectorTiny is the dimensionless floor (scaled by circle radius at each use, ADR-0042) below
// which an arc's two boundary points are treated as near-antipodal and its bisector midpoint as
// ill-defined — the arc-construction sibling of ribbonDirTiny's direction-collapse floor.
const arcBisectorTiny = 1e-9
