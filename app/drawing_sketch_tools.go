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
type SketchRectangleTool struct {
	dialogTool
	centerX, centerY float64
}

// NewSketchRectangleTool creates the tool.
func NewSketchRectangleTool() *SketchRectangleTool {
	return &SketchRectangleTool{centerX: 150, centerY: 150}
}

func (t *SketchRectangleTool) Name() string              { return "Sketch Rectangle" }
func (t *SketchRectangleTool) CanCommit() bool           { return true }
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

// hatchPatterns indexes the hatch-region Pattern dropdown.
var hatchPatterns = []types.HatchPattern{types.HatchGeneral, types.HatchCross, types.HatchANSI31}

// HatchRegionTool fills a default-sized rectangular region with a hatch pattern at the cursor.
type HatchRegionTool struct {
	dialogTool
	patternIndex     int
	centerX, centerY float64
}

// NewHatchRegionTool creates the tool.
func NewHatchRegionTool() *HatchRegionTool { return &HatchRegionTool{centerX: 150, centerY: 150} }

func (t *HatchRegionTool) Name() string              { return "Hatch Region" }
func (t *HatchRegionTool) CanCommit() bool           { return true }
func (t *HatchRegionTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// Commit drops a hatch-filled rectangle centred on the placed point.
func (t *HatchRegionTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	pattern := hatchPatterns[clampIndex(t.patternIndex, len(hatchPatterns))]
	w, h := sketchRectWidthMM/2, sketchRectHeightMM/2
	if _, err := c.Sheets().Active().Sketches().AddHatchRegion("", t.centerX-w, t.centerY-h, sketchRectWidthMM, sketchRectHeightMM, pattern, 0); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the hatch-pattern choice.
func (t *HatchRegionTool) Params() ToolParams {
	return ToolParams{Choices: []ChoiceParam{
		{Label: "Pattern", Options: []string{"General", "Cross", "ANSI31"}, Get: func() int { return t.patternIndex }, Set: func(i int) { t.patternIndex = i }},
	}}
}

// SketchCircleTool drops a new sketch containing a default-sized circle at the cursor.
type SketchCircleTool struct {
	dialogTool
	centerX, centerY float64
}

// NewSketchCircleTool creates the tool.
func NewSketchCircleTool() *SketchCircleTool { return &SketchCircleTool{centerX: 150, centerY: 150} }

func (t *SketchCircleTool) Name() string              { return "Sketch Circle" }
func (t *SketchCircleTool) CanCommit() bool           { return true }
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
