// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"fmt"

	"oblikovati.org/api/types"
)

// The constraint factories (M12-F01). Each takes its geometry inputs as [Ref]s (an
// occurrence + a definition-space [Primitive] + the reference key it came from), mints the
// constraint, adds it to the set, and returns the typed constraint. None solve: the caller
// runs [ConstraintSet.Solve] once after authoring (so a batch of adds solves once).

// AddMate adds a mate making geometry A coincident with geometry B at offset, with the
// given solution sense (opposed/aligned).
func (s *ConstraintSet) AddMate(a, b Ref, offset float64, sol types.MateConstraintSolutionType) *MateConstraint {
	m := &MateConstraint{constraintBase: s.newBase(types.ConstraintMate, a, b), offset: offset, solution: sol}
	s.add(m)
	return m
}

// AddFlush adds a flush making faces A and B co-planar (aligned normals) at offset.
func (s *ConstraintSet) AddFlush(a, b Ref, offset float64) *FlushConstraint {
	f := &FlushConstraint{constraintBase: s.newBase(types.ConstraintFlush, a, b), offset: offset}
	s.add(f)
	return f
}

// AddAngle adds an angle constraint holding angle (radians) between directions A and B.
func (s *ConstraintSet) AddAngle(a, b Ref, angle float64, sol types.AngleConstraintSolutionType) *AngleConstraint {
	c := &AngleConstraint{constraintBase: s.newBase(types.ConstraintAngle, a, b), angle: angle, solution: sol}
	s.add(c)
	return c
}

// AddTangent adds a tangent constraint keeping plane A tangent to cylinder B.
func (s *ConstraintSet) AddTangent(a, b Ref, inside bool) *TangentConstraint {
	c := &TangentConstraint{constraintBase: s.newBase(types.ConstraintTangent, a, b), inside: inside}
	s.add(c)
	return c
}

// AddInsert adds an insert (collinear axes plus an axial offset) of A into B.
func (s *ConstraintSet) AddInsert(a, b Ref, offset float64, aligned bool) *InsertConstraint {
	c := &InsertConstraint{constraintBase: s.newBase(types.ConstraintInsert, a, b), offset: offset, aligned: aligned}
	s.add(c)
	return c
}

// AddSymmetry adds a symmetry constraint positioning A and B symmetrically about plane.
func (s *ConstraintSet) AddSymmetry(a, b, plane Ref) *AssemblySymmetryConstraint {
	c := &AssemblySymmetryConstraint{
		constraintBase: s.newBase(types.ConstraintSymmetry, a, b),
		plane:          toAnchor(plane),
	}
	s.add(c)
	return c
}

// AddRotateRotate adds a gear-ratio coupling between rotation A and rotation B.
func (s *ConstraintSet) AddRotateRotate(a, b Ref, ratio float64) *RotateRotateConstraint {
	c := &RotateRotateConstraint{constraintBase: s.newBase(types.ConstraintRotateRotate, a, b), ratio: ratio}
	s.add(c)
	return c
}

// AddRotateTranslate adds a rack-and-pinion coupling between rotation A and translation B.
func (s *ConstraintSet) AddRotateTranslate(a, b Ref, distance float64) *RotateTranslateConstraint {
	c := &RotateTranslateConstraint{constraintBase: s.newBase(types.ConstraintRotateTranslate, a, b), distance: distance}
	s.add(c)
	return c
}

// AddTranslateTranslate adds a ratio coupling between translation A and translation B.
func (s *ConstraintSet) AddTranslateTranslate(a, b Ref, ratio float64) *TranslateTranslateConstraint {
	c := &TranslateTranslateConstraint{constraintBase: s.newBase(types.ConstraintTranslateTranslate, a, b), ratio: ratio}
	s.add(c)
	return c
}

// AddTransitional adds a sliding-contact constraint keeping geometry A on transition face B.
func (s *ConstraintSet) AddTransitional(a, b Ref) *TransitionalConstraint {
	c := &TransitionalConstraint{constraintBase: s.newBase(types.ConstraintTransitional, a, b)}
	s.add(c)
	return c
}

// AddCustom adds an add-in-solved relationship of the given kind, driven by params.
func (s *ConstraintSet) AddCustom(a, b Ref, kind string, params []float64) *CustomConstraint {
	c := &CustomConstraint{
		constraintBase: s.newBase(types.ConstraintCustom, a, b),
		kind:           kind,
		params:         append([]float64(nil), params...),
		set:            s,
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
