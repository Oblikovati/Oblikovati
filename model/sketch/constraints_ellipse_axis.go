// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// Ellipse-axis operands (#1879): Inventor's parallel/perpendicular/collinear and horizontal/
// vertical accept an ellipse (or elliptical arc) operand, its constrained direction being the
// ellipse's major or minor axis (UseEllipse*MajorAxis). An ellipse's orientation IS a solver DOF
// (#1879 AC2 — see solve_sketch.go and ellipse_orientation.go): the operand's direction is a live
// function of the ellipse's orientation angle, so the solver can rotate the ellipse to satisfy the
// relation (this is what makes horizontal/vertical of an ellipse axis solvable — the ellipse turns
// to align to the fixed world axis). The residuals reuse the scale-invariant angle/perp-distance
// primitives, so the unit-length axis direction is fine.

// ellipseAxisSource is an ellipse-like entity that offers a centre and an orientation-angle DOF —
// satisfied by both Ellipse and EllipticalArc (#1879).
type ellipseAxisSource interface {
	Entity
	axisCentre() *Point
	// axisAngle is the ellipse's major-axis orientation DOF; the operand builds its direction
	// from it so the solver can rotate the ellipse.
	axisAngle() *math.Scalar
}

func (e *Ellipse) axisCentre() *Point       { return e.Center }
func (e *EllipticalArc) axisCentre() *Point { return e.Center }

// AxisOperand is a direction operand of an axis-relation constraint: a line (direction from its
// endpoints, DOFs), an ellipse axis (direction from the ellipse's orientation DOF, anchored at its
// centre), or a fixed world axis (a constant direction). The interface is sealed (its methods are
// unexported); construct one with LineAxis, EllipseAxisOf, or the horizontal/vertical factories.
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
	return []*math.Scalar{&c.X, &c.Y, o.E.axisAngle()}
}
func (o ellipseAxisOperand) readDir(v []ad.Number, off int) (ad.Vec2, ad.Vec2, int) {
	// The centre is the anchor DOF (used by collinear); the direction is (cosθ, sinθ) of the
	// orientation DOF, so the solver can rotate the ellipse. The minor axis is a quarter turn on.
	centre := ad.V2(v[off], v[off+1])
	angle := v[off+2]
	if !o.major {
		angle = angle.AddConst(stdmath.Pi / 2)
	}
	return ad.V2(angle.Cos(), angle.Sin()), centre, 3
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

// worldAxisOperand is a fixed world direction (the +X horizontal or +Y vertical reference) — the
// immovable operand of an ellipse horizontal/vertical constraint. It contributes no DOF, names no
// entity, and always remaps (it survives any copy) (#1879 AC2).
type worldAxisOperand struct{ dir math.Vector2 }

func (o worldAxisOperand) dirVars() []*math.Scalar { return nil }
func (o worldAxisOperand) readDir(_ []ad.Number, _ int) (ad.Vec2, ad.Vec2, int) {
	d := ad.V2(ad.Const(float64(o.dir.X)), ad.Const(float64(o.dir.Y)))
	return d, ad.V2(ad.Const(0), ad.Const(0)), 0
}
func (o worldAxisOperand) operandEntity() Entity               { return nil }
func (o worldAxisOperand) axisMajor() bool                     { return false }
func (o worldAxisOperand) remap(*cloneMap) (AxisOperand, bool) { return o, true }

// axisRelation is the geometric relation an [AxisRelationConstraint] enforces between its two
// operand directions.
type axisRelation int

const (
	axisParallel axisRelation = iota
	axisPerpendicular
	axisCollinear
)

// AxisRelationConstraint aligns two direction operands — at least one an ellipse axis — parallel,
// perpendicular, or collinear, or aligns one ellipse axis to a fixed world axis (horizontal/
// vertical, #1879). Every ellipse operand's direction tracks the ellipse's orientation DOF, so the
// solver satisfies the relation by rotating whichever operand is free (the ellipse for a world-axis
// relation; either an ellipse or a line for an ellipse-line relation).
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

// AddEllipseHorizontal / AddEllipseVertical make an ellipse's chosen axis horizontal / vertical by
// rotating the ellipse until that axis is parallel to the fixed world +X / +Y direction (#1879
// AC2). The ellipse operand is the free (rotatable) side; the world axis is the datum. Like the
// existing parallel/perpendicular constraints, an operand sitting EXACTLY perpendicular to the
// target starts at the angular residual's zero-gradient dead point and will not rotate — a
// measure-zero configuration real geometry does not hit.
func (g *GeometricConstraints) AddEllipseHorizontal(e AxisOperand) *AxisRelationConstraint {
	return g.addAxisRelation(e, worldAxisOperand{dir: math.V2(1, 0)}, axisParallel, EllipseHorizontalKind)
}
func (g *GeometricConstraints) AddEllipseVertical(e AxisOperand) *AxisRelationConstraint {
	return g.addAxisRelation(e, worldAxisOperand{dir: math.V2(0, 1)}, axisParallel, EllipseVerticalKind)
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
