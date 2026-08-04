// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Show / Hide Constraints (Inventor's F8 / F9): while a 2D sketch is open, every geometric
// constraint it carries draws a small marker on the geometry it relates, and a marker can be
// clicked and deleted.
//
// Until now a placed constraint was invisible and unremovable from the viewport: the solver
// honoured it, the browser never listed it, and nothing could select one — so an accidental
// coincidence or a wrong auto-inferred horizontal could only be undone, never repaired. The pick
// and delete paths mirror the sketch dimension's (#2017), which had exactly this gap and closed
// it the same way.

// constraintGlyphPixels is the screen-space radius within which a click grabs a marker, and the
// spacing used to fan out markers that share an anchor. It matches the dimension label's pick
// radius so the two annotation layers feel the same under the cursor.
const constraintGlyphPixels = 14

// ConstraintGlyphView is one constraint marker positioned for drawing: what it relates, the glyph
// to draw, where it sits after fanning out co-located markers, and whether it is selected.
type ConstraintGlyphView struct {
	Constraint sketch.Constraint
	Kind       sketch.ConstraintKind
	At         math.Point2
	Selected   bool
}

// ShowSketchConstraints reports whether constraint markers are drawn in the sketch.
func (s *Session) ShowSketchConstraints() bool { return s.showSketchConstraints }

// SetShowSketchConstraints turns constraint markers on or off (Show All / Hide All Constraints).
// Hiding them also drops any selected constraint, so a marker cannot stay selected — and be
// deleted by a later Delete — while invisible.
func (s *Session) SetShowSketchConstraints(show bool) {
	s.showSketchConstraints = show
	if !show {
		s.deselectSketchConstraints()
	}
}

// ToggleSketchConstraints flips constraint-marker visibility — the ribbon button and F8.
func (s *Session) ToggleSketchConstraints() { s.SetShowSketchConstraints(!s.showSketchConstraints) }

// SketchConstraintGlyphs returns the markers to draw for the active sketch, empty when constraints
// are hidden or no sketch is open. Markers sharing an anchor — the four coincidences at a
// rectangle's corner, say — are fanned into a row so each stays separately clickable.
//
//	for _, g := range s.SketchConstraintGlyphs() { drawGlyph(g.Kind, g.At, g.Selected) }
func (s *Session) SketchConstraintGlyphs() []ConstraintGlyphView {
	if !s.showSketchConstraints || s.activeSketch == nil {
		return nil
	}
	selected := s.selectedConstraintSet()
	step := constraintGlyphPixels * s.camera.WorldPerPixel()
	views := make([]ConstraintGlyphView, 0, len(s.activeSketch.ConstraintGlyphs()))
	shared := map[math.Point2]int{}
	for _, g := range s.activeSketch.ConstraintGlyphs() {
		views = append(views, ConstraintGlyphView{
			Constraint: g.Constraint, Kind: g.Kind, Selected: selected[g.Constraint],
			At: fannedGlyphAnchor(g.At, shared[g.At], step),
		})
		shared[g.At]++
	}
	return views
}

// fannedGlyphAnchor offsets the nth marker sharing an anchor, so co-located markers sit in a row
// beside the geometry rather than on top of one another.
//
// The offset is DIAGONAL, not straight up: a vertical tick drawn on a vertical line, or a
// horizontal dash on a horizontal line, is invisible — it disappears into the very edge it
// annotates, which is exactly where axis-aligned constraints land.
func fannedGlyphAnchor(at math.Point2, n int, step float64) math.Point2 {
	return math.P2(at.X+math.Scalar(float64(n+1)*step), at.Y+math.Scalar(step))
}

// selectedConstraintSet indexes the selection's constraints for the per-glyph Selected flag.
func (s *Session) selectedConstraintSet() map[sketch.Constraint]bool {
	set := map[sketch.Constraint]bool{}
	for _, c := range s.SelectedSketchConstraints() {
		set[c] = true
	}
	return set
}

// PickSketchConstraintAt returns the constraint whose marker lies nearest the viewport point
// (px, py), within the marker pick radius, or false. Later constraints win ties so the most
// recently placed marker is grabbed first — the same topmost-first rule the dimension pick uses.
func (s *Session) PickSketchConstraintAt(px, py float64) (sketch.Constraint, bool) {
	p, ok := screenToSketch(s, px, py)
	if !ok {
		return nil, false
	}
	return nearestConstraintGlyph(s.SketchConstraintGlyphs(), p, constraintGlyphPixels*s.camera.WorldPerPixel())
}

// nearestConstraintGlyph returns the marker closest to p within tol.
func nearestConstraintGlyph(views []ConstraintGlyphView, p math.Point2, tol float64) (sketch.Constraint, bool) {
	var best sketch.Constraint
	bestD := tol
	for _, v := range views {
		if d := p.DistanceTo(v.At); d <= bestD {
			best, bestD = v.Constraint, d
		}
	}
	return best, best != nil
}

// SelectSketchConstraintAt selects the constraint marker under (px, py), extending the selection
// when mods carry Shift/Ctrl, and reports whether one was hit. The head calls it before its
// dimension and entity picks, since a marker sits on top of the geometry it annotates.
func (s *Session) SelectSketchConstraintAt(px, py float64, mods Modifier) bool {
	c, ok := s.PickSketchConstraintAt(px, py)
	if !ok {
		return false
	}
	s.applyPickToSelection(SketchConstraintHandle{Constraint: c}, mods)
	return true
}

// SelectedSketchConstraints returns the constraints currently in the selection set — the input
// the Delete action consumes.
func (s *Session) SelectedSketchConstraints() []sketch.Constraint {
	var out []sketch.Constraint
	for _, it := range s.selection.Items() {
		if h, ok := it.(SketchConstraintHandle); ok {
			out = append(out, h.Constraint)
		}
	}
	return out
}

// deselectSketchConstraints drops every constraint from the selection, leaving other kinds alone.
func (s *Session) deselectSketchConstraints() {
	for _, c := range s.SelectedSketchConstraints() {
		s.selection.Remove(SketchConstraintHandle{Constraint: c})
	}
}
