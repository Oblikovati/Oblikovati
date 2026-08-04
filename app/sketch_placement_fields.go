// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"strconv"

	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// In-place dimension input (#2014): while a shape is being placed, each of its dimensionable
// quantities gets an input box on a dotted witness line. Typing a value and pressing Tab locks
// it — the drag then changes only the remaining quantities, and on commit the locked value
// becomes a real driving dimension. A field left tracking the cursor creates nothing.
//
// This file holds the locked-value plumbing the preview needs; the typing state machine and the
// dimension creation build on it.

// placementField is one input box's typing state.
type placementField struct {
	typed  string // "" ⇒ the field tracks the cursor
	locked bool   // typed and committed with Tab/Enter ⇒ becomes a dimension
}

// placementFieldState is the whole in-place input strip for the shape being placed.
type placementFieldState struct {
	fields  []placementField
	active  int
	engaged bool
}

// placementFieldValues returns each field's typed override ("" ⇒ tracks the cursor), which the
// recipe builders use to freeze locked quantities during the drag.
func (s *Session) placementFieldValues() []string {
	out := make([]string, len(s.placementFields.fields))
	for i, f := range s.placementFields.fields {
		if f.locked {
			out[i] = f.typed
		}
	}
	return out
}

// lockedCorner replaces the cursor's offsets from the anchor with any locked Width and Height,
// keeping the cursor's sign so the shape still flips through the anchor as the drag crosses it.
func (s *Session) lockedCorner(anchor, cursor math.Point2, locked []string) math.Point2 {
	x, y := cursor.X, cursor.Y
	if w, ok := s.lockedLength(locked, 0); ok {
		x = anchor.X + math.Scalar(stdmath.Copysign(w, float64(cursor.X-anchor.X)))
	}
	if h, ok := s.lockedLength(locked, 1); ok {
		y = anchor.Y + math.Scalar(stdmath.Copysign(h, float64(cursor.Y-anchor.Y)))
	}
	return math.P2(x, y)
}

// lockedLength parses locked field i as a model-unit length, or reports false when the field is
// untouched or unparseable (a half-typed "1." must not snap the preview).
func (s *Session) lockedLength(locked []string, i int) (float64, bool) {
	if i >= len(locked) || locked[i] == "" {
		return 0, false
	}
	return s.placementFieldModelValue(locked[i], sketch.FieldLength)
}

// placementFieldModelValue parses a typed field into model units (or radians).
func (s *Session) placementFieldModelValue(typed string, u sketch.FieldUnit) (float64, bool) {
	v, err := strconv.ParseFloat(typed, 64)
	if err != nil {
		return 0, false
	}
	if u == sketch.FieldAngle {
		return v * stdmath.Pi / 180, true
	}
	return s.DocumentUnits().FromPreferred(v, param.Length).Value, true
}

// placementFieldExpression renders a typed value as a parameter expression carrying its unit.
// The unit is always explicit because the parameter engine is unit-strict: a bare "10" would
// silently mean 10 cm, the kernel's length unit, rather than 10 of the document's unit.
func (s *Session) placementFieldExpression(typed string, u sketch.FieldUnit) string {
	if u == sketch.FieldAngle {
		return typed + " deg"
	}
	return typed + " " + s.DocumentUnits().PreferredName(param.Length)
}
