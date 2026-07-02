// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// Point3D is a constrainable point in a 3D sketch: three solver variables. The
// solver is dimension-agnostic — it sees [Constraint.Variables] as []*math.Scalar
// regardless of 2D or 3D — so the same Newton core (F05) solves 3D sketches
// (modeling/00, ADR-0009).
type Point3D struct {
	id      ID
	X, Y, Z math.Scalar
}

// NewPoint3D creates a 3D sketch point at the given position.
func NewPoint3D(p math.Point3) *Point3D {
	return &Point3D{id: nextID(), X: p.X, Y: p.Y, Z: p.Z}
}

// EntityID implements [Entity].
func (p *Point3D) EntityID() ID { return p.id }

// setID pins the point's local id to a persisted value (restore path only); see
// [entityBase.setID]. Keeps a 3D point's derived persistent reference key stable (#153).
func (p *Point3D) setID(id ID) { p.id = id }

// Position returns the point as a [math.Point3].
func (p *Point3D) Position() math.Point3 { return math.P3(p.X, p.Y, p.Z) }

// SetPosition moves the 3D point.
func (p *Point3D) SetPosition(q math.Point3) { p.X, p.Y, p.Z = q.X, q.Y, q.Z }

// Coincident3D makes two 3D points share a location.
type Coincident3D struct {
	constraintBase
	A, B *Point3D
}

// NewCoincident3D constrains two 3D points to coincide.
func NewCoincident3D(a, b *Point3D) *Coincident3D {
	return &Coincident3D{constraintBase: newConstraint(), A: a, B: b}
}

// residualAD: v = [A.xyz, B.xyz]; the two points must coincide.
func (c *Coincident3D) residualAD(v []ad.Number) []ad.Number {
	d := adV3(v, 0).Sub(adV3(v, 3))
	return []ad.Number{d.X, d.Y, d.Z}
}
func (c *Coincident3D) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *Coincident3D) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

func (c *Coincident3D) Variables() []*math.Scalar {
	return []*math.Scalar{&c.A.X, &c.A.Y, &c.A.Z, &c.B.X, &c.B.Y, &c.B.Z}
}

// Collinear3D forces three 3D points onto a common line (the cross product of the
// two spanning vectors vanishes — three residual components).
type Collinear3D struct {
	constraintBase
	A, B, C *Point3D
}

// NewCollinear3D constrains points a, b, c to be collinear.
func NewCollinear3D(a, b, c *Point3D) *Collinear3D {
	return &Collinear3D{constraintBase: newConstraint(), A: a, B: b, C: c}
}

// residualAD: v = [A.xyz, B.xyz, C.xyz]; (B−A)×(C−A) vanishes iff the three are collinear.
func (c *Collinear3D) residualAD(v []ad.Number) []ad.Number {
	a := adV3(v, 0)
	cr := adV3(v, 3).Sub(a).Cross(adV3(v, 6).Sub(a))
	return []ad.Number{cr.X, cr.Y, cr.Z}
}
func (c *Collinear3D) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *Collinear3D) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

func (c *Collinear3D) Variables() []*math.Scalar {
	return []*math.Scalar{&c.A.X, &c.A.Y, &c.A.Z, &c.B.X, &c.B.Y, &c.B.Z, &c.C.X, &c.C.Y, &c.C.Z}
}

// Concentric3D forces two 3D circle/arc centers (points) to coincide. (It is
// Coincident3D specialized to centers; kept distinct for intent/diagnostics.)
type Concentric3D struct {
	constraintBase
	Center1, Center2 *Point3D
}

// NewConcentric3D constrains two 3D centers to coincide.
func NewConcentric3D(c1, c2 *Point3D) *Concentric3D {
	return &Concentric3D{constraintBase: newConstraint(), Center1: c1, Center2: c2}
}

// residualAD: v = [C1.xyz, C2.xyz]; the two centers must coincide.
func (c *Concentric3D) residualAD(v []ad.Number) []ad.Number {
	d := adV3(v, 0).Sub(adV3(v, 3))
	return []ad.Number{d.X, d.Y, d.Z}
}
func (c *Concentric3D) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *Concentric3D) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

func (c *Concentric3D) Variables() []*math.Scalar {
	return []*math.Scalar{&c.Center1.X, &c.Center1.Y, &c.Center1.Z, &c.Center2.X, &c.Center2.Y, &c.Center2.Z}
}

// Equal3D forces the radius DOFs of two radius-bearing curves to be equal. It
// keeps the operand entities, not just their scalar DOFs: without them the
// constraint could not name its refs over the API nor serialize at all — an
// Equal3D save failed at runtime until #1625 (the #1416 drift class).
type Equal3D struct {
	constraintBase
	E1, E2 Entity
	A, B   *math.Scalar
}

// NewEqual3D constrains the radius DOFs of two radius-bearing curves (a circle
// or a helix) to be equal, rejecting entities without a radius DOF.
func NewEqual3D(e1, e2 Entity) (*Equal3D, error) {
	a, err := radiusDOFOf(e1)
	if err != nil {
		return nil, err
	}
	b, err := radiusDOFOf(e2)
	if err != nil {
		return nil, err
	}
	return &Equal3D{constraintBase: newConstraint(), E1: e1, E2: e2, A: a, B: b}, nil
}

// radiusDOFOf resolves an entity's radius solver DOF through its RadiusDOF
// capability (#1624).
func radiusDOFOf(e Entity) (*math.Scalar, error) {
	rd, ok := e.(interface{ RadiusDOF() *math.Scalar })
	if !ok {
		return nil, fmt.Errorf("sketch: equal needs radius-bearing curves (circle/helix), got %T", e)
	}
	return rd.RadiusDOF(), nil
}

// residualAD: v = [A, B]; the two scalar DOFs must be equal.
func (c *Equal3D) residualAD(v []ad.Number) []ad.Number { return []ad.Number{v[0].Sub(v[1])} }
func (c *Equal3D) Residuals() []float64                 { return adResiduals(c.Variables(), c.residualAD) }
func (c *Equal3D) Partials() [][]float64                { return adPartials(c.Variables(), c.residualAD) }
func (c *Equal3D) Variables() []*math.Scalar            { return []*math.Scalar{c.A, c.B} }

// CustomConstraint3D adapts an arbitrary residual function over given variables
// into a [Constraint], for relations not covered by the built-ins.
type CustomConstraint3D struct {
	constraintBase
	residual func() []float64
	vars     []*math.Scalar
}

// NewCustomConstraint3D builds a custom constraint from a residual closure and the
// variables it touches.
func NewCustomConstraint3D(residual func() []float64, vars []*math.Scalar) *CustomConstraint3D {
	return &CustomConstraint3D{constraintBase: newConstraint(), residual: residual, vars: vars}
}
func (c *CustomConstraint3D) Residuals() []float64      { return c.residual() }
func (c *CustomConstraint3D) Variables() []*math.Scalar { return c.vars }
