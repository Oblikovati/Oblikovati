// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "github.com/Oblikovati/oblikovati/math"

// Constraint is one relation the solver must satisfy. It exposes residual
// equations (each zero when satisfied) and the scalar variables it reads, as
// pointers into the entities (point coordinates, a circle radius, …). Exposing
// variables as []*math.Scalar keeps the solver dimension-agnostic: 2D points, 3D
// points and scalar DOFs all look the same to it (ADR-0009, modeling/00).
type Constraint interface {
	// EntityID returns the constraint's sketch-local id.
	EntityID() ID
	// Residuals returns the constraint's residual values; all zero ⇒ satisfied.
	Residuals() []float64
	// Variables returns the scalar DOFs this constraint touches, by pointer.
	Variables() []*math.Scalar
}

// constraintBase carries the constraint id.
type constraintBase struct{ id ID }

func newConstraint() constraintBase { return constraintBase{id: nextID()} }

// EntityID implements [Constraint].
func (c *constraintBase) EntityID() ID { return c.id }

// GeometricConstraints owns a sketch's geometric (non-dimensional) constraints and
// is the factory for them (the COM GeometricConstraints collection).
type GeometricConstraints struct {
	items []Constraint
}

// All returns the constraints in creation order (consumed by the solver).
func (g *GeometricConstraints) All() []Constraint {
	out := make([]Constraint, len(g.items))
	copy(out, g.items)
	return out
}

// Count returns the number of geometric constraints; Item returns the i-th.
func (g *GeometricConstraints) Count() int            { return len(g.items) }
func (g *GeometricConstraints) Item(i int) Constraint { return g.items[i] }

// Delete removes a constraint, reporting whether it was present (used to resolve
// over-constraint by dropping the offending relation).
func (g *GeometricConstraints) Delete(c Constraint) bool {
	for i, existing := range g.items {
		if existing == c {
			g.items = append(g.items[:i], g.items[i+1:]...)
			return true
		}
	}
	return false
}

func (g *GeometricConstraints) add(c Constraint) {
	g.items = append(g.items, c)
}

// Add appends an already-constructed constraint to the collection and returns it. It is
// the general seam the 3D constraint factories use (M22-F05) until they grow typed
// helpers; the 2D typed factories call the unexported add directly.
func (g *GeometricConstraints) Add(c Constraint) Constraint {
	g.add(c)
	return c
}
