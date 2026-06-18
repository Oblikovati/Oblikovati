// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
)

// Drawing-sketch tools (M14-F08 #638): drop 2D geometry directly in sheet space. The GUI tools
// stamp a default-sized primitive at the cursor in a new sketch; the API/bridge author arbitrary
// geometry via drawingSketches.addEntity.

// sketchRectDefaultMM and sketchCircleDefaultMM size the stamped primitives.
const (
	sketchRectWidthMM  = 50.0
	sketchRectHeightMM = 30.0
	sketchCircleRadMM  = 15.0
)

// SketchRectangleTool drops a new sketch containing a default-sized rectangle at the cursor.
type SketchRectangleTool struct{ centerX, centerY float64 }

// NewSketchRectangleTool creates the tool.
func NewSketchRectangleTool() *SketchRectangleTool {
	return &SketchRectangleTool{centerX: 150, centerY: 150}
}

func (t *SketchRectangleTool) Name() string              { return "Sketch Rectangle" }
func (t *SketchRectangleTool) Start(*Session)            {}
func (t *SketchRectangleTool) Pick(*Session, Selectable) {}
func (t *SketchRectangleTool) CanCommit() bool           { return true }
func (t *SketchRectangleTool) Cancel(*Session)           {}
func (t *SketchRectangleTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// Commit drops a sketch with a rectangle centred on the placed point.
func (t *SketchRectangleTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	sk := c.Sheets().Active().Sketches().Add("")
	w, h := sketchRectWidthMM/2, sketchRectHeightMM/2
	corners := [][2]float64{{t.centerX - w, t.centerY - h}, {t.centerX + w, t.centerY + h}}
	if _, err := c.Sheets().Active().Sketches().AddEntity(sk.Name(), types.SketchRectangleEntity, corners, 0); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// SketchCircleTool drops a new sketch containing a default-sized circle at the cursor.
type SketchCircleTool struct{ centerX, centerY float64 }

// NewSketchCircleTool creates the tool.
func NewSketchCircleTool() *SketchCircleTool { return &SketchCircleTool{centerX: 150, centerY: 150} }

func (t *SketchCircleTool) Name() string              { return "Sketch Circle" }
func (t *SketchCircleTool) Start(*Session)            {}
func (t *SketchCircleTool) Pick(*Session, Selectable) {}
func (t *SketchCircleTool) CanCommit() bool           { return true }
func (t *SketchCircleTool) Cancel(*Session)           {}
func (t *SketchCircleTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// Commit drops a sketch with a circle centred on the placed point.
func (t *SketchCircleTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	sk := c.Sheets().Active().Sketches().Add("")
	centre := [][2]float64{{t.centerX, t.centerY}}
	if _, err := c.Sheets().Active().Sketches().AddEntity(sk.Name(), types.SketchCircleEntity, centre, sketchCircleRadMM); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}
