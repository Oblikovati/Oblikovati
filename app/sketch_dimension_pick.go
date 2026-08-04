// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Picking a sketch dimension by its value label (#2017). Until now a sketch dimension was a
// read-only annotation of the solve: it could be double-clicked to edit its value, but nothing
// could select it, so it could not be deleted or moved either — unlike a drawing dimension, which
// has been pickable and draggable on the sheet since M14-F03. The pick lives here rather than in
// the head so it stays headless and testable, mirroring [Session.pickSketchEntity].

// dimLabelPixels is the screen-space radius within which a click grabs a dimension's label. It is
// wider than snapPixels because the target is a text box, not a point.
const dimLabelPixels = 18

// PickSketchDimensionAt returns the dimension whose value label lies nearest the viewport point
// (px, py), within the label pick radius, or false. Later dimensions win ties so the most
// recently added one is grabbed first when labels overlap — the same topmost-first rule
// [Session.PickDrawingDimensionAt] applies on a sheet.
//
// Example: s.PickSketchDimensionAt(cursorX, cursorY) to select or begin dragging a dimension.
func (s *Session) PickSketchDimensionAt(px, py float64) (*sketch.DimensionConstraint, bool) {
	if s.activeSketch == nil {
		return nil, false
	}
	p, ok := screenToSketch(s, px, py)
	if !ok {
		return nil, false
	}
	return nearestDimensionLabel(s.SketchDimensions(), p, dimLabelPixels*s.camera.WorldPerPixel())
}

// nearestDimensionLabel returns the view whose label anchor is closest to p within tol. It scans
// forward with a `<=` test so a later dimension displaces an equidistant earlier one.
func nearestDimensionLabel(views []DimensionView, p math.Point2, tol float64) (*sketch.DimensionConstraint, bool) {
	var best *sketch.DimensionConstraint
	bestD := tol
	for _, v := range views {
		if d := p.DistanceTo(v.LabelAt); d <= bestD {
			best, bestD = v.Dim, d
		}
	}
	return best, best != nil
}

// SelectSketchDimensionAt selects the dimension under (px, py), extending the selection when mods
// carry Shift/Ctrl, and reports whether one was hit. The head calls it before its entity-pick so a
// label lying over geometry selects the dimension rather than the curve beneath it.
func (s *Session) SelectSketchDimensionAt(px, py float64, mods Modifier) bool {
	d, ok := s.PickSketchDimensionAt(px, py)
	if !ok {
		return false
	}
	s.applyPickToSelection(SketchDimensionHandle{Dim: d}, mods)
	return true
}

// SelectedSketchDimensions returns the dimensions currently in the selection set — the input the
// Delete action and the Format panel's driven toggle consume.
func (s *Session) SelectedSketchDimensions() []*sketch.DimensionConstraint {
	var out []*sketch.DimensionConstraint
	for _, it := range s.selection.Items() {
		if h, ok := it.(SketchDimensionHandle); ok {
			out = append(out, h.Dim)
		}
	}
	return out
}
