// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// Ellipse-axis operands (#1879): Inventor's parallel/perpendicular/collinear and horizontal/
// vertical accept an ellipse (or elliptical arc) operand, its constrained direction being the
// ellipse's major or minor axis (UseEllipse*MajorAxis). An ellipse's axis direction is a fixed
// reference (the major-axis direction is not a solver DOF — see solve_sketch.go's variable
// universe), so an ellipse operand contributes a constant direction anchored at its centre; the
// line operand adjusts. This is a direction-alignment relation, not a branch-selection one, so
// the residuals reuse the existing scale-invariant angle/perp-distance primitives.

// ellipseAxisSource is an ellipse-like entity that offers a centre and a major-axis direction —
// satisfied by both Ellipse and EllipticalArc (#1879).
type ellipseAxisSource interface {
	Entity
	axisCentre() *Point
	// axisVector returns the (unnormalised) major or minor axis direction; adSineAngle/
	// adCosAngle normalise, so a raw vector is fine.
	axisVector(major bool) math.Vector2
}

func (e *Ellipse) axisCentre() *Point { return e.Center }
func (e *Ellipse) axisVector(major bool) math.Vector2 {
	return ellipseAxisVector(e.MajorAxis, major)
}
func (e *EllipticalArc) axisCentre() *Point { return e.Center }
func (e *EllipticalArc) axisVector(major bool) math.Vector2 {
	return ellipseAxisVector(e.MajorAxis, major)
}

// ellipseAxisVector returns the major axis as-is, or the minor axis as its left perpendicular.
func ellipseAxisVector(major math.Vector2, useMajor bool) math.Vector2 {
	if useMajor {
		return major
	}
	return math.V2(-major.Y, major.X)
}

// AxisOperand is a direction operand of an axis-relation constraint: a line (direction from its
// endpoints, DOFs) or an ellipse axis (a fixed direction anchored at the ellipse centre). The
// interface is sealed (its methods are unexported); construct one with LineAxis or EllipseAxisOf.
type AxisOperand interface {
	dirVars() []*math.Scalar
	// readDir reads the operand's direction and anchor point from the seeded row starting at
	// off, returning the number of variables consumed.
	readDir(v []ad.Number, off int) (dir, anchor ad.Vec2, consumed int)
	// operandEntity is the entity the operand names for enumeration (nil for a world axis).
	operandEntity() Entity
	// axisMajor reports whether an ellipse operand uses its major axis (false for line/world
	// operands); persisted so the major/minor choice survives a round trip (#1879).
	axisMajor() bool
	// remap rebuilds the operand against a sketch clone, reporting ok=false when its entity
	// lies outside the copied set (a world axis always remaps).
	remap(m *cloneMap) (AxisOperand, bool)
}

// lineAxisOperand is a line's direction operand.
type lineAxisOperand struct{ L *Line }

// LineAxis wraps a line as an axis-relation operand.
func LineAxis(l *Line) AxisOperand { return lineAxisOperand{L: l} }

func (o lineAxisOperand) dirVars() []*math.Scalar {
	return []*math.Scalar{&o.L.A.X, &o.L.A.Y, &o.L.B.X, &o.L.B.Y}
}
func (o lineAxisOperand) readDir(v []ad.Number, off int) (ad.Vec2, ad.Vec2, int) {
	a, b := ad.V2(v[off], v[off+1]), ad.V2(v[off+2], v[off+3])
	return b.Sub(a), a, 4
}
func (o lineAxisOperand) operandEntity() Entity { return o.L }
func (o lineAxisOperand) axisMajor() bool       { return false }
func (o lineAxisOperand) remap(m *cloneMap) (AxisOperand, bool) {
	nl, ok := m.line(o.L)
	if !ok {
		return nil, false
	}
	return LineAxis(nl), true
}

// ellipseAxisOperand is an ellipse's major/minor axis: a fixed direction anchored at the centre.
type ellipseAxisOperand struct {
	E     ellipseAxisSource
	major bool
}

// EllipseAxisOf wraps an entity's major (or minor) axis as an operand, reporting ok=false when
// the entity is not an ellipse or elliptical arc.
func EllipseAxisOf(e Entity, major bool) (AxisOperand, bool) {
	src, ok := e.(ellipseAxisSource)
	if !ok {
		return nil, false
	}
	return ellipseAxisOperand{E: src, major: major}, true
}

func (o ellipseAxisOperand) dirVars() []*math.Scalar {
	c := o.E.axisCentre()
	return []*math.Scalar{&c.X, &c.Y}
}
func (o ellipseAxisOperand) readDir(v []ad.Number, off int) (ad.Vec2, ad.Vec2, int) {
	// The centre is a DOF (the anchor for collinear); the axis direction is a live constant.
	centre := ad.V2(v[off], v[off+1])
	d := o.E.axisVector(o.major)
	return ad.V2(ad.Const(float64(d.X)), ad.Const(float64(d.Y))), centre, 2
}
func (o ellipseAxisOperand) operandEntity() Entity { return o.E }
func (o ellipseAxisOperand) axisMajor() bool       { return o.major }
func (o ellipseAxisOperand) remap(m *cloneMap) (AxisOperand, bool) {
	ne, ok := m.entity(o.E)
	if !ok {
		return nil, false
	}
	return EllipseAxisOf(ne, o.major)
}

// axisRelation is the geometric relation an [AxisRelationConstraint] enforces between its two
// operand directions.
type axisRelation int

const (
	axisParallel axisRelation = iota
	axisPerpendicular
	axisCollinear
)

// AxisRelationConstraint aligns two direction operands — at least one an ellipse axis — parallel,
// perpendicular, or collinear (#1879). An ellipse operand is a fixed directional reference (its
// orientation is not a solver DOF), so it is the line operand that the solver moves to satisfy
// the relation; two fixed ellipse axes only hold when already so related.
type AxisRelationConstraint struct {
	constraintBase
	a, b     AxisOperand
	relation axisRelation
	kind     ConstraintKind
}

// AddEllipseParallel / AddEllipsePerpendicular / AddEllipseCollinear relate two operands, at
// least one an ellipse axis (the other typically a line the solver aligns to it).
func (g *GeometricConstraints) AddEllipseParallel(a, b AxisOperand) *AxisRelationConstraint {
	return g.addAxisRelation(a, b, axisParallel, EllipseParallelKind)
}
func (g *GeometricConstraints) AddEllipsePerpendicular(a, b AxisOperand) *AxisRelationConstraint {
	return g.addAxisRelation(a, b, axisPerpendicular, EllipsePerpendicularKind)
}
func (g *GeometricConstraints) AddEllipseCollinear(a, b AxisOperand) *AxisRelationConstraint {
	return g.addAxisRelation(a, b, axisCollinear, EllipseCollinearKind)
}

func (g *GeometricConstraints) addAxisRelation(a, b AxisOperand, rel axisRelation, kind ConstraintKind) *AxisRelationConstraint {
	c := &AxisRelationConstraint{constraintBase: newConstraint(), a: a, b: b, relation: rel, kind: kind}
	g.add(c)
	return c
}

// residualAD: the two operand directions are parallel (sine of the angle zero), perpendicular
// (cosine zero), or collinear (parallel AND the second anchor on the first operand's line). All
// three primitives are scale-invariant, so a fixed ellipse-axis direction of arbitrary length is
// fine (#1418, #1879).
func (c *AxisRelationConstraint) residualAD(v []ad.Number) []ad.Number {
	da, aa, n := c.a.readDir(v, 0)
	db, ab, _ := c.b.readDir(v, n)
	switch c.relation {
	case axisPerpendicular:
		return []ad.Number{adCosAngle(da, db)}
	case axisCollinear:
		return []ad.Number{adSineAngle(da, db), adSignedPerpDistance(da, ab.Sub(aa))}
	default: // axisParallel
		return []ad.Number{adSineAngle(da, db)}
	}
}
func (c *AxisRelationConstraint) Residuals() []float64 {
	return adResiduals(c.Variables(), c.residualAD)
}
func (c *AxisRelationConstraint) Partials() [][]float64 {
	return adPartials(c.Variables(), c.residualAD)
}
func (c *AxisRelationConstraint) Variables() []*math.Scalar {
	return append(c.a.dirVars(), c.b.dirVars()...)
}
