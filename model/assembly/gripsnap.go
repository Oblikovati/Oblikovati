// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"fmt"

	"oblikovati.org/api/types"
)

// Grip snap (M12-F-grip, Oblikovati/Oblikovati#794): infer the assembly constraint that snaps one
// component's geometry onto another's, then let the existing solver position the part. The only new
// logic is the inference from the two inputs' primitive kinds — the constraint is built with the
// same factories the explicit Add* paths use, so it solves and reports identically.

// InferGripConstraint adds the constraint that snaps geometry a (on the component to move) onto target
// geometry b, at offset 0, and returns it with the chosen kind. prefer overrides the inference (0 ⇒
// auto). It does NOT solve — the caller runs the assembly solve so the part jumps into place.
func (s *ConstraintSet) InferGripConstraint(a, b Ref, prefer types.AssemblyConstraintType) (Constraint, types.AssemblyConstraintType, error) {
	kind := prefer
	if kind == 0 {
		inferred, err := inferSnapKind(a, b)
		if err != nil {
			return nil, 0, err
		}
		kind = inferred
	}
	c, err := s.addSnapConstraint(kind, a, b)
	if err != nil {
		return nil, 0, err
	}
	return c, kind, nil
}

// inferSnapKind picks the snap constraint from the two inputs' geometry kinds (and, for two planes,
// their current world orientation): planes already-aligned → flush, else mate; two cylinder axes →
// insert; an axis pair → mate (collinear); a plane + a cylinder → tangent; any point → coincident mate.
func inferSnapKind(a, b Ref) (types.AssemblyConstraintType, error) {
	ka, kb := a.Primitive.kind, b.Primitive.kind
	switch {
	case ka == pointKind || kb == pointKind:
		return types.ConstraintMate, nil // a point coincides (mate handles point inputs)
	case ka == planeKind && kb == planeKind:
		if planesAligned(a, b) {
			return types.ConstraintFlush, nil
		}
		return types.ConstraintMate, nil
	case ka == lineKind && kb == lineKind:
		if a.Primitive.radius > 0 && b.Primitive.radius > 0 {
			return types.ConstraintInsert, nil // two cylinders → bolt-in-hole
		}
		return types.ConstraintMate, nil // collinear axes
	case isCylinderPlanePair(a, b):
		return types.ConstraintTangent, nil
	default:
		return 0, fmt.Errorf("grip snap: cannot infer a constraint between %s and %s geometry", ka, kb)
	}
}

// addSnapConstraint builds the chosen snap constraint at offset 0, ordering tangent's plane/cylinder
// inputs as the factory expects.
func (s *ConstraintSet) addSnapConstraint(kind types.AssemblyConstraintType, a, b Ref) (Constraint, error) {
	switch kind {
	case types.ConstraintMate:
		return s.AddMate(a, b, 0, types.MateSolutionOpposed), nil
	case types.ConstraintFlush:
		return s.AddFlush(a, b, 0), nil
	case types.ConstraintInsert:
		return s.AddInsert(a, b, 0, false), nil
	case types.ConstraintTangent:
		if a.Primitive.radius > 0 { // tangent wants the plane first, the cylinder second
			a, b = b, a
		}
		return s.AddTangent(a, b, false), nil
	default:
		return nil, fmt.Errorf("grip snap: %s is not a snap constraint (want mate, flush, insert, or tangent)", kind)
	}
}

// planesAligned reports whether the two planar inputs' normals currently point the same way in world
// space — already roughly aligned → a flush snap, opposed → a mate.
func planesAligned(a, b Ref) bool {
	na := worldDir(a.Occurrence.Transform(), a.Primitive)
	nb := worldDir(b.Occurrence.Transform(), b.Primitive)
	return na.Dot(nb) > 0
}

// isCylinderPlanePair reports whether one input is a plane and the other a cylinder (an axis with a
// radius) — the tangent case.
func isCylinderPlanePair(a, b Ref) bool {
	cyl := func(p Primitive) bool { return p.kind == lineKind && p.radius > 0 }
	return (a.Primitive.kind == planeKind && cyl(b.Primitive)) || (cyl(a.Primitive) && b.Primitive.kind == planeKind)
}

// String renders a primitive kind for diagnostics.
func (k primKind) String() string {
	switch k {
	case pointKind:
		return "point"
	case lineKind:
		return "axis"
	case planeKind:
		return "plane"
	default:
		return "unknown"
	}
}
