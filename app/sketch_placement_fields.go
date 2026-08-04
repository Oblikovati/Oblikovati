// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"
	stdmath "math"
	"strconv"

	"oblikovati.org/api/types"
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

// PlacementFieldView is one input box as the head draws it: its label, the value it shows
// (typed text, or the live cursor measurement), and the two positions its dotted witness line
// spans. Active is the box receiving keystrokes; Locked draws the padlock.
type PlacementFieldView struct {
	Label   string
	Value   string
	Unit    string
	Active  bool
	Locked  bool
	Witness [2]math.Point2
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

// PlacementFields returns the in-place input boxes for the shape being placed, labelled and
// positioned from the active tool's recipe. It is empty when no shape is in progress.
func (s *Session) PlacementFields() []PlacementFieldView {
	r, ok := s.ActiveToolRecipe(s.lastCursorSketchPoint)
	if !ok || !s.DimensionInputEnabled() {
		s.placementFields = placementFieldState{}
		return nil
	}
	s.syncPlacementFieldCount(len(r.Fields))
	views := make([]PlacementFieldView, len(r.Fields))
	for i, f := range r.Fields {
		views[i] = s.placementFieldView(i, f)
	}
	return views
}

// ensurePlacementFields makes the input strip match the shape currently being placed, so typing
// works whether or not the head has drawn a frame yet. Without it the strip would exist only as
// a side effect of painting, and a keystroke arriving first would be silently dropped.
func (s *Session) ensurePlacementFields() {
	r, ok := s.ActiveToolRecipe(s.lastCursorSketchPoint)
	if !ok || !s.DimensionInputEnabled() {
		return
	}
	s.syncPlacementFieldCount(len(r.Fields))
}

// syncPlacementFieldCount resizes the typing state when the tool's field count changes, which
// happens as a three-point shape moves from its base edge to its width.
func (s *Session) syncPlacementFieldCount(n int) {
	if len(s.placementFields.fields) == n {
		return
	}
	s.placementFields.fields = make([]placementField, n)
	s.placementFields.active = 0
}

// placementFieldView renders field i: its typed text when the user has entered one, otherwise
// the live measurement in the document's preferred unit.
func (s *Session) placementFieldView(i int, f sketch.RecipeField) PlacementFieldView {
	st := s.placementFields.fields[i]
	v := PlacementFieldView{
		Label: f.Label, Value: st.typed, Witness: f.Witness,
		Active: i == s.placementFields.active, Locked: st.locked,
		Unit: s.placementFieldUnitName(f.Unit),
	}
	if st.typed == "" {
		v.Value = formatHUDNumber(s.placementFieldLive(f))
	}
	return v
}

// placementFieldLive converts a field's model-unit value into the document's display unit.
func (s *Session) placementFieldLive(f sketch.RecipeField) float64 {
	if f.Unit == sketch.FieldAngle {
		return f.Value * 180 / stdmath.Pi
	}
	return s.DocumentUnits().ToPreferred(param.Q(f.Value, param.Length))
}

// placementFieldUnitName is the suffix shown after a field's value.
func (s *Session) placementFieldUnitName(u sketch.FieldUnit) string {
	if u == sketch.FieldAngle {
		return "deg"
	}
	return s.DocumentUnits().PreferredName(param.Length)
}

// PlacementFieldInput appends a typed character to the active field when it is part of a number.
// Other runes are ignored, so ordinary shortcuts still reach the viewport.
func (s *Session) PlacementFieldInput(r rune) {
	s.ensurePlacementFields()
	if !isHUDNumberRune(r) || len(s.placementFields.fields) == 0 {
		return
	}
	s.placementFields.engaged = true
	f := &s.placementFields.fields[s.placementFields.active]
	f.typed += string(r)
}

// PlacementFieldTab locks the active field and moves focus to the next — the padlock in the
// reference behaviour. A locked field freezes that quantity for the rest of the drag.
func (s *Session) PlacementFieldTab() {
	s.ensurePlacementFields()
	if len(s.placementFields.fields) == 0 {
		return
	}
	s.placementFields.engaged = true
	s.lockActivePlacementField()
	s.placementFields.active = (s.placementFields.active + 1) % len(s.placementFields.fields)
}

// lockActivePlacementField marks the active field locked when it holds a typed value.
func (s *Session) lockActivePlacementField() {
	f := &s.placementFields.fields[s.placementFields.active]
	if f.typed != "" {
		f.locked = true
	}
}

// PlacementFieldBackspace deletes the active field's last character and releases its lock, so a
// mistyped value can be corrected without cancelling the shape.
func (s *Session) PlacementFieldBackspace() {
	s.ensurePlacementFields()
	if len(s.placementFields.fields) == 0 {
		return
	}
	f := &s.placementFields.fields[s.placementFields.active]
	if f.typed == "" {
		return
	}
	f.typed, f.locked = f.typed[:len(f.typed)-1], false
}

// PlacementFieldCancel clears all typed state, returning every box to cursor tracking.
func (s *Session) PlacementFieldCancel() { s.placementFields = placementFieldState{} }

// PlacementFieldEngaged reports whether the user has begun typing into the strip, so the head
// can claim plain keystrokes before the viewport does.
func (s *Session) PlacementFieldEngaged() bool { return s.placementFields.engaged }

// PlacementFieldCommit locks the active field and finishes the shape at the cursor, which is
// what Enter does while placing.
func (s *Session) PlacementFieldCommit(px, py float64) error {
	s.ensurePlacementFields()
	if len(s.placementFields.fields) == 0 {
		return errors.New("placement fields: no shape is being placed")
	}
	s.lockActivePlacementField()
	if _, ok := s.CursorSketchPoint(px, py); !ok {
		return fmt.Errorf("placement fields: the cursor at (%v,%v) is not over the sketch plane", px, py)
	}
	s.sketchClick(px, py)
	return nil
}

// commitRecipe applies a tool's recipe with a dimension for every locked field, under the
// document's over-constrained behaviour, then clears the input strip for the next shape.
func (s *Session) commitRecipe(r sketch.Recipe) error {
	if s.activeSketch == nil {
		return errors.New("sketch: no active sketch")
	}
	_, _, err := s.activeSketch.ApplyWithFields(r, s.lockedFieldExpressions(r), s.overConstrainedBehavior())
	s.placementFields = placementFieldState{}
	return err
}

// lockedFieldExpressions renders each locked field as a unit-carrying parameter expression, and
// leaves untouched fields empty so they create no dimension.
func (s *Session) lockedFieldExpressions(r sketch.Recipe) []string {
	out := make([]string, len(r.Fields))
	if !s.CreateDimensionsOnValueInput() {
		return out // typed values still size the shape, but state nothing afterwards
	}
	for i, f := range r.Fields {
		if st := s.placementFieldAt(i); st.locked {
			out[i] = s.placementFieldExpression(st.typed, f.Unit)
		}
	}
	return out
}

// ShowConstraintsOnCreation reports whether the inference glyphs should be drawn beside the
// cursor while geometry is being placed — the active document's DisplayConstraintsOnCreation
// setting, which was declared and persisted but read by nothing before #2014.
func (s *Session) ShowConstraintsOnCreation() bool {
	settings, err := s.DocumentSketchSettings(0)
	if err != nil {
		return types.DefaultSketchSettings().DisplayConstraintsOnCreation
	}
	return settings.DisplayConstraintsOnCreation
}

// overConstrainedBehavior is the active document's preference for a dimension that would make
// the sketch redundant, defaulting to adding it as driven when no document is open.
func (s *Session) overConstrainedBehavior() types.OverConstrainedDimensionBehavior {
	settings, err := s.DocumentSketchSettings(0)
	if err != nil {
		return types.OverConstrainedApplyDriven
	}
	return settings.OverConstrainedBehavior
}

// placementFieldAt returns field i's typing state, or a zero field when out of range.
func (s *Session) placementFieldAt(i int) placementField {
	if i < 0 || i >= len(s.placementFields.fields) {
		return placementField{}
	}
	return s.placementFields.fields[i]
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
