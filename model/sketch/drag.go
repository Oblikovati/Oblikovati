// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// Interactive drag-resolve: dragging a sketch entity should not blindly translate its points
// (which breaks any constraint touching them) but re-solve the sketch with the dragged points
// pinned to the cursor target, so the dependent geometry follows within the remaining degrees of
// freedom — Inventor's behaviour. This is the constrained counterpart to MoveEntities; the free
// (unconstrained) case falls out of it automatically (a lone pin just moves the point).

// PinTarget pairs a sketch point with the position a drag wants it at.
type PinTarget struct {
	P      *Point
	Target math.Point2
}

// dragPin is a temporary constraint that pulls point P toward (tx,ty). It is appended to the
// constraint set only for the duration of one DragSolve and is never persisted on the sketch.
type dragPin struct {
	constraintBase
	P      *Point
	tx, ty math.Scalar
}

func (c *dragPin) Residuals() []float64      { return []float64{c.P.X - c.tx, c.P.Y - c.ty} }
func (c *dragPin) Variables() []*math.Scalar { return []*math.Scalar{&c.P.X, &c.P.Y} }

// DragSolve re-solves the sketch with the given points temporarily pinned to their drag targets.
// The dragged points follow the cursor while the existing constraints pull the dependent geometry
// along (and hold the pinned points back along any directions they are constrained in). The pins
// are not persisted; geometry is left in the solved configuration. Returns the solve result.
func (s *Sketch) DragSolve(pins []PinTarget) SolveResult {
	cons := append([]Constraint(nil), s.Constraints()...)
	for _, pt := range pins {
		cons = append(cons, &dragPin{constraintBase: newConstraint(), P: pt.P, tx: pt.Target.X, ty: pt.Target.Y})
	}
	return Solve(cons, s.variables(), Options{})
}

// DefiningPoints returns the entity's constrainable points — the handles a drag pins. Dragging a
// single point pins just it; dragging a whole curve pins all its points so it translates rigidly
// (the solver then re-satisfies the constraints touching them).
func DefiningPoints(e Entity) []*Point {
	switch t := e.(type) {
	case *Point:
		return []*Point{t}
	case *Line:
		return []*Point{t.A, t.B}
	case *Circle:
		return []*Point{t.Center}
	case *Arc:
		return []*Point{t.Center, t.Start, t.End}
	case *Ellipse:
		return []*Point{t.Center}
	case *EllipticalArc:
		return []*Point{t.Center}
	default:
		return nil
	}
}
