// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/sketch"

// The Format panel's four toggles share one rule (#2015): with geometry selected the button
// CONVERTS the selection; with nothing selected it flips a CREATION MODE that applies to whatever
// is drawn next. One helper implements it for all four so they cannot drift apart.
//
// Before this, Construction and Centerline converted a selection and had no creation mode at all,
// which is half of what the panel is for — the issue's own wording ("changes selected sketch
// geometry … or creates new geometry as …") describes both halves.

// sketchFormatModes is the armed creation state: what newly drawn geometry becomes.
type sketchFormatModes struct {
	construction bool // new geometry is construction
	centerline   bool // new lines are centerlines
	centerPoint  bool // the Point tool places centre points
	drivenDim    bool // new dimensions are driven
}

// ConstructionMode, CenterlineMode, CenterPointMode and DrivenDimensionMode report which creation
// modes are armed, so the ribbon can draw those buttons pressed.
func (s *Session) ConstructionMode() bool    { return s.formatModes.construction }
func (s *Session) CenterlineMode() bool      { return s.formatModes.centerline }
func (s *Session) CenterPointMode() bool     { return s.formatModes.centerPoint }
func (s *Session) DrivenDimensionMode() bool { return s.formatModes.drivenDim }

// ToggleConstruction converts the selected geometry to construction, or — with nothing selected —
// arms construction mode. It returns how many entities it converted, so 0 means it flipped the
// mode instead.
func (s *Session) ToggleConstruction() int {
	return s.convertOrArm(&s.formatModes.construction, flipConstruction)
}

// flipConstruction inverts one entity's construction flag, reporting whether it could.
func flipConstruction(e sketch.Entity) bool {
	c, ok := e.(interface {
		IsConstruction() bool
		SetConstruction(bool)
	})
	if !ok {
		return false
	}
	c.SetConstruction(!c.IsConstruction())
	return true
}

// ToggleCenterline converts the selected lines to centerlines (an axis for revolve, mirror and
// symmetry), or arms centerline mode.
func (s *Session) ToggleCenterline() int {
	return s.convertOrArm(&s.formatModes.centerline, flipCenterline)
}

// flipCenterline inverts one line's centerline flag, reporting whether the entity was a line.
func flipCenterline(e sketch.Entity) bool {
	l, ok := e.(*sketch.Line)
	if !ok {
		return false
	}
	l.SetCenterline(!l.IsCenterline())
	return true
}

// ToggleCenterPoint converts the selected points to hole-centre markers, or arms centre-point
// mode for the Point tool.
func (s *Session) ToggleCenterPoint() int {
	return s.convertOrArm(&s.formatModes.centerPoint, flipCenterPoint)
}

// flipCenterPoint inverts one point's centre-point marker, reporting whether it was a point.
func flipCenterPoint(e sketch.Entity) bool {
	p, ok := e.(*sketch.Point)
	if !ok {
		return false
	}
	p.SetCenterPoint(!p.IsCenterPoint())
	return true
}

// ToggleDrivenDimension flips the selected dimensions between driving and driven, or — with none
// selected — arms driven mode for dimensions created next.
func (s *Session) ToggleDrivenDimension() int {
	n := 0
	for _, d := range s.selectedDimensions() {
		d.SetDriven(!d.Driven())
		n++
	}
	if n == 0 {
		s.formatModes.drivenDim = !s.formatModes.drivenDim
	}
	return n
}

// convertOrArm applies convert to every selected sketch entity it accepts; when the selection
// yields none, it flips the armed mode instead. This is the dual rule, in one place.
func (s *Session) convertOrArm(mode *bool, convert func(sketch.Entity) bool) int {
	n := 0
	for _, e := range s.selectedSketchEntities() {
		if convert(e) {
			n++
		}
	}
	if n == 0 {
		*mode = !*mode
	}
	return n
}

// selectedDimensions returns the sketch dimensions in the current selection.
func (s *Session) selectedDimensions() []*sketch.DimensionConstraint {
	var out []*sketch.DimensionConstraint
	for _, it := range s.Selection().Items() {
		if h, ok := it.(SketchDimensionHandle); ok {
			out = append(out, h.Dim)
		}
	}
	return out
}

// applyFormatModes marks freshly created geometry per the armed creation modes.
//
// Most 2D tools funnel through commitRecipe and are covered by its single call. Two do not and
// call this directly: the line tool, which builds a chain and runs constraint inference on each
// segment, and the point tool, which adds points without a recipe. Every path that creates
// geometry must land here, or a mode would be honoured by some tools and quietly ignored by
// others — which is exactly the inconsistency this panel is meant to remove.
func (s *Session) applyFormatModes(ents []sketch.Entity) {
	for _, e := range ents {
		s.applyFormatModesTo(e)
	}
}

// applyFormatModesTo marks one entity per the armed modes. A recipe's own construction geometry
// (a centre rectangle's diagonals) is already flagged, so re-flagging it is a harmless no-op.
func (s *Session) applyFormatModesTo(e sketch.Entity) {
	if s.formatModes.construction {
		if c, ok := e.(interface{ SetConstruction(bool) }); ok {
			c.SetConstruction(true)
		}
	}
	if l, ok := e.(*sketch.Line); ok && s.formatModes.centerline {
		l.SetCenterline(true)
	}
	if p, ok := e.(*sketch.Point); ok && s.formatModes.centerPoint {
		p.SetCenterPoint(true)
	}
}

// applyDrivenDimensionMode makes the dimensions added by this commit driven when the mode is
// armed. before is the dimension count taken before the geometry was applied, so only the new
// ones are touched. It runs after the recipe's own over-constrained handling, which may already
// have demoted a redundant dimension — both set the same flag, so the two compose.
func (s *Session) applyDrivenDimensionMode(before int) {
	if !s.formatModes.drivenDim || s.activeSketch == nil {
		return
	}
	dims := s.activeSketch.DimensionConstraints().All()
	for i := before; i < len(dims); i++ {
		dims[i].SetDriven(true)
	}
}

// dimensionCount is the active sketch's dimension count, or 0 with no sketch — the "before"
// mark applyDrivenDimensionMode measures against.
func (s *Session) dimensionCount() int {
	if s.activeSketch == nil {
		return 0
	}
	return len(s.activeSketch.DimensionConstraints().All())
}

// ShowFormat reports whether formatting overrides are being suppressed.
//
// The name follows the button's label, but the behaviour is the documented one and is the inverse
// of what the label suggests: ON draws the sketch with DEFAULT attributes, hiding per-entity line
// type, colour and thickness; OFF shows the user's formatting again. The stored field is named
// for what it does so the code does not read backwards.
func (s *Session) ShowFormat() bool { return s.appOptions.Sketch.SuppressFormatOverrides }

// ToggleShowFormat flips the suppression and persists it.
func (s *Session) ToggleShowFormat() {
	s.appOptions.Sketch.SuppressFormatOverrides = !s.appOptions.Sketch.SuppressFormatOverrides
	_ = s.saveOptions()
}
