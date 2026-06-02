// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "github.com/Oblikovati/oblikovati/math"

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

func (c *Coincident3D) Residuals() []float64 {
	return []float64{c.A.X - c.B.X, c.A.Y - c.B.Y, c.A.Z - c.B.Z}
}

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

func (c *Collinear3D) Residuals() []float64 {
	u := c.A.Position().VectorTo(c.B.Position())
	v := c.A.Position().VectorTo(c.C.Position())
	cr := u.Cross(v)
	return []float64{cr.X, cr.Y, cr.Z}
}

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

func (c *Concentric3D) Residuals() []float64 {
	return []float64{c.Center1.X - c.Center2.X, c.Center1.Y - c.Center2.Y, c.Center1.Z - c.Center2.Z}
}

func (c *Concentric3D) Variables() []*math.Scalar {
	return []*math.Scalar{&c.Center1.X, &c.Center1.Y, &c.Center1.Z, &c.Center2.X, &c.Center2.Y, &c.Center2.Z}
}

// Equal3D forces two scalar DOFs (e.g. radii or distances) to be equal.
type Equal3D struct {
	constraintBase
	A, B *math.Scalar
}

// NewEqual3D constrains two scalar DOFs to be equal.
func NewEqual3D(a, b *math.Scalar) *Equal3D {
	return &Equal3D{constraintBase: newConstraint(), A: a, B: b}
}
func (c *Equal3D) Residuals() []float64      { return []float64{*c.A - *c.B} }
func (c *Equal3D) Variables() []*math.Scalar { return []*math.Scalar{c.A, c.B} }

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
