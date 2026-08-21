// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// Where a 3D relation's markers sit — the 3D counterpart of constraint_glyph_anchor.go.
//
// A relation is anchored once per operand, each marker at that operand's constrained-points
// centroid: a line's midpoint, a circle's centre, a lone point's own position. Anchoring per
// operand (not per point) is what keeps a parallel between two lines to two markers rather than
// four, and a relation whose operands coincide (a coincidence, a concentricity) collapses to the
// single marker at the shared place.

// constraintAnchors3D returns one anchor per operand the constraint acts on. Operands that end up in
// the same place — a coincidence's two points, a tangency's contact — collapse to a single marker.
func (s *Sketch3D) constraintAnchors3D(c Constraint, owners map[*math.Scalar]*Point3D) []math.Point3 {
	groups := s.operandGroups3D(involvedPoints3D(c, owners))
	ats := make([]math.Point3, 0, len(groups))
	for _, g := range groups {
		ats = append(ats, centroid3D(g))
	}
	return dedupeAnchors3D(ats)
}

// involvedPoints3D lists the distinct 3D points a constraint's variables live in, in the order the
// constraint names them.
func involvedPoints3D(c Constraint, owners map[*math.Scalar]*Point3D) []*Point3D {
	seen := map[*Point3D]bool{}
	var pts []*Point3D
	for _, v := range c.Variables() {
		p, known := owners[v]
		if !known || seen[p] {
			continue
		}
		seen[p] = true
		pts = append(pts, p)
	}
	return pts
}

// operandGroups3D splits a constraint's points into one group per operand. An entity claims a group
// when the constraint moves enough of its points to identify it — two, or one for a circular curve
// reached through its centre. The points no entity claims (standalone points, a curve endpoint a
// relation only grazes) fall into ONE group: a relation between bare points marks their meeting
// place once, not each point.
func (s *Sketch3D) operandGroups3D(involved []*Point3D) [][]*Point3D {
	claimed := map[*Point3D]bool{}
	var groups [][]*Point3D
	for _, e := range s.Entities() {
		owned := entityPoints3DAmong(e, involved)
		if !claimsOperand3D(e, owned) {
			continue
		}
		for _, p := range owned {
			claimed[p] = true
		}
		groups = append(groups, owned)
	}
	var loose []*Point3D
	for _, p := range involved {
		if !claimed[p] {
			loose = append(loose, p)
		}
	}
	if len(loose) > 0 {
		groups = append(groups, loose)
	}
	return groups
}

// claimsOperand3D reports whether an entity is one of the constraint's operands. Two of its points
// identify it; one does not, because a shared endpoint belongs to every line meeting there. A
// circular curve is the exception: the constraint reaches it through its centre alone.
func claimsOperand3D(e Entity, owned []*Point3D) bool {
	if len(owned) >= 2 {
		return true
	}
	return len(owned) == 1 && isCircular3D(e)
}

// isCircular3D reports whether a 3D entity is a circular curve reached through its centre.
func isCircular3D(e Entity) bool {
	switch e.(type) {
	case *Circle3D, *Arc3D, *Ellipse3D:
		return true
	}
	return false
}

// entityPoints3DAmong returns the entity's own defining points that appear in involved.
func entityPoints3DAmong(e Entity, involved []*Point3D) []*Point3D {
	mine := map[*Point3D]bool{}
	for _, p := range entityDefiningPoints3D(e) {
		mine[p] = true
	}
	var out []*Point3D
	for _, p := range involved {
		if mine[p] {
			out = append(out, p)
		}
	}
	return out
}

// entityDefiningPoints3D returns the constrainable points a 3D entity owns — the 3D counterpart of
// entityDefiningPoints (blocks_create.go). A new 3D entity kind that carries points must be added
// here to be marked; TestConstraintGlyphs3DCoversEveryEntityKind guards the omission.
func entityDefiningPoints3D(e Entity) []*Point3D {
	switch t := e.(type) {
	case *Point3D:
		return []*Point3D{t}
	case *Line3D:
		return []*Point3D{t.A, t.B}
	case *Circle3D:
		return []*Point3D{t.Center}
	case *Arc3D:
		return []*Point3D{t.Center, t.Start, t.End}
	case *Ellipse3D:
		return []*Point3D{t.Center}
	case *Spline3D:
		return append([]*Point3D(nil), t.Points...)
	}
	return nil
}

// centroid3D averages a group's points — where its marker sits.
func centroid3D(pts []*Point3D) math.Point3 {
	var sum math.Vector3
	for _, p := range pts {
		sum = sum.Add(math.V3(p.X, p.Y, p.Z))
	}
	return math.P3(0, 0, 0).TranslateBy(sum.Scale(1 / float64(len(pts))))
}

// dedupeAnchors3D collapses anchors that settled on the same place, so a relation whose operands
// coincide draws one marker rather than two stacked on the contact point.
func dedupeAnchors3D(ats []math.Point3) []math.Point3 {
	var out []math.Point3
	for _, at := range ats {
		if !containsAnchor3D(out, at) {
			out = append(out, at)
		}
	}
	return out
}

// containsAnchor3D reports whether at is already present, within a tolerance relative to how far
// from the origin the sketch works — an absolute epsilon would merge nothing on a micron-scale
// sketch and everything on a kilometre-scale one. It shares anchorEpsilon with the 2D path.
func containsAnchor3D(ats []math.Point3, at math.Point3) bool {
	for _, other := range ats {
		if float64(other.DistanceTo(at)) <= anchorEpsilon*(1+float64(at.DistanceTo(math.P3(0, 0, 0)))) {
			return true
		}
	}
	return false
}
