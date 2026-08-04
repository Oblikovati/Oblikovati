// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Dragging a sketch dimension's label to reposition it (#2017), the 2D-sketch counterpart of the
// drawing canvas's dimension drag (Session.DragDimension, M14-F03). The placement is stored on the
// dimension as its TextPoint — the field #1875 added and persisted but which nothing ever read,
// leaving dimensions pinned to a derived position no user could change.
//
// Unlike an entity drag this never touches the solver: moving a dimension's annotation must not
// move the geometry it measures, so no re-solve is triggered and the sketch's DOF is untouched.

// dimensionDrag is an in-progress drag of one dimension's label. start is the label's plane
// position when the drag began and grab the cursor there, so each frame applies an absolute
// delta and the label cannot drift from accumulated rounding.
type dimensionDrag struct {
	dim    *sketch.DimensionConstraint
	start  math.Point2
	grab   math.Point2
	moved  bool
	active bool
}

// DimensionDragActive reports whether a sketch dimension's label is currently being dragged.
func (s *Session) DimensionDragActive() bool { return s.dimensionDrag.active }

// BeginDimensionDrag arms a drag of the dimension label under the viewport point (px, py), also
// selecting it so the drag and a plain click agree on what is selected. It returns false — so the
// caller falls through to entity drag and box-select — when no label is there.
func (s *Session) BeginDimensionDrag(px, py float64, mods Modifier) bool {
	if s.activeSketch == nil || s.tool != nil {
		return false
	}
	d, ok := s.PickSketchDimensionAt(px, py)
	if !ok {
		return false
	}
	grab, ok := screenToSketch(s, px, py)
	if !ok {
		return false
	}
	s.applyPickToSelection(SketchDimensionHandle{Dim: d}, mods)
	s.dimensionDrag = dimensionDrag{dim: d, start: dimensionLabelAt(s, d), grab: grab, active: true}
	return true
}

// dimensionLabelAt returns where d's label is currently drawn, which is the drag's origin. A
// dimension with no stored TextPoint is being moved off its derived default for the first time,
// so the derived anchor is read back out of the view rather than assumed.
func dimensionLabelAt(s *Session, d *sketch.DimensionConstraint) math.Point2 {
	if tp, ok := d.TextPoint(); ok {
		return tp
	}
	for _, v := range s.SketchDimensions() {
		if v.Dim == d {
			return v.LabelAt
		}
	}
	return math.Point2{}
}

// UpdateDimensionDrag moves the dragged label to its start position plus the cursor delta. The
// overlay rebuilds from the live sketch each frame, so the label tracks the cursor immediately.
func (s *Session) UpdateDimensionDrag(px, py float64) {
	if !s.dimensionDrag.active {
		return
	}
	cur, ok := screenToSketch(s, px, py)
	if !ok {
		return
	}
	delta := s.dimensionDrag.grab.VectorTo(cur)
	s.dimensionDrag.dim.SetTextPoint(s.dimensionDrag.start.TranslateBy(delta))
	s.dimensionDrag.moved = true
}

// CommitDimensionDrag ends the drag, recording one undoable edit when the label actually moved.
// A press-and-release that never moved is a plain click (it selected the dimension), so it must
// not push an empty step onto the undo stack.
func (s *Session) CommitDimensionDrag() {
	moved := s.dimensionDrag.active && s.dimensionDrag.moved
	s.dimensionDrag = dimensionDrag{}
	if moved {
		s.RecordActiveEdit("Move Dimension")
	}
}

// CancelDimensionDrag abandons the drag and restores the label to where it started, so Escape
// undoes an in-flight move rather than leaving it half-applied.
func (s *Session) CancelDimensionDrag() {
	if s.dimensionDrag.active && s.dimensionDrag.moved {
		s.dimensionDrag.dim.SetTextPoint(s.dimensionDrag.start)
	}
	s.dimensionDrag = dimensionDrag{}
}
