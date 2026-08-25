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
//
// A pin on a point that is already fixed — grounded to a reference anchor, or held by a Fix
// constraint — is dropped rather than added. A drag pin is just another equal-weight positional
// residual (a FixConstraint), so pinning a point that a hard constraint already holds at the origin
// would make the least-squares solve split the difference and let a coincident-to-origin endpoint
// drag off the origin (#2160). Dropping that pin lets the hard constraint hold the point while the
// rest of the dragged geometry follows the cursor.
func (s *Sketch) DragSolve(pins []PinTarget) SolveResult {
	grounded := s.groundedPoints()
	cons := append([]Constraint(nil), s.Constraints()...)
	for _, pt := range pins {
		if grounded[pt.P] {
			continue
		}
		cons = append(cons, dragFix(pt.P, pt.Target))
	}
	r := Solve(cons, s.variables(), Options{})
	s.syncEllipseAxes()
	return r
}

// groundedPoints returns the points a drag must not pin: those transitively coincident with a fixed
// anchor. The seeds are the reference points (projected/included anchors, excluded from the solver
// so they never move) and any point held by a Fix constraint; the grounded flag then floods across
// coincidence constraints, so a point coincident to the origin — or coincident to a point that is —
// is grounded. See DragSolve for why pinning such a point is wrong (#2160).
func (s *Sketch) groundedPoints() map[*Point]bool {
	grounded, adj := s.groundSeedsAndCoincidences()
	floodGrounded(grounded, adj)
	return grounded
}

// groundSeedsAndCoincidences seeds the grounded set with every fixed anchor (reference points and
// Fix-held points) and builds the undirected coincidence adjacency the flood spreads along.
func (s *Sketch) groundSeedsAndCoincidences() (map[*Point]bool, map[*Point][]*Point) {
	grounded := make(map[*Point]bool, len(s.refPts))
	for _, p := range s.refPts {
		grounded[p] = true
	}
	adj := map[*Point][]*Point{}
	for _, c := range s.Constraints() {
		switch k := c.(type) {
		case *CoincidentConstraint:
			adj[k.A] = append(adj[k.A], k.B)
			adj[k.B] = append(adj[k.B], k.A)
		case *FixConstraint:
			grounded[k.P] = true
		}
	}
	return grounded, adj
}

// floodGrounded spreads the grounded flag across coincidence edges in place: a point coincident to a
// grounded point is itself grounded.
func floodGrounded(grounded map[*Point]bool, adj map[*Point][]*Point) {
	stack := make([]*Point, 0, len(grounded))
	for p := range grounded {
		stack = append(stack, p)
	}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, q := range adj[p] {
			if !grounded[q] {
				grounded[q] = true
				stack = append(stack, q)
			}
		}
	}
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

// A spline is dragged by all its interpolation/control points, so a spline drag translates
// the whole curve rigidly — closing the silent hole where a spline drag pinned nothing
// (#1633, audit I10). The coverage table in capability_coverage_test.go now records this.
func (s *Spline) definingPoints() []*Point { return s.Points }

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
