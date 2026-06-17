// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/model/drawing"
)

// sheetSizeChoice pairs a New Sheet dropdown label with the size it selects, in dropdown
// order (index → size).
type sheetSizeChoice struct {
	label string
	size  types.SheetSize
}

var sheetSizeChoices = []sheetSizeChoice{
	{"Custom", types.SheetSizeCustom},
	{"A0", types.SheetSizeA0},
	{"A1", types.SheetSizeA1},
	{"A2", types.SheetSizeA2},
	{"A3", types.SheetSizeA3},
	{"A4", types.SheetSizeA4},
	{"ANSI A", types.SheetSizeAnsiA},
	{"ANSI B", types.SheetSizeAnsiB},
	{"ANSI C", types.SheetSizeAnsiC},
	{"ANSI D", types.SheetSizeAnsiD},
	{"ANSI E", types.SheetSizeAnsiE},
}

func sheetSizeIndexOf(s types.SheetSize) int {
	for i, c := range sheetSizeChoices {
		if c.size == s {
			return i
		}
	}
	return 0
}

// AddSheetTool adds a sheet to the active drawing. It is a dialog-only tool (no picks):
// the user chooses a standard size or a custom width×height and an orientation, then OK
// adds the sheet and makes it active.
type AddSheetTool struct {
	sizeIndex   int
	orientation int // index into {portrait, landscape}
	widthMM     float64
	heightMM    float64
}

// NewAddSheetTool starts on the default A3 landscape sheet (the drawing's default sheet).
func NewAddSheetTool() *AddSheetTool {
	return &AddSheetTool{
		sizeIndex:   sheetSizeIndexOf(types.SheetSizeA3),
		orientation: int(types.SheetLandscape),
		widthMM:     297,
		heightMM:    210,
	}
}

func (t *AddSheetTool) Name() string              { return "New Sheet" }
func (t *AddSheetTool) Start(*Session)            {}
func (t *AddSheetTool) Pick(*Session, Selectable) {}
func (t *AddSheetTool) CanCommit() bool           { return true }
func (t *AddSheetTool) Cancel(*Session)           {}

// Commit adds the configured sheet to the active drawing.
func (t *AddSheetTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	idx := t.sizeIndex
	if idx < 0 || idx >= len(sheetSizeChoices) {
		idx = 0
	}
	spec := drawing.SheetSpec{
		Size:        sheetSizeChoices[idx].size,
		Orientation: types.SheetOrientation(t.orientation),
		WidthMM:     t.widthMM,
		HeightMM:    t.heightMM,
	}
	if _, err := c.Sheets().Add(spec); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the size, orientation and custom-dimension fields for the property
// dialog (the width/height fields apply only when Size is Custom).
func (t *AddSheetTool) Params() ToolParams {
	sizeLabels := make([]string, len(sheetSizeChoices))
	for i, c := range sheetSizeChoices {
		sizeLabels[i] = c.label
	}
	return ToolParams{
		Choices: []ChoiceParam{
			{Label: "Size", Options: sizeLabels, Get: func() int { return t.sizeIndex }, Set: func(i int) { t.sizeIndex = i }},
			{Label: "Orientation", Options: []string{"Portrait", "Landscape"}, Get: func() int { return t.orientation }, Set: func(i int) { t.orientation = i }},
		},
		Floats: []FloatParam{
			{"Width (mm)", func() float64 { return t.widthMM }, func(v float64) { t.widthMM = v }},
			{"Height (mm)", func() float64 { return t.heightMM }, func(v float64) { t.heightMM = v }},
		},
	}
}
