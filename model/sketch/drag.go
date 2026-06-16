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

// dragFix is a temporary pin pulling point p toward target. It reuses FixConstraint (a fix to a
// position) rather than re-declaring its residual/variable math; it is appended only for the
// duration of one DragSolve and is never added to the sketch's persisted constraints.
func dragFix(p *Point, target math.Point2) *FixConstraint {
	return &FixConstraint{constraintBase: newConstraint(), P: p, x0: target.X, y0: target.Y}
}

// DragSolve re-solves the sketch with the given points temporarily pinned to their drag targets.
// The dragged points follow the cursor while the existing constraints pull the dependent geometry
// along (and hold the pinned points back along any directions they are constrained in). The pins
// are not persisted; geometry is left in the solved configuration. Returns the solve result.
func (s *Sketch) DragSolve(pins []PinTarget) SolveResult {
	cons := append([]Constraint(nil), s.Constraints()...)
	for _, pt := range pins {
		cons = append(cons, dragFix(pt.P, pt.Target))
	}
	return Solve(cons, s.variables(), Options{})
}

// pointDefined is an entity that can name its constrainable points (each entity declares its own
// so DefiningPoints needs no type switch — which would duplicate entityFreedomVars in
// moveable_status.go). The method is unexported, sealing the set to this package's entities.
type pointDefined interface{ definingPoints() []*Point }

func (p *Point) definingPoints() []*Point         { return []*Point{p} }
func (l *Line) definingPoints() []*Point          { return []*Point{l.A, l.B} }
func (c *Circle) definingPoints() []*Point        { return []*Point{c.Center} }
func (a *Arc) definingPoints() []*Point           { return []*Point{a.Center, a.Start, a.End} }
func (e *Ellipse) definingPoints() []*Point       { return []*Point{e.Center} }
func (e *EllipticalArc) definingPoints() []*Point { return []*Point{e.Center} }

// DefiningPoints returns the entity's constrainable points — the handles a drag pins. Dragging a
// single point pins just it; dragging a whole curve pins all its points so it translates rigidly
// (the solver then re-satisfies the constraints touching them). Entities without points (text,
// images) and nil return none.
func DefiningPoints(e Entity) []*Point {
	if d, ok := e.(pointDefined); ok {
		return d.definingPoints()
	}
	return nil
}
