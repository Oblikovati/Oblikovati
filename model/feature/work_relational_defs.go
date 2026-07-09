// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"oblikovati.org/math"
)

// Relational datum-axis and datum-point constructors built purely on work-feature references
// (points / lines / planes) — the reference-model half of Inventor's WorkAxes / WorkPoints
// constructor sets (#1840, #1842). The face/edge-topology constructors (revolved-face,
// analytic-edge, torus/sphere centre, curve∩surface) need ADR-0040 geometric selectors and are a
// follow-up. Each captures its references so it round-trips and re-derives on recompute, and
// reports an error (→ healthy=false) on degenerate input rather than producing garbage.

// --- work axes -------------------------------------------------------------

// pointAndPlaneAxisDef is the axis through a point, normal to a plane (Inventor AddByPointAndPlane).
type pointAndPlaneAxisDef struct {
	point WorkRef
	plane WorkRef
}

func (d pointAndPlaneAxisDef) kindName() string { return "point-and-plane" }
func (d pointAndPlaneAxisDef) refs() []WorkRef  { return []WorkRef{d.point, d.plane} }
func (d pointAndPlaneAxisDef) eval(r workResolver) (math.Point3, math.UnitVector3, error) {
	pt, err := r.point(d.point)
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, err
	}
	pl, err := r.plane(d.plane)
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, err
	}
	return pt, pl.Normal(), nil
}

// lineAndPointAxisDef is the axis through a point, parallel to a line (Inventor AddByLineAndPoint).
type lineAndPointAxisDef struct {
	line  WorkRef
	point WorkRef
}

func (d lineAndPointAxisDef) kindName() string { return "line-and-point" }
func (d lineAndPointAxisDef) refs() []WorkRef  { return []WorkRef{d.line, d.point} }
func (d lineAndPointAxisDef) eval(r workResolver) (math.Point3, math.UnitVector3, error) {
	line, err := r.axis(d.line)
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, err
	}
	pt, err := r.point(d.point)
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, err
	}
	return pt, line.dir, nil
}

// lineAndPlaneAxisDef is the input line projected onto a plane along the plane normal (Inventor
// AddByLineAndPlane). Degenerate when the line is perpendicular to the plane (its projection is a
// point, not a line).
type lineAndPlaneAxisDef struct {
	line  WorkRef
	plane WorkRef
}

func (d lineAndPlaneAxisDef) kindName() string { return "line-and-plane" }
func (d lineAndPlaneAxisDef) refs() []WorkRef  { return []WorkRef{d.line, d.plane} }
func (d lineAndPlaneAxisDef) eval(r workResolver) (math.Point3, math.UnitVector3, error) {
	line, err := r.axis(d.line)
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, err
	}
	pl, err := r.plane(d.plane)
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, err
	}
	n := pl.Normal().AsVector()
	ld := line.dir.AsVector()
	dir, err := math.UnitVector3FromVector(ld.Sub(n.Scale(ld.Dot(n)))) // drop the out-of-plane component
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, errors.New("the line is perpendicular to the plane, so its projection is a point")
	}
	origin := projectPointOntoPlane(line.origin, pl.Origin(), n)
	return origin, dir, nil
}

// projectPointOntoPlane drops p onto the plane through planeOrigin with unit normal n.
func projectPointOntoPlane(p, planeOrigin math.Point3, n math.Vector3) math.Point3 {
	return p.TranslateBy(n.Scale(p.VectorTo(planeOrigin).Dot(n)))
}

// AddByPointAndPlane creates the axis through a point, normal to a plane (#1840).
func (c *WorkAxes) AddByPointAndPlane(point, plane WorkRef) *WorkAxis {
	return c.addUser(pointAndPlaneAxisDef{point: point, plane: plane})
}

// AddByLineAndPoint creates the axis through a point, parallel to a line (#1840).
func (c *WorkAxes) AddByLineAndPoint(line, point WorkRef) *WorkAxis {
	return c.addUser(lineAndPointAxisDef{line: line, point: point})
}

// AddByLineAndPlane creates the axis of the input line projected onto a plane (#1840).
func (c *WorkAxes) AddByLineAndPlane(line, plane WorkRef) *WorkAxis {
	return c.addUser(lineAndPlaneAxisDef{line: line, plane: plane})
}

// --- work points -----------------------------------------------------------

// pointRefPointDef is a datum point coincident with a referenced point/vertex (Inventor AddByPoint).
type pointRefPointDef struct{ point WorkRef }

func (d pointRefPointDef) kindName() string                         { return "point" }
func (d pointRefPointDef) refs() []WorkRef                          { return []WorkRef{d.point} }
func (d pointRefPointDef) eval(r workResolver) (math.Point3, error) { return r.point(d.point) }

// twoLinesPointDef is the intersection of two lines (Inventor AddByTwoLines). Degenerate when the
// lines are parallel or skew (do not meet).
type twoLinesPointDef struct{ l1, l2 WorkRef }

func (d twoLinesPointDef) kindName() string { return "two-lines" }
func (d twoLinesPointDef) refs() []WorkRef  { return []WorkRef{d.l1, d.l2} }
func (d twoLinesPointDef) eval(r workResolver) (math.Point3, error) {
	l1, err := r.axis(d.l1)
	if err != nil {
		return math.Point3{}, err
	}
	l2, err := r.axis(d.l2)
	if err != nil {
		return math.Point3{}, err
	}
	return lineLineIntersection(l1.origin, l1.dir.AsVector(), l2.origin, l2.dir.AsVector())
}

// lineLineIntersection returns the point where two lines meet, or an error when they are parallel
// (no cross product) or skew (their closest points do not coincide).
func lineLineIntersection(o1 math.Point3, d1 math.Vector3, o2 math.Point3, d2 math.Vector3) (math.Point3, error) {
	cross := d1.Cross(d2)
	denom := cross.LengthSquared()
	if math.IsNearZero(denom, math.DefaultTolerance) {
		return math.Point3{}, errors.New("the two lines are parallel")
	}
	r := o1.VectorTo(o2)
	p1 := o1.TranslateBy(d1.Scale(r.Cross(d2).Dot(cross) / denom))
	p2 := o2.TranslateBy(d2.Scale(r.Cross(d1).Dot(cross) / denom))
	if p1.VectorTo(p2).LengthSquared() > math.DefaultTolerance {
		return math.Point3{}, errors.New("the two lines are skew and do not intersect")
	}
	return p1, nil
}

// threePlanesPointDef is the intersection point of three planes (Inventor AddByThreePlanes) — the
// line where the first two meet, intersected with the third. Degenerate when any two are parallel
// or the third is parallel to that line (no unique point).
type threePlanesPointDef struct{ p1, p2, p3 WorkRef }

func (d threePlanesPointDef) kindName() string { return "three-planes" }
func (d threePlanesPointDef) refs() []WorkRef  { return []WorkRef{d.p1, d.p2, d.p3} }
func (d threePlanesPointDef) eval(r workResolver) (math.Point3, error) {
	p1, err := r.plane(d.p1)
	if err != nil {
		return math.Point3{}, err
	}
	p2, err := r.plane(d.p2)
	if err != nil {
		return math.Point3{}, err
	}
	p3, err := r.plane(d.p3)
	if err != nil {
		return math.Point3{}, err
	}
	o, dir, err := planeIntersectionLine(p1, p2)
	if err != nil {
		return math.Point3{}, errors.New("the first two planes are parallel")
	}
	n3 := p3.Normal().AsVector()
	denom := dir.AsVector().Dot(n3)
	if math.IsNearZero(denom, math.DefaultTolerance) {
		return math.Point3{}, errors.New("the three planes do not meet at a single point")
	}
	return o.TranslateBy(dir.AsVector().Scale(o.VectorTo(p3.Origin()).Dot(n3) / denom)), nil
}

// AddByPoint creates a datum point coincident with a referenced point (#1842).
func (c *WorkPoints) AddByPoint(point WorkRef) *WorkPoint {
	return c.addUser(pointRefPointDef{point: point})
}

// AddByTwoLines creates a datum point where two lines intersect (#1842).
func (c *WorkPoints) AddByTwoLines(l1, l2 WorkRef) *WorkPoint {
	return c.addUser(twoLinesPointDef{l1: l1, l2: l2})
}

// AddByThreePlanes creates a datum point at the intersection of three planes (#1842).
func (c *WorkPoints) AddByThreePlanes(p1, p2, p3 WorkRef) *WorkPoint {
	return c.addUser(threePlanesPointDef{p1: p1, p2: p2, p3: p3})
}
