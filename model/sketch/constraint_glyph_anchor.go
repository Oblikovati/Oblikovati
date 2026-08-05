// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// Where a relation's markers sit.
//
// A constraint relating TWO curves has no single home, and averaging both operands gave it one
// anyway: a perpendicular between a line at y=60 and a line at x=100 anchored at (55,37.5), a
// point on neither line, floating in the space between them. The further apart the operands, the
// further the marker drifted from both — two circles held to an equal radius put their marker
// ~90 units from either circumference. Only relations whose operands are already in contact
// (a coincidence) looked right, and only by accident: their centroid IS the contact point.
//
// So a relation is anchored once per operand, each marker ON the curve it annotates — which is
// also how Inventor draws them. Each anchor is the point of its own operand nearest the others,
// so a perpendicular between two edges of a rectangle lands on the corner they share and a
// tangency lands on the touch point, rather than at either curve's midpoint.

// operandGroup is one operand of a constraint: an entity and the constrained points of that
// entity the constraint acts on. ent is nil for a standalone point no entity claims.
type operandGroup struct {
	ent Entity
	pts []*Point
}

// center is the group's unprojected anchor — the centroid of its constrained points.
func (g operandGroup) center() math.Point2 {
	var sum math.Vector2
	for _, p := range g.pts {
		sum = sum.Add(math.V2(p.X, p.Y))
	}
	return math.P2(0, 0).TranslateBy(sum.Scale(1 / float64(len(g.pts))))
}

// constraintAnchors returns one anchor per operand the constraint acts on, each lying on that
// operand's geometry. Operands that touch yield a single anchor: their projections converge on
// the contact point and collapse, so a coincidence and a tangency each draw one marker.
//
//	ats := sk.constraintAnchors(c, sk.variableOwners()) // one point per constrained curve
func (s *Sketch) constraintAnchors(c Constraint, owners map[*math.Scalar]*Point) []math.Point2 {
	groups := s.operandGroups(involvedPoints(c, owners))
	if len(groups) == 0 {
		return nil
	}
	if len(groups) == 1 {
		return []math.Point2{groups[0].center()}
	}
	return dedupeAnchors(settledAnchors(groups, touchesCurveDOF(c, owners)))
}

// involvedPoints lists the distinct points a constraint's variables live in, in the order the
// constraint names them.
func involvedPoints(c Constraint, owners map[*math.Scalar]*Point) []*Point {
	seen := map[*Point]bool{}
	var pts []*Point
	for _, v := range c.Variables() {
		p, known := owners[v]
		if !known || seen[p] {
			continue
		}
		seen[p] = true
		pts = append(pts, p)
	}
	return pts
}

// operandGroups splits a constraint's points into one group per operand. An entity claims a group
// when the constraint moves enough of its points to identify it; the points left over — standalone
// sketch points, and the endpoints of a curve the constraint only grazes — each stand alone.
func (s *Sketch) operandGroups(involved []*Point) []operandGroup {
	claimed := map[*Point]bool{}
	var groups []operandGroup
	for _, e := range s.Entities() {
		owned := entityPointsAmong(e, involved)
		if !claimsOperand(e, owned) {
			continue
		}
		for _, p := range owned {
			claimed[p] = true
		}
		groups = append(groups, operandGroup{ent: e, pts: owned})
	}
	return append(groups, unclaimedGroups(involved, claimed)...)
}

// claimsOperand reports whether an entity is one of the constraint's operands. Two of its points
// under one constraint identify it; one does not, because a shared corner belongs to every edge
// meeting there and would otherwise make each of them an operand. A circular curve is the
// exception: the constraint reaches it through its centre alone.
func claimsOperand(e Entity, owned []*Point) bool {
	if len(owned) >= 2 {
		return true
	}
	_, circular := e.(CircularCurve)
	return len(owned) == 1 && circular
}

// entityPointsAmong returns the entity's own defining points that appear in involved.
func entityPointsAmong(e Entity, involved []*Point) []*Point {
	pd, ok := e.(pointDefined)
	if !ok {
		return nil
	}
	mine := map[*Point]bool{}
	for _, p := range pd.definingPoints() {
		mine[p] = true
	}
	return filterPoints(involved, func(p *Point) bool { return mine[p] })
}

// unclaimedGroups gathers every constrained point no entity claimed into ONE operand.
//
// One, not one each: a relation between bare points has no curve for its markers to lie on, so
// splitting it would put a marker on each point and draw a coincidence twice. Their midpoint is
// where the points end up once the solver honours the relation, and a single marker there is
// what a point-to-point relation should look like.
func unclaimedGroups(involved []*Point, claimed map[*Point]bool) []operandGroup {
	loose := filterPoints(involved, func(p *Point) bool { return !claimed[p] })
	if len(loose) == 0 {
		return nil
	}
	return []operandGroup{{pts: loose}}
}

// filterPoints returns the points satisfying keep, preserving order.
func filterPoints(pts []*Point, keep func(*Point) bool) []*Point {
	var out []*Point
	for _, p := range pts {
		if keep(p) {
			out = append(out, p)
		}
	}
	return out
}

// touchesCurveDOF reports whether a constraint moves a scalar that is not a point coordinate —
// a circle's radius, an ellipse's orientation. It separates a relation about a curve (tangency,
// equal radius) from one about its centre alone (concentricity), which decides whether the
// marker belongs on the circumference or at the centre.
func touchesCurveDOF(c Constraint, owners map[*math.Scalar]*Point) bool {
	for _, v := range c.Variables() {
		if _, isPoint := owners[v]; !isPoint {
			return true
		}
	}
	return false
}

// settledAnchors pulls every operand's anchor onto its own geometry, toward the other operands.
//
// One sweep is not a fixed point, so it sweeps twice: a tangent line's anchor reaches the touch
// point at once (the nearest point of the line to the circle's centre IS the touch point), but the
// circle's aims at the line's MIDPOINT and lands beside it. A second sweep, now aiming at settled
// neighbours, puts both on the contact point, where they collapse into the single marker a
// tangency should be.
func settledAnchors(groups []operandGroup, curveDOF bool) []math.Point2 {
	ats := make([]math.Point2, len(groups))
	for i, g := range groups {
		ats[i] = g.center()
	}
	for pass := 0; pass < settlePasses; pass++ {
		settleOnce(groups, ats, curveDOF)
	}
	return ats
}

// settlePasses is how many times the anchors are swept. Two is what a curve-to-curve contact
// needs; a third changes nothing, because by then every anchor is already the nearest point of
// its operand to a neighbour that has itself stopped moving.
const settlePasses = 2

// settleOnce moves each anchor to the point of its own operand nearest the others.
//
// It updates ats IN PLACE, so an operand settled earlier in the sweep is aimed at by the ones
// after it. Reading a whole sweep from the previous one instead leaves a tangency's two markers
// four units apart — the circle chases the line's midpoint while the line has already moved.
func settleOnce(groups []operandGroup, ats []math.Point2, curveDOF bool) {
	for i, g := range groups {
		ats[i] = projectedAnchor(g, centroidExcept(ats, i), curveDOF)
	}
}

// projectedAnchor is the point of one operand's geometry nearest toward, or the operand's own
// centroid when it has no curve to lie on.
func projectedAnchor(g operandGroup, toward math.Point2, curveDOF bool) math.Point2 {
	if g.ent == nil || isCentreRelation(g, curveDOF) {
		return g.center()
	}
	if at, ok := ClosestPointOnEntity(g.ent, toward); ok {
		return at
	}
	return g.center()
}

// isCentreRelation reports whether the constraint holds a circular curve by its centre rather
// than by its curve — concentricity, or a coincidence pinning the centre. Such a marker belongs
// at the centre; pushing it out to the circumference would put it where nothing is constrained.
func isCentreRelation(g operandGroup, curveDOF bool) bool {
	c, circular := g.ent.(CircularCurve)
	return circular && !curveDOF && len(g.pts) == 1 && g.pts[0] == c.CenterPoint()
}

// centroidExcept averages every anchor but the i'th — what operand i aims at.
func centroidExcept(ats []math.Point2, i int) math.Point2 {
	var sum math.Vector2
	for j, at := range ats {
		if j != i {
			sum = sum.Add(math.V2(at.X, at.Y))
		}
	}
	return math.P2(0, 0).TranslateBy(sum.Scale(1 / float64(len(ats)-1)))
}

// dedupeAnchors collapses anchors that settled on the same place, so a relation whose operands
// touch draws one marker rather than two stacked on the contact point.
func dedupeAnchors(ats []math.Point2) []math.Point2 {
	var out []math.Point2
	for _, at := range ats {
		if !containsAnchor(out, at) {
			out = append(out, at)
		}
	}
	return out
}

// containsAnchor reports whether at is already present, within a tolerance relative to how far
// from the origin the sketch is working — an absolute epsilon would merge nothing on a sketch
// laid out in microns and everything on one laid out in kilometres.
func containsAnchor(ats []math.Point2, at math.Point2) bool {
	for _, other := range ats {
		if other.DistanceTo(at) <= anchorEpsilon*(1+magnitude(at)) {
			return true
		}
	}
	return false
}

// anchorEpsilon is the relative distance below which two anchors are the same place. It is far
// above the solver's convergence (~1e-10) so a satisfied tangency collapses, and far below any
// distance a person could see, so two genuinely separate operands never merge.
const anchorEpsilon = 1e-7

// magnitude is a point's distance from the origin, the scale its tolerance is relative to.
func magnitude(p math.Point2) float64 { return float64(p.DistanceTo(math.P2(0, 0))) }
