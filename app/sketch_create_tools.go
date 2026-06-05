// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// Additional Create-panel tools: Chamfer (two-line bevel, sibling of Fillet), Slot (a
// two-point centre axis + width), and Text (an anchor + string). Their geometry lives in
// model/sketch and is tested there; these tools are the interaction shells, tested via the
// app's headless click/pick path.

// SketchChamferTool bevels the corner between two picked lines (like Fillet, but a flat
// cut). It uses an equal setback on both lines.
type SketchChamferTool struct {
	pickCollector
	distance math.Scalar
}

// NewSketchChamferTool makes a chamfer tool with the given default setback.
func NewSketchChamferTool(distance float64) *SketchChamferTool {
	return &SketchChamferTool{pickCollector: pickCollector{want: 2}, distance: math.Scalar(distance)}
}

func (t *SketchChamferTool) Name() string                  { return "Chamfer" }
func (t *SketchChamferTool) Start(*Session)                {}
func (t *SketchChamferTool) Pick(_ *Session, s Selectable) { t.take(s) }
func (t *SketchChamferTool) CanCommit() bool               { return t.ready() }
func (t *SketchChamferTool) AutoCommitOnPick() bool        { return true }
func (t *SketchChamferTool) Cancel(*Session)               { t.reset() }
func (t *SketchChamferTool) Prompt(*Session) string        { return "Pick two lines to chamfer." }

// SetDistance sets the chamfer setback.
func (t *SketchChamferTool) SetDistance(d float64) { t.distance = math.Scalar(d) }

// Commit chamfers the two picked lines.
func (t *SketchChamferTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("chamfer: no active sketch")
	}
	l1, l2, err := twoPickedLines(t.picks)
	if err != nil {
		return err
	}
	_, err = sk.AddChamfer(l1, l2, t.distance, t.distance)
	return err
}

// SketchSlotTool draws a straight slot from two centre-axis clicks and a width.
type SketchSlotTool struct {
	points []math.Point2
	width  math.Scalar
}

// NewSketchSlotTool makes a slot tool with the given default width.
func NewSketchSlotTool(width float64) *SketchSlotTool {
	return &SketchSlotTool{width: math.Scalar(width)}
}

func (t *SketchSlotTool) Name() string              { return "Slot" }
func (t *SketchSlotTool) Start(*Session)            {}
func (t *SketchSlotTool) Pick(*Session, Selectable) {}
func (t *SketchSlotTool) Cancel(*Session)           { t.points = nil }
func (t *SketchSlotTool) AutoCommits() bool         { return true }
func (t *SketchSlotTool) CanCommit() bool           { return len(t.points) == 2 }
func (t *SketchSlotTool) Prompt(*Session) string {
	if len(t.points) == 0 {
		return "Click the slot's first centre point."
	}
	return "Click the slot's second centre point."
}

// ClickAt records a centre-axis endpoint from a clicked pixel (snapped).
func (t *SketchSlotTool) ClickAt(s *Session, px, py float64) {
	if p, ok := s.sketchClickPoint(px, py); ok {
		t.points = append(t.points, p)
	}
}

// SetWidth sets the slot width.
func (t *SketchSlotTool) SetWidth(w float64) { t.width = math.Scalar(w) }

// Commit creates the straight slot.
func (t *SketchSlotTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("slot: no active sketch")
	}
	_, err := sk.AddStraightSlot(t.points[0], t.points[1], t.width)
	return err
}

// CenterPointArcSlotTool draws an arc-shaped slot whose centre line is an arc given by its
// center, start and end (Inventor's Center Point Arc Slot); the width is a parameter.
type CenterPointArcSlotTool struct {
	collectClicks
	width math.Scalar
}

// NewCenterPointArcSlotTool makes a center-point arc-slot tool with the given default width.
func NewCenterPointArcSlotTool(width float64) *CenterPointArcSlotTool {
	return &CenterPointArcSlotTool{width: math.Scalar(width)}
}

func (t *CenterPointArcSlotTool) Name() string              { return "Center Point Arc Slot" }
func (t *CenterPointArcSlotTool) Start(*Session)            {}
func (t *CenterPointArcSlotTool) Pick(*Session, Selectable) {}
func (t *CenterPointArcSlotTool) Cancel(*Session)           { t.reset() }
func (t *CenterPointArcSlotTool) CanCommit() bool           { return len(t.pts) == 3 }
func (t *CenterPointArcSlotTool) AutoCommits() bool         { return true }

// SetWidth sets the slot width.
func (t *CenterPointArcSlotTool) SetWidth(w float64) { t.width = math.Scalar(w) }

// Commit creates the arc slot; it sweeps CCW when the end lies CCW of the start about the
// center (the cross product of the two radii is positive).
func (t *CenterPointArcSlotTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errNoSketch("center point arc slot")
	}
	center, start, end := t.pts[0], t.pts[1], t.pts[2]
	_, err := sk.AddArcSlot(center, start, end, t.width, leftTurn(center, start, end))
	return err
}

// Prompt guides the center, start and end of the slot's arc.
func (t *CenterPointArcSlotTool) Prompt(*Session) string {
	switch len(t.pts) {
	case 0:
		return "Click the arc slot center"
	case 1:
		return "Click the slot's first centre point"
	case 2:
		return "Click the slot's second centre point"
	default:
		return "Click OK to create the slot"
	}
}

func (t *CenterPointArcSlotTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{scalarParam("Width", &t.width)}}
}

// ThreePointArcSlotTool draws an arc-shaped slot whose centre line is the arc through three
// clicked points — start, a point on the arc, then end (Inventor's Three Point Arc Slot).
type ThreePointArcSlotTool struct {
	collectClicks
	width math.Scalar
}

// NewThreePointArcSlotTool makes a three-point arc-slot tool with the given default width.
func NewThreePointArcSlotTool(width float64) *ThreePointArcSlotTool {
	return &ThreePointArcSlotTool{width: math.Scalar(width)}
}

func (t *ThreePointArcSlotTool) Name() string              { return "Three Point Arc Slot" }
func (t *ThreePointArcSlotTool) Start(*Session)            {}
func (t *ThreePointArcSlotTool) Pick(*Session, Selectable) {}
func (t *ThreePointArcSlotTool) Cancel(*Session)           { t.reset() }
func (t *ThreePointArcSlotTool) CanCommit() bool           { return len(t.pts) == 3 }
func (t *ThreePointArcSlotTool) AutoCommits() bool         { return true }

// SetWidth sets the slot width.
func (t *ThreePointArcSlotTool) SetWidth(w float64) { t.width = math.Scalar(w) }

// Commit fits the centre arc's center as the circumcentre of start, through and end, then
// creates the arc slot; the sweep follows the start→through→end orientation.
func (t *ThreePointArcSlotTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errNoSketch("three point arc slot")
	}
	start, through, end := t.pts[0], t.pts[1], t.pts[2]
	center, ok := circumcenter(start, through, end)
	if !ok {
		return errors.New("three point arc slot: the three points are collinear")
	}
	_, err := sk.AddArcSlot(center, start, end, t.width, leftTurn(start, through, end))
	return err
}

// Prompt guides the start, a point on the arc, then the end.
func (t *ThreePointArcSlotTool) Prompt(*Session) string {
	switch len(t.pts) {
	case 0:
		return "Click the slot's first centre point"
	case 1:
		return "Click a point on the slot's arc"
	case 2:
		return "Click the slot's second centre point"
	default:
		return "Click OK to create the slot"
	}
}

func (t *ThreePointArcSlotTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{scalarParam("Width", &t.width)}}
}

// SketchTextTool places a text box at a clicked anchor; the string and height are set
// before committing (Inventor's text dialog). It does not auto-commit — the user enters
// the text first.
type SketchTextTool struct {
	anchor *math.Point2
	text   string
	height math.Scalar
}

// NewSketchTextTool makes a text tool with a default height.
func NewSketchTextTool() *SketchTextTool { return &SketchTextTool{height: 1} }

func (t *SketchTextTool) Name() string              { return "Text" }
func (t *SketchTextTool) Start(*Session)            {}
func (t *SketchTextTool) Pick(*Session, Selectable) {}
func (t *SketchTextTool) Cancel(*Session)           { t.anchor, t.text = nil, "" }
func (t *SketchTextTool) CanCommit() bool           { return t.anchor != nil && t.text != "" }
func (t *SketchTextTool) Prompt(*Session) string {
	if t.anchor == nil {
		return "Click to place the text anchor."
	}
	return "Type the text, then OK."
}

// ClickAt sets the text anchor from a clicked pixel (snapped).
func (t *SketchTextTool) ClickAt(s *Session, px, py float64) {
	if p, ok := s.sketchClickPoint(px, py); ok {
		t.anchor = &p
	}
}

// SetText and SetHeight set the string and character height.
func (t *SketchTextTool) SetText(text string) { t.text = text }
func (t *SketchTextTool) SetHeight(h float64) { t.height = math.Scalar(h) }

// Commit creates the text box.
func (t *SketchTextTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("text: no active sketch")
	}
	if t.anchor == nil || t.text == "" {
		return errors.New("text: needs an anchor and a non-empty string")
	}
	sk.TextBoxes().Add(*t.anchor, t.text, t.height, 0, sketch.TextLeft)
	return nil
}

// --- Params (generic property dialog) -------------------------------------

func (t *SketchChamferTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{scalarParam("Distance", &t.distance)}}
}

func (t *SketchSlotTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{scalarParam("Width", &t.width)}}
}

func (t *SketchTextTool) Params() ToolParams {
	return ToolParams{
		Texts:  []TextParam{{"Text", func() string { return t.text }, func(s string) { t.text = s }}},
		Floats: []FloatParam{scalarParam("Height", &t.height)},
	}
}
