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

// take is the 3D-viewport pick route (Tool.Pick); PickSnap is the in-sketch route. Both
// funnel through takeEntity so the two entry points share the dedup / capacity rule.
func (c *pickCollector) take(sel Selectable) {
	if h, ok := sel.(SketchEntityHandle); ok {
		c.takeEntity(h.Entity)
	}
}

// PickSnap implements the entity-pick half of [SketchEntityTool]: inside a 2D sketch,
// Session.sketchEntityPointer delivers every entity click HERE, not through Pick. A modify
// tool that had only Pick therefore had its in-sketch clicks silently dropped, so Fillet/
// Chamfer/Offset/Mirror did nothing (#1799). The snap carries nothing a corner blend,
// offset, or mirror needs.
func (c *pickCollector) PickSnap(ent sketch.Entity, _ SnapResult) { c.takeEntity(ent) }

// takeEntity records one entity pick, ignoring a nil, a repeat of an already-picked entity
// (a corner needs two DISTINCT lines), and any pick past `want`.
func (c *pickCollector) takeEntity(ent sketch.Entity) {
	if ent == nil || len(c.picks) >= c.want {
		return
	}
	for _, p := range c.picks {
		if p == ent {
			return
		}
	}
	c.picks = append(c.picks, ent)
}

func (c *pickCollector) ready() bool             { return len(c.picks) == c.want }
func (c *pickCollector) reset()                  { c.picks = nil }
func (c *pickCollector) Picked() []sketch.Entity { return c.picks }

// The modify tools must satisfy SketchEntityTool so in-sketch clicks route to them (#1799).
var (
	_ SketchEntityTool = (*SketchFilletTool)(nil)
	_ SketchEntityTool = (*SketchOffsetTool)(nil)
	_ SketchEntityTool = (*SketchMirrorTool)(nil)
)

// SketchFilletTool rounds the corner between two picked lines with a tangent arc.
type SketchFilletTool struct {
	dialogTool
	pickCollector
	radius math.Scalar
}

// NewSketchFilletTool makes a fillet tool with the given default radius.
func NewSketchFilletTool(radius float64) *SketchFilletTool {
	return &SketchFilletTool{pickCollector: pickCollector{want: 2}, radius: math.Scalar(radius)}
}

func (t *SketchFilletTool) Name() string                  { return "Sketch Fillet" }
func (t *SketchFilletTool) Pick(_ *Session, s Selectable) { t.take(s) }

// Accepts highlights the lines a fillet can blend (the hover-candidate cue).
func (t *SketchFilletTool) Accepts(e sketch.Entity) bool { return entityKindIs(e, sketch.LineKind) }
func (t *SketchFilletTool) CanCommit() bool              { return t.ready() }
func (t *SketchFilletTool) AutoCommitOnPick() bool       { return true }
func (t *SketchFilletTool) Cancel(*Session)              { t.reset() }
func (t *SketchFilletTool) Prompt(*Session) string       { return "Pick two lines to fillet." }

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
	dialogTool
	pickCollector
	distance math.Scalar
}

// NewSketchOffsetTool makes an offset tool with the given default distance.
func NewSketchOffsetTool(distance float64) *SketchOffsetTool {
	return &SketchOffsetTool{pickCollector: pickCollector{want: 1}, distance: math.Scalar(distance)}
}

func (t *SketchOffsetTool) Name() string                  { return "Offset" }
func (t *SketchOffsetTool) Pick(_ *Session, s Selectable) { t.take(s) }

// Accepts highlights the curves OffsetEntity handles: line, circle, arc. Uses the entity's
// Kind() capability via entityKindIs, not a type switch, per the sketch-entity seam (#1624).
func (t *SketchOffsetTool) Accepts(e sketch.Entity) bool {
	return entityKindIs(e, sketch.LineKind, sketch.CircleKind, sketch.ArcKind)
}
func (t *SketchOffsetTool) CanCommit() bool        { return t.ready() }
func (t *SketchOffsetTool) AutoCommitOnPick() bool { return true }
func (t *SketchOffsetTool) Cancel(*Session)        { t.reset() }
func (t *SketchOffsetTool) Prompt(*Session) string { return "Pick a curve to offset." }

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
	dialogTool
	pickCollector
}

// NewSketchMirrorTool makes a mirror tool (pick geometry, then the mirror line).
func NewSketchMirrorTool() *SketchMirrorTool {
	return &SketchMirrorTool{pickCollector: pickCollector{want: 2}}
}

func (t *SketchMirrorTool) Name() string                  { return "Mirror" }
func (t *SketchMirrorTool) Pick(_ *Session, s Selectable) { t.take(s) }

// Accepts highlights any geometry as a mirror candidate — the first pick is the geometry
// to copy, the second the mirror line (Commit validates the second is a line).
func (t *SketchMirrorTool) Accepts(e sketch.Entity) bool { return e != nil }
func (t *SketchMirrorTool) CanCommit() bool              { return t.ready() }
func (t *SketchMirrorTool) AutoCommitOnPick() bool       { return true }
func (t *SketchMirrorTool) Cancel(*Session)              { t.reset() }
func (t *SketchMirrorTool) Prompt(*Session) string       { return "Pick geometry, then a mirror line." }

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
