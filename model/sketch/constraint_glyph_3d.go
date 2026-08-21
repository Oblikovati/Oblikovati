// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// Where a 3D sketch's geometric-constraint markers belong (Show Constraints in a 3D sketch, #1998).
//
// This is the 3D counterpart of ConstraintGlyphs (constraint_glyph.go). It follows the same rule —
// a marker per operand, derived from each constraint's Variables() so every kind is drawn without a
// per-kind table — but anchors in model space: a 3D sketch has no host plane to map through, so the
// head draws each anchor directly. The anchor is the operand's constrained-points centroid (a
// line's midpoint, a point's own position); the 2D path's extra pass that slides a marker onto a
// curve's contact point is a refinement left out here, because the centroid already sits on or
// beside the operand it marks and never floats between two.

// ConstraintGlyph3D is one 3D geometric constraint's marker: the constraint, its kind, and the
// model-space point it annotates.
type ConstraintGlyph3D struct {
	Constraint Constraint
	Kind       ConstraintKind
	At         math.Point3
}

// ConstraintGlyphs returns the markers for every geometric constraint the 3D sketch carries, in
// creation order — one per operand the constraint relates, so a relation between two lines is marked
// on each (at its midpoint) and a relation whose operands coincide is marked once at the shared
// point. Two kinds are left out for the same reasons as the 2D path: a system-owned constraint (a
// marker the user cannot delete is a dead end) and a record with no numeric footprint (it has
// nowhere on the sketch to sit). What is drawn is exactly what Delete can remove.
//
//	for _, g := range sk.ConstraintGlyphs() { draw(g.Kind, g.At) }
func (s *Sketch3D) ConstraintGlyphs() []ConstraintGlyph3D {
	owners := s.variableOwners3D()
	var out []ConstraintGlyph3D
	for _, c := range s.geomCons.All() {
		kc, named := c.(kindedConstraint)
		if !named || !userDeletable(c) {
			continue
		}
		for _, at := range s.constraintAnchors3D(kc, owners) {
			out = append(out, ConstraintGlyph3D{Constraint: c, Kind: kc.ConstraintKind(), At: at})
		}
	}
	return out
}

// variableOwners3D maps every 3D point coordinate to its point — the free position points and the
// fixed reference anchors, both of which a constraint may name. A constraint's Variables() are
// pointers straight into these coordinates, so the map turns them back into the points it moves.
func (s *Sketch3D) variableOwners3D() map[*math.Scalar]*Point3D {
	owners := make(map[*math.Scalar]*Point3D, (len(s.pts)+len(s.refPts))*3)
	for _, group := range [][]*Point3D{s.pts, s.refPts} {
		for _, p := range group {
			owners[&p.X], owners[&p.Y], owners[&p.Z] = p, p, p
		}
	}
	return owners
}
