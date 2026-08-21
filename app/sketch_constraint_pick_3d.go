// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/sketch"

// Picking a 3D-sketch constraint marker (#1998 follow-up). A 3D marker sits at a model-space point,
// so it is picked the way a standalone 3D point is: by the pick ray's distance of closest approach
// (rayPointDistance), within the same screen-space radius the marker is drawn at. Selecting one puts
// a SketchConstraintHandle in the selection — the same handle the 2D path uses — so Delete removes
// it through the shared selected-constraints path.

// PickSketchConstraint3DAt returns the 3D-sketch constraint whose marker the pick ray through
// (px, py) passes nearest, within the marker's screen radius, or false. Later markers win ties, so
// the most recently placed one is grabbed first — the topmost-first rule the 2D pick uses.
func (s *Session) PickSketchConstraint3DAt(px, py float64) (sketch.Constraint, bool) {
	if s.activeSketch3D == nil {
		return nil, false
	}
	origin, dir := s.camera.RayThrough(px, py)
	tol := constraintGlyphPixels * s.camera.WorldPerPixel()
	var best sketch.Constraint
	bestD := tol
	for _, g := range s.SketchConstraintGlyphs3D() {
		if d, _, ok := rayPointDistance(origin, dir, g.At); ok && d <= bestD {
			best, bestD = g.Constraint, d
		}
	}
	return best, best != nil
}

// SelectSketchConstraint3DAt selects the 3D constraint marker under (px, py), extending the
// selection when mods carry Shift/Ctrl, and reports whether one was hit. The input router tests it
// before the RayPicker, since a marker sits on top of the geometry it annotates.
func (s *Session) SelectSketchConstraint3DAt(px, py float64, mods Modifier) bool {
	c, ok := s.PickSketchConstraint3DAt(px, py)
	if !ok {
		return false
	}
	s.applyPickToSelection(SketchConstraintHandle{Constraint: c}, mods)
	return true
}
