// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// Where a geometric constraint's on-screen marker belongs (Show Constraints).
//
// A constraint knows the scalar DOFs it acts on — Variables() hands back pointers straight into
// entity coordinates — and that is enough to place it: the marker sits at the centroid of the
// points those DOFs live in. Deriving the anchor from Variables rather than from a per-kind table
// is what makes this total: every constraint kind already implements Variables (the solver
// requires it), so a kind added later is drawn without anyone remembering to register it. The two
// existing per-kind registries in this package — serialization codecs and copy carriers — each
// needed a closure test to catch the kinds people forgot; this needs none.

// ConstraintGlyph is one geometric constraint's marker: the constraint, what kind it is, and the
// sketch point it annotates.
type ConstraintGlyph struct {
	Constraint Constraint
	Kind       ConstraintKind
	At         math.Point2
}

// kindedConstraint is a constraint that names its kind — which a marker needs in order to pick a
// glyph. The solver's own bookkeeping (arc circularity, fillet tangency) names none, and does not
// live in this collection either; it is the arc's property, not a relation the user placed.
type kindedConstraint interface {
	Constraint
	ConstraintKind() ConstraintKind
}

// ConstraintGlyphs returns a marker for every geometric constraint the sketch carries, in creation
// order. Two kinds are left out, and both are deliberate: a system-owned constraint, because a
// marker the user can select but not delete is a dead end; and a record with no numeric footprint
// — a text-box anchor, an add-in's Custom tag — because a relation that constrains nothing has
// nowhere on the sketch to sit. Both fall out of the same rule the deletion path already uses, so
// what is drawn is exactly what Delete can remove.
//
//	for _, g := range sk.ConstraintGlyphs() { draw(g.Kind, g.At) }
func (s *Sketch) ConstraintGlyphs() []ConstraintGlyph {
	owners := s.variableOwners()
	var out []ConstraintGlyph
	for _, c := range s.geomCons.All() {
		kc, named := c.(kindedConstraint)
		if !named || !userDeletable(c) {
			continue
		}
		if at, ok := constraintAnchor(kc, owners); ok {
			out = append(out, ConstraintGlyph{Constraint: c, Kind: kc.ConstraintKind(), At: at})
		}
	}
	return out
}

// constraintAnchor averages the distinct points a constraint's variables live in. Distinct
// matters: a coincidence exposes both coordinates of both its points, and counting each point
// twice would still average correctly, but a line-to-line relation exposing four coordinates of
// one line and two of another would pull the marker toward the busier operand.
func constraintAnchor(c Constraint, owners map[*math.Scalar]*Point) (math.Point2, bool) {
	seen := map[*Point]bool{}
	var sum math.Vector2
	for _, v := range c.Variables() {
		p, known := owners[v]
		if !known || seen[p] {
			continue
		}
		seen[p] = true
		sum = sum.Add(math.V2(p.X, p.Y))
	}
	if len(seen) == 0 {
		return math.Point2{}, false
	}
	return math.P2(0, 0).TranslateBy(sum.Scale(1 / float64(len(seen)))), true
}

// userDeletable reports whether the user may remove this constraint, matching the rule
// [GeometricConstraints.DeleteAllowed] enforces.
func userDeletable(c Constraint) bool {
	nd, marked := c.(NonDeletable)
	return !marked || nd.Deletable()
}

// variableOwners maps every point coordinate in the sketch to its point.
//
// It walks s.pts — every point the sketch has allocated — NOT the Points() collection. Those are
// different sets: Points().Add registers a standalone point, while Lines().AddByTwoPoints (how
// every tool and the wire actually draw a line) allocates its endpoints without registering them.
// Walking the collection therefore found the endpoints of hand-built test fixtures and none of a
// real sketch's, so every marker was silently dropped in the running app while the tests passed.
//
// Only coordinates are mapped, and that is enough: a circular operand exposes its centre alongside
// its radius (Circle.circularVars), and an ellipse-axis operand its centre alongside its
// orientation, so every relation in the vocabulary reaches at least one point.
func (s *Sketch) variableOwners() map[*math.Scalar]*Point {
	owners := make(map[*math.Scalar]*Point, len(s.pts)*2)
	for _, p := range s.pts {
		owners[&p.X], owners[&p.Y] = p, p
	}
	return owners
}
