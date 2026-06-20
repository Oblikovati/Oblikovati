// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// Constraint/dimension carry-over for sketch-to-sketch copy (#1083, follow-up to #151).
// CopyEntities duplicates geometry only; Inventor's CopyEntitiesTo also re-creates the
// constraints and dimensions whose operands are *entirely* within the copied set, dropping
// any that reference geometry left behind. CopyEntitiesWithConstraints adds that carry-over
// on top of the geometry clone: every operand is remapped through the source→clone maps, and
// a relation with any external operand is silently dropped.

// cloneMap resolves a source operand (a point or an entity) to its clone. Points are keyed
// separately from entities because a constraint references the shared *Point inside a line or
// arc, not the owning entity.
type cloneMap struct {
	points   map[*Point]*Point
	entities map[Entity]Entity
}

func (m *cloneMap) point(p *Point) (*Point, bool) {
	np, ok := m.points[p]
	return np, ok
}

func (m *cloneMap) line(l *Line) (*Line, bool) {
	c, ok := m.entities[l]
	if !ok {
		return nil, false
	}
	nl, ok := c.(*Line)
	return nl, ok
}

func (m *cloneMap) curve(c CircularCurve) (CircularCurve, bool) {
	e, ok := m.entities[c]
	if !ok {
		return nil, false
	}
	nc, ok := e.(CircularCurve)
	return nc, ok
}

func (m *cloneMap) arc(a *Arc) (*Arc, bool) {
	c, ok := m.entities[a]
	if !ok {
		return nil, false
	}
	na, ok := c.(*Arc)
	return na, ok
}

func (m *cloneMap) ellipse(e *Ellipse) (*Ellipse, bool) {
	c, ok := m.entities[e]
	if !ok {
		return nil, false
	}
	ne, ok := c.(*Ellipse)
	return ne, ok
}

// CopyEntitiesWithConstraints clones a selection from source into this sketch (translated by
// v) and carries over every geometric constraint and dimension whose operands lie entirely
// within the copied set, remapping operands to the clones and dropping any relation with an
// external operand (Inventor CopyEntitiesTo with constraint carry-over, #1083). A copied
// driving dimension mints a fresh parameter in this sketch (DimensionConstraints.nextName),
// so names never collide with the source's. Returns the created entity clones.
//
// Example: target.CopyEntitiesWithConstraints(source, source.Entities(), math.V2(50, 0))
// duplicates the whole source sketch 50 units to the right, fully constrained.
func (target *Sketch) CopyEntitiesWithConstraints(source *Sketch, ents []Entity, v math.Vector2) []Entity {
	clones, pmap, emap := target.cloneEntitiesFull(ents, translation(v))
	m := &cloneMap{points: pmap, entities: emap}
	for _, c := range source.geomCons.All() {
		target.geomCons.carryFrom(c, m)
	}
	for _, d := range source.dimCons.All() {
		target.carryDimension(d, m)
	}
	return clones
}

// carryFrom re-creates one source geometric constraint on this (target) collection via the
// clone map, returning whether it was carried. It is dropped when any operand lies outside
// the copied set or the kind is not a carryable 2D constraint. The dispatch is split into the
// point-anchored and line/curve-anchored kinds to keep each switch within the complexity bound.
func (g *GeometricConstraints) carryFrom(c Constraint, m *cloneMap) bool {
	if carried, matched := carryPointConstraint(g, c, m); matched {
		return carried
	}
	return carryLineCurveConstraint(g, c, m)
}

// carryPointConstraint handles the geometric constraints whose operands are points (some with a
// line or curve anchor). matched reports whether c was one of these kinds, so carryFrom can fall
// through to the line/curve kinds; carried reports whether it was actually re-created.
func carryPointConstraint(g *GeometricConstraints, c Constraint, m *cloneMap) (carried, matched bool) {
	switch v := c.(type) {
	case *CoincidentConstraint:
		return carryPoints(m, v.A, v.B, g.AddCoincident), true
	case *HorizontalConstraint:
		return carryPoints(m, v.A, v.B, g.AddHorizontal), true
	case *VerticalConstraint:
		return carryPoints(m, v.A, v.B, g.AddVertical), true
	case *MidpointConstraint:
		return carryPointLine(m, v.P, v.L, g.AddMidpoint), true
	case *PointOnLineConstraint:
		return carryPointLine(m, v.P, v.L, g.AddPointOnLine), true
	case *PointOnCircleConstraint:
		return carryPointCurve(m, v.P, v.C, g.AddPointOnCircle), true
	case *SymmetryConstraint:
		return carrySymmetry(m, v, g), true
	case *FixConstraint:
		return carryFix(m, v.P, g), true
	}
	return false, false
}

// carryLineCurveConstraint handles the geometric constraints whose operands are whole lines or
// circular curves, returning whether the constraint was carried (false for any other kind).
func carryLineCurveConstraint(g *GeometricConstraints, c Constraint, m *cloneMap) bool {
	switch v := c.(type) {
	case *ParallelConstraint:
		return carryLines(m, v.L1, v.L2, g.AddParallel)
	case *PerpendicularConstraint:
		return carryLines(m, v.L1, v.L2, g.AddPerpendicular)
	case *CollinearConstraint:
		return carryLines(m, v.L1, v.L2, g.AddCollinear)
	case *EqualLengthConstraint:
		return carryLines(m, v.L1, v.L2, g.AddEqualLength)
	case *ConcentricConstraint:
		return carryCurves(m, v.C1, v.C2, g.AddConcentric)
	case *EqualRadiusConstraint:
		return carryCurves(m, v.C1, v.C2, g.AddEqualRadius)
	case *CircularTangentConstraint:
		return carryCurves(m, v.C1, v.C2, g.AddCircularTangent)
	case *TangentConstraint:
		return carryLineCurve(m, v.L, v.C, g.AddTangent)
	default:
		return false
	}
}

// The carry* helpers remap a fixed operand shape and invoke the matching factory iff every
// operand is inside the copied set. The factory's return value is discarded — the relation is
// already registered by Add — so they are generic over it (R).

func carryPoints[R any](m *cloneMap, a, b *Point, add func(*Point, *Point) R) bool {
	na, ok1 := m.point(a)
	nb, ok2 := m.point(b)
	if !ok1 || !ok2 {
		return false
	}
	add(na, nb)
	return true
}

func carryPointLine[R any](m *cloneMap, p *Point, l *Line, add func(*Point, *Line) R) bool {
	np, ok1 := m.point(p)
	nl, ok2 := m.line(l)
	if !ok1 || !ok2 {
		return false
	}
	add(np, nl)
	return true
}

func carryPointCurve[R any](m *cloneMap, p *Point, c CircularCurve, add func(*Point, CircularCurve) R) bool {
	np, ok1 := m.point(p)
	nc, ok2 := m.curve(c)
	if !ok1 || !ok2 {
		return false
	}
	add(np, nc)
	return true
}

func carryLines[R any](m *cloneMap, l1, l2 *Line, add func(*Line, *Line) R) bool {
	n1, ok1 := m.line(l1)
	n2, ok2 := m.line(l2)
	if !ok1 || !ok2 {
		return false
	}
	add(n1, n2)
	return true
}

func carryCurves[R any](m *cloneMap, c1, c2 CircularCurve, add func(CircularCurve, CircularCurve) R) bool {
	n1, ok1 := m.curve(c1)
	n2, ok2 := m.curve(c2)
	if !ok1 || !ok2 {
		return false
	}
	add(n1, n2)
	return true
}

func carryLineCurve[R any](m *cloneMap, l *Line, c CircularCurve, add func(*Line, CircularCurve) R) bool {
	nl, ok1 := m.line(l)
	nc, ok2 := m.curve(c)
	if !ok1 || !ok2 {
		return false
	}
	add(nl, nc)
	return true
}

func carrySymmetry(m *cloneMap, c *SymmetryConstraint, g *GeometricConstraints) bool {
	a, ok1 := m.point(c.A)
	b, ok2 := m.point(c.B)
	about, ok3 := m.line(c.About)
	if !ok1 || !ok2 || !ok3 {
		return false
	}
	g.AddSymmetry(a, b, about)
	return true
}

func carryFix(m *cloneMap, p *Point, g *GeometricConstraints) bool {
	np, ok := m.point(p)
	if !ok {
		return false
	}
	g.AddFix(np)
	return true
}

// carryDimension re-creates one source dimension on this (target) sketch, preserving its
// expression, driven/driving role and any drive limits. Returns whether it was carried.
func (target *Sketch) carryDimension(d *DimensionConstraint, m *cloneMap) bool {
	nd, ok := target.recreateDimension(d, m)
	if !ok {
		return false
	}
	nd.SetDriven(d.driven)
	if d.limits.Enabled {
		nd.SetLimits(d.limits.Min, d.limits.Max)
	}
	return true
}

// recreateDimension remaps the source dimension's refs to their clones and calls the matching
// factory with the source expression. A dimension whose refs are not all inside the copied set,
// or a kind that is not a 2D dimension (the 3D kinds), is dropped (nil, false). The dispatch is
// split into the core and advanced kinds to keep each switch within the complexity bound; both
// "wrong kind" and "external operand" yield (nil, false), so the only success signal is (nd, true).
func (target *Sketch) recreateDimension(d *DimensionConstraint, m *cloneMap) (*DimensionConstraint, bool) {
	if nd, ok := target.recreateCoreDimension(d, m); ok {
		return nd, true
	}
	return target.recreateAdvancedDimension(d, m)
}

// recreateCoreDimension handles the original M21 dimension kinds (distance, angle, radius,
// diameter, arc length).
func (target *Sketch) recreateCoreDimension(d *DimensionConstraint, m *cloneMap) (*DimensionConstraint, bool) {
	dc := target.dimCons
	expr := d.param.Expression()
	switch d.kind {
	case DistanceDim:
		if a, b, ok := m.refPoints2(d.refs); ok {
			return added(dc.AddDistance(a, b, expr))
		}
	case AngleDim:
		if l1, l2, ok := m.refLines2(d.refs); ok {
			return added(dc.AddAngle(l1, l2, expr))
		}
	case RadiusDim:
		if c, ok := m.refCurve(d.refs); ok {
			return added(dc.AddRadius(c, expr))
		}
	case DiameterDim:
		if c, ok := m.refCurve(d.refs); ok {
			return added(dc.AddDiameter(c, expr))
		}
	case ArcLengthDim:
		if a, ok := m.refArc(d.refs); ok {
			return added(dc.AddArcLength(a, expr))
		}
	}
	return nil, false
}

// recreateAdvancedDimension handles the M21-F07/#152 dimension kinds (offset, three-point angle,
// ellipse radius, tangent distance).
func (target *Sketch) recreateAdvancedDimension(d *DimensionConstraint, m *cloneMap) (*DimensionConstraint, bool) {
	dc := target.dimCons
	expr := d.param.Expression()
	switch d.kind {
	case OffsetDim:
		if p, l, ok := m.refPointLine(d.refs); ok {
			return added(dc.AddOffsetDim(p, l, expr))
		}
	case ThreePointAngleDim:
		if vx, a, b, ok := m.refPoints3(d.refs); ok {
			return added(dc.AddThreePointAngle(vx, a, b, expr))
		}
	case EllipseRadiusDim:
		if e, ok := m.refEllipse(d.refs); ok {
			return added(dc.AddEllipseRadius(e, expr))
		}
	case TangentDistanceDim:
		if l, c, ok := m.refLineCurve(d.refs); ok {
			return added(dc.AddTangentDistance(l, c, d.farSide, expr))
		}
	}
	return nil, false
}

// added adapts a (dimension, error) factory result: a creation error (e.g. an expression that
// references a parameter absent in the target) drops the dimension rather than failing the copy.
func added(d *DimensionConstraint, err error) (*DimensionConstraint, bool) {
	if err != nil {
		return nil, false
	}
	return d, true
}

// The ref* helpers extract a dimension's typed operands from its []Entity refs and remap each
// to its clone, reporting false when any is the wrong type or outside the copied set.

func (m *cloneMap) refPoint(e Entity) (*Point, bool) {
	p, ok := e.(*Point)
	if !ok {
		return nil, false
	}
	return m.point(p)
}

func (m *cloneMap) refPoints2(refs []Entity) (*Point, *Point, bool) {
	if len(refs) != 2 {
		return nil, nil, false
	}
	a, ok1 := m.refPoint(refs[0])
	b, ok2 := m.refPoint(refs[1])
	return a, b, ok1 && ok2
}

func (m *cloneMap) refPoints3(refs []Entity) (*Point, *Point, *Point, bool) {
	if len(refs) != 3 {
		return nil, nil, nil, false
	}
	a, ok1 := m.refPoint(refs[0])
	b, ok2 := m.refPoint(refs[1])
	c, ok3 := m.refPoint(refs[2])
	return a, b, c, ok1 && ok2 && ok3
}

func (m *cloneMap) refLines2(refs []Entity) (*Line, *Line, bool) {
	if len(refs) != 2 {
		return nil, nil, false
	}
	l1, ok1 := refLine(m, refs[0])
	l2, ok2 := refLine(m, refs[1])
	return l1, l2, ok1 && ok2
}

func (m *cloneMap) refPointLine(refs []Entity) (*Point, *Line, bool) {
	if len(refs) != 2 {
		return nil, nil, false
	}
	p, ok1 := m.refPoint(refs[0])
	l, ok2 := refLine(m, refs[1])
	return p, l, ok1 && ok2
}

func (m *cloneMap) refLineCurve(refs []Entity) (*Line, CircularCurve, bool) {
	if len(refs) != 2 {
		return nil, nil, false
	}
	l, ok1 := refLine(m, refs[0])
	c, ok2 := refCurve(m, refs[1])
	return l, c, ok1 && ok2
}

func (m *cloneMap) refCurve(refs []Entity) (CircularCurve, bool) {
	if len(refs) != 1 {
		return nil, false
	}
	return refCurve(m, refs[0])
}

func (m *cloneMap) refArc(refs []Entity) (*Arc, bool) {
	if len(refs) != 1 {
		return nil, false
	}
	a, ok := refs[0].(*Arc)
	if !ok {
		return nil, false
	}
	return m.arc(a)
}

func (m *cloneMap) refEllipse(refs []Entity) (*Ellipse, bool) {
	if len(refs) != 1 {
		return nil, false
	}
	e, ok := refs[0].(*Ellipse)
	if !ok {
		return nil, false
	}
	return m.ellipse(e)
}

func refLine(m *cloneMap, e Entity) (*Line, bool) {
	l, ok := e.(*Line)
	if !ok {
		return nil, false
	}
	return m.line(l)
}

func refCurve(m *cloneMap, e Entity) (CircularCurve, bool) {
	c, ok := e.(CircularCurve)
	if !ok {
		return nil, false
	}
	return m.curve(c)
}
