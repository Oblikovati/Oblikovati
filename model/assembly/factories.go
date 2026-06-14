// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/model/occurrence"
)

// The classic-constraint factories (M12-F01). Each takes its geometry inputs as an
// occurrence plus a definition-space [Primitive], mints the constraint, adds it to the
// set, and returns the typed constraint. None solve: the caller runs [ConstraintSet.Solve]
// once after authoring (so a batch of adds solves once).

// AddMate adds a mate making geometry A coincident with geometry B at offset, with the
// given solution sense (opposed/aligned).
func (s *ConstraintSet) AddMate(occA *occurrence.Occurrence, primA Primitive, occB *occurrence.Occurrence, primB Primitive, offset float64, sol types.MateConstraintSolutionType) *MateConstraint {
	m := &MateConstraint{constraintBase: s.newBase(types.ConstraintMate, anchor{occA, primA}, anchor{occB, primB}), offset: offset, solution: sol}
	s.add(m)
	return m
}

// AddFlush adds a flush making faces A and B co-planar (aligned normals) at offset.
func (s *ConstraintSet) AddFlush(occA *occurrence.Occurrence, primA Primitive, occB *occurrence.Occurrence, primB Primitive, offset float64) *FlushConstraint {
	f := &FlushConstraint{constraintBase: s.newBase(types.ConstraintFlush, anchor{occA, primA}, anchor{occB, primB}), offset: offset}
	s.add(f)
	return f
}

// AddAngle adds an angle constraint holding angle (radians) between directions A and B.
func (s *ConstraintSet) AddAngle(occA *occurrence.Occurrence, primA Primitive, occB *occurrence.Occurrence, primB Primitive, angle float64, sol types.AngleConstraintSolutionType) *AngleConstraint {
	c := &AngleConstraint{constraintBase: s.newBase(types.ConstraintAngle, anchor{occA, primA}, anchor{occB, primB}), angle: angle, solution: sol}
	s.add(c)
	return c
}

// AddTangent adds a tangent constraint keeping plane A tangent to cylinder B.
func (s *ConstraintSet) AddTangent(occA *occurrence.Occurrence, primA Primitive, occB *occurrence.Occurrence, primB Primitive, inside bool) *TangentConstraint {
	c := &TangentConstraint{constraintBase: s.newBase(types.ConstraintTangent, anchor{occA, primA}, anchor{occB, primB}), inside: inside}
	s.add(c)
	return c
}

// AddInsert adds an insert (collinear axes plus an axial offset) of A into B.
func (s *ConstraintSet) AddInsert(occA *occurrence.Occurrence, primA Primitive, occB *occurrence.Occurrence, primB Primitive, offset float64, aligned bool) *InsertConstraint {
	c := &InsertConstraint{constraintBase: s.newBase(types.ConstraintInsert, anchor{occA, primA}, anchor{occB, primB}), offset: offset, aligned: aligned}
	s.add(c)
	return c
}

// AddSymmetry adds a symmetry constraint positioning A and B symmetrically about the
// plane (a third anchor on its own occurrence).
func (s *ConstraintSet) AddSymmetry(occA *occurrence.Occurrence, primA Primitive, occB *occurrence.Occurrence, primB Primitive, occPlane *occurrence.Occurrence, primPlane Primitive) *AssemblySymmetryConstraint {
	c := &AssemblySymmetryConstraint{
		constraintBase: s.newBase(types.ConstraintSymmetry, anchor{occA, primA}, anchor{occB, primB}),
		plane:          anchor{occPlane, primPlane},
	}
	s.add(c)
	return c
}

// constraintNames maps each kind to its display prefix (e.g. "Mate" → "Mate:1").
var constraintNames = map[types.AssemblyConstraintType]string{
	types.ConstraintMate:               "Mate",
	types.ConstraintFlush:              "Flush",
	types.ConstraintAngle:              "Angle",
	types.ConstraintTangent:            "Tangent",
	types.ConstraintInsert:             "Insert",
	types.ConstraintSymmetry:           "Symmetry",
	types.ConstraintRotateRotate:       "RotateRotate",
	types.ConstraintRotateTranslate:    "RotateTranslate",
	types.ConstraintTranslateTranslate: "TranslateTranslate",
	types.ConstraintTransitional:       "Transitional",
	types.ConstraintCustom:             "Custom",
}

// constraintName builds a unique display name for the n-th constraint of a kind.
func constraintName(kind types.AssemblyConstraintType, n int) string {
	prefix, ok := constraintNames[kind]
	if !ok {
		prefix = "Constraint"
	}
	return fmt.Sprintf("%s:%d", prefix, n)
}
