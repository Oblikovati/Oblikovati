// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// pickCollector gathers sketch-entity picks for the modify/pattern tools. It mirrors the
// ConstraintTool flow (tool-first: activate, then feed picks) but commits a model edit
// rather than a constraint.
type pickCollector struct {
	picks []sketch.Entity
	want  int
}

func (c *pickCollector) take(sel Selectable) {
	if h, ok := sel.(SketchEntityHandle); ok && len(c.picks) < c.want {
		c.picks = append(c.picks, h.Entity)
	}
}

func (c *pickCollector) ready() bool             { return len(c.picks) == c.want }
func (c *pickCollector) reset()                  { c.picks = nil }
func (c *pickCollector) Picked() []sketch.Entity { return c.picks }

// SketchFilletTool rounds the corner between two picked lines with a tangent arc.
type SketchFilletTool struct {
	pickCollector
	radius math.Scalar
}

// NewSketchFilletTool makes a fillet tool with the given default radius.
func NewSketchFilletTool(radius float64) *SketchFilletTool {
	return &SketchFilletTool{pickCollector: pickCollector{want: 2}, radius: math.Scalar(radius)}
}

func (t *SketchFilletTool) Name() string                  { return "Sketch Fillet" }
func (t *SketchFilletTool) Start(*Session)                {}
func (t *SketchFilletTool) Pick(_ *Session, s Selectable) { t.take(s) }
func (t *SketchFilletTool) CanCommit() bool               { return t.ready() }
func (t *SketchFilletTool) AutoCommitOnPick() bool        { return true }
func (t *SketchFilletTool) Cancel(*Session)               { t.reset() }
func (t *SketchFilletTool) Prompt(*Session) string        { return "Pick two lines to fillet." }

// SetRadius sets the fillet radius.
func (t *SketchFilletTool) SetRadius(r float64) { t.radius = math.Scalar(r) }

// Params exposes the fillet radius to the generic property dialog.
func (t *SketchFilletTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{scalarParam("Radius", &t.radius)}}
}

// Commit applies the fillet to the two picked lines.
func (t *SketchFilletTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("sketch fillet: no active sketch")
	}
	l1, l2, err := twoPickedLines(t.picks)
	if err != nil {
		return err
	}
	_, err = sk.AddFillet(l1, l2, t.radius)
	return err
}

// SketchOffsetTool offsets a single picked curve by a distance.
type SketchOffsetTool struct {
	pickCollector
	distance math.Scalar
}

// NewSketchOffsetTool makes an offset tool with the given default distance.
func NewSketchOffsetTool(distance float64) *SketchOffsetTool {
	return &SketchOffsetTool{pickCollector: pickCollector{want: 1}, distance: math.Scalar(distance)}
}

func (t *SketchOffsetTool) Name() string                  { return "Offset" }
func (t *SketchOffsetTool) Start(*Session)                {}
func (t *SketchOffsetTool) Pick(_ *Session, s Selectable) { t.take(s) }
func (t *SketchOffsetTool) CanCommit() bool               { return t.ready() }
func (t *SketchOffsetTool) AutoCommitOnPick() bool        { return true }
func (t *SketchOffsetTool) Cancel(*Session)               { t.reset() }
func (t *SketchOffsetTool) Prompt(*Session) string        { return "Pick a curve to offset." }

// SetDistance sets the offset distance.
func (t *SketchOffsetTool) SetDistance(d float64) { t.distance = math.Scalar(d) }

// Params exposes the offset distance to the generic property dialog.
func (t *SketchOffsetTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{scalarParam("Distance", &t.distance)}}
}

// Commit offsets the picked curve.
func (t *SketchOffsetTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("offset: no active sketch")
	}
	_, err := sk.OffsetEntity(t.picks[0], t.distance)
	return err
}

// SketchMirrorTool mirrors the first picked entity across the second picked line.
type SketchMirrorTool struct {
	pickCollector
}

// NewSketchMirrorTool makes a mirror tool (pick geometry, then the mirror line).
func NewSketchMirrorTool() *SketchMirrorTool {
	return &SketchMirrorTool{pickCollector: pickCollector{want: 2}}
}

func (t *SketchMirrorTool) Name() string                  { return "Mirror" }
func (t *SketchMirrorTool) Start(*Session)                {}
func (t *SketchMirrorTool) Pick(_ *Session, s Selectable) { t.take(s) }
func (t *SketchMirrorTool) CanCommit() bool               { return t.ready() }
func (t *SketchMirrorTool) AutoCommitOnPick() bool        { return true }
func (t *SketchMirrorTool) Cancel(*Session)               { t.reset() }
func (t *SketchMirrorTool) Prompt(*Session) string        { return "Pick geometry, then a mirror line." }

// Commit mirrors the first picked entity across the second picked line.
func (t *SketchMirrorTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("mirror: no active sketch")
	}
	line, ok := t.picks[1].(*sketch.Line)
	if !ok {
		return errors.New("mirror: the second pick must be a line")
	}
	if sk.MirrorEntities([]sketch.Entity{t.picks[0]}, line) == nil {
		return errors.New("mirror: produced no copies (zero-length line?)")
	}
	return nil
}

// twoPickedLines casts two picks to lines.
func twoPickedLines(picks []sketch.Entity) (*sketch.Line, *sketch.Line, error) {
	if len(picks) != 2 {
		return nil, nil, errors.New("need two line picks")
	}
	l1, ok1 := picks[0].(*sketch.Line)
	l2, ok2 := picks[1].(*sketch.Line)
	if !ok1 || !ok2 {
		return nil, nil, errors.New("both picks must be lines")
	}
	return l1, l2, nil
}
