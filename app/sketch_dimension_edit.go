// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// Editing a dimension's value — Inventor's edit-on-place flow. When the Dimension tool
// places a dimension it is held as the session's "pending" dimension; the UI shows an
// edit box pre-filled with its current expression, and committing a value drives the
// geometry through the parameter DAG + solver. Double-clicking an existing dimension
// re-opens the same edit by making it pending again.

// PendingDimension returns the dimension awaiting a typed value (just placed or being
// re-edited), or nil. The head shows an edit popup while this is non-nil.
func (s *Session) PendingDimension() *sketch.DimensionConstraint { return s.pendingDim }

// PendingDimensionExpression returns the current expression text to pre-fill the edit
// box (e.g. "25 mm"), or "" when no dimension is pending.
func (s *Session) PendingDimensionExpression() string {
	if s.pendingDim == nil {
		return ""
	}
	return s.pendingDim.Parameter().Expression()
}

// BeginEditDimension re-opens a placed dimension for editing (the double-click path).
func (s *Session) BeginEditDimension(d *sketch.DimensionConstraint) {
	s.pendingDim = d
}

// CommitPendingDimension sets the pending dimension's value from a parseable expression
// (e.g. "30 mm", "width/2", "45 deg") and re-solves, so the geometry moves to satisfy
// it. It errors (and leaves the dimension pending) when there is none pending or the
// expression is invalid, so the UI can keep the edit box open to correct it.
func (s *Session) CommitPendingDimension(expression string) error {
	if s.pendingDim == nil {
		return errors.New("app: no dimension is being edited")
	}
	// A bare number typed here means the document's display unit (7 → 7 mm), not the raw
	// database unit — qualify before the raw setter, or every unitless value inflates 10× (#1783).
	p := s.pendingDim.Parameter()
	qualified := p.QualifyAuthored(expression, dimensionCategory(s.pendingDim.Kind()), s.DocumentUnits())
	if err := p.SetExpression(qualified); err != nil {
		return err
	}
	s.pendingDim = nil
	if s.activeSketch != nil {
		s.activeSketch.Solve()
	}
	s.RecordActiveEdit("Edit Dimension") // one undo step per dimension value change (#1270)
	return nil
}

// CancelPendingDimension dismisses the edit box, keeping the dimension at its current
// (measured) value — Inventor accepts the placed dimension when you cancel the edit.
func (s *Session) CancelPendingDimension() { s.pendingDim = nil }

// dimensionCategory is the unit category a dimension kind's value is authored in — Angle for the
// angular dimensions, Length for every distance/radius/arc-length kind — so a bare number is
// qualified with the right document display unit (#1783).
func dimensionCategory(k sketch.DimKind) param.Unit {
	switch k {
	case sketch.AngleDim, sketch.ThreePointAngleDim:
		return param.Angle
	default:
		return param.Length
	}
}
