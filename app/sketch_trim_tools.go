// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// The Trim/Extend/Split tools are single-pick sketch-entity tools: the user clicks a
// curve at a location, and the click's snap point (carried by PickSnap) is the pick the
// underlying model edit needs — TrimLine/SplitLine cut at that point, ExtendLine extends
// the nearer end. They commit on the pick (then deactivate) like the other modify tools.
// The model behavior is tested headlessly here; the head only forwards picks.

// curveEditPick captures the one entity+point pick the trim/extend/split tools share. It
// satisfies SketchEntityTool (PickSnap) and the auto-commit-on-pick capability.
type curveEditPick struct {
	ent   sketch.Entity
	point math.Point2
	has   bool
}

// PickSnap records the picked entity and the snapped click location on it.
func (c *curveEditPick) PickSnap(ent sketch.Entity, snap SnapResult) {
	c.ent, c.point, c.has = ent, snap.Point, true
}

func (c *curveEditPick) CanCommit() bool        { return c.has }
func (c *curveEditPick) AutoCommitOnPick() bool { return true }
func (c *curveEditPick) reset()                 { c.ent, c.has = nil, false }

// pickedLine resolves the pick to a line (the kinds trim/extend/split currently edit).
func (c *curveEditPick) pickedLine() (*sketch.Line, error) {
	if !c.has {
		return nil, errors.New("no curve picked")
	}
	l, ok := c.ent.(*sketch.Line)
	if !ok {
		return nil, errors.New("pick must be a line")
	}
	return l, nil
}

// SketchTrimTool removes the picked segment of a line up to its nearest crossings with
// other sketch geometry (lines, circles, arcs).
type SketchTrimTool struct {
	dialogTool
	curveEditPick
}

// NewSketchTrimTool makes a trim tool.
func NewSketchTrimTool() *SketchTrimTool { return &SketchTrimTool{} }

func (t *SketchTrimTool) Name() string           { return "Trim" }
func (t *SketchTrimTool) Cancel(*Session)        { t.reset() }
func (t *SketchTrimTool) Prompt(*Session) string { return "Pick the curve segment to trim." }

// Commit trims the picked curve (line, circle or arc) at the pick point.
func (t *SketchTrimTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("trim: no active sketch")
	}
	if !t.has {
		return errors.New("trim: no curve picked")
	}
	var err error
	switch e := t.ent.(type) {
	case *sketch.Line:
		_, err = sk.TrimLine(e, t.point)
	case *sketch.Circle:
		_, err = sk.TrimCircle(e, t.point)
	case *sketch.Arc:
		_, err = sk.TrimArc(e, t.point)
	default:
		err = fmt.Errorf("trim: unsupported target %T", t.ent)
	}
	return err
}

// SketchExtendTool lengthens the picked line's nearer end to the next crossing of its
// support with other sketch geometry.
type SketchExtendTool struct {
	dialogTool
	curveEditPick
}

// NewSketchExtendTool makes an extend tool.
func NewSketchExtendTool() *SketchExtendTool { return &SketchExtendTool{} }

func (t *SketchExtendTool) Name() string           { return "Extend" }
func (t *SketchExtendTool) Cancel(*Session)        { t.reset() }
func (t *SketchExtendTool) Prompt(*Session) string { return "Pick near the line end to extend." }

// Commit extends whichever end of the picked line is nearer the pick point.
func (t *SketchExtendTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("extend: no active sketch")
	}
	l, err := t.pickedLine()
	if err != nil {
		return fmt.Errorf("extend: %w", err)
	}
	_, err = sk.ExtendLine(l, nearerToEndB(l, t.point))
	return err
}

// SketchSplitTool splits the picked line into two at the pick point.
type SketchSplitTool struct {
	dialogTool
	curveEditPick
}

// NewSketchSplitTool makes a split tool.
func NewSketchSplitTool() *SketchSplitTool { return &SketchSplitTool{} }

func (t *SketchSplitTool) Name() string           { return "Split" }
func (t *SketchSplitTool) Cancel(*Session)        { t.reset() }
func (t *SketchSplitTool) Prompt(*Session) string { return "Pick the point to split the curve." }

// Commit splits the picked line at the pick point.
func (t *SketchSplitTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("split: no active sketch")
	}
	l, err := t.pickedLine()
	if err != nil {
		return fmt.Errorf("split: %w", err)
	}
	_, err = sk.SplitLine(l, t.point)
	return err
}

// nearerToEndB reports whether p is closer to the line's B end than its A end (so extend
// lengthens the picked end).
func nearerToEndB(l *sketch.Line, p math.Point2) bool {
	return p.DistanceTo(l.B.Position()) <= p.DistanceTo(l.A.Position())
}
