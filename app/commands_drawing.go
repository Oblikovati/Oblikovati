// SPDX-License-Identifier: GPL-2.0-only

package app

// Drawing ribbon commands (M14-F01, #384). New Drawing launches a drawing document from
// the ZeroDoc ribbon (alongside New Part / New Assembly). The Drawing tab — shown on the
// drawing ribbon — manages the document's sheets and the model it documents: each command
// starts a dialog-only tool (the GUI counterpart of the drawing.* wire surface).

// newDrawingCommand is the New Drawing launch action on the Get Started tab.
func newDrawingCommand() *CommandDefinition {
	return NewCommand("GetStarted.NewDrawing", "New Drawing", "Launch", func(s *Session) error {
		_, err := s.NewDrawing()
		return err
	}).WithTab("Get Started").WithRibbons(ZeroDocRibbon).
		WithIcon("new-drawing").WithButtonStyle(LargeIconButton).
		WithTooltip("New Drawing — create a drawing document with one sheet, where you lay out views and annotations of a model.")
}

// drawingTabCommands are the Drawing tab: the Sheets panel (add/delete sheets) and the
// Setup panel (choose the model the drawing documents).
func drawingTabCommands() []*CommandDefinition {
	return []*CommandDefinition{
		drawingToolCommand("New Sheet", "Sheets", "sheet-add",
			"Add a sheet to the drawing — choose a standard size (ISO A or ANSI) or a custom size and orientation.",
			func() Tool { return NewAddSheetTool() }),
		NewCommand("Drawing.DeleteSheet", "Delete Sheet", "Sheets", deleteActiveSheet).
			WithTab("Drawing").WithRibbons(DrawingRibbon).WithEnable(canDeleteSheet).
			WithIcon("sheet-delete").WithButtonStyle(SmallIconButton).
			WithTooltip("Delete the active sheet (a drawing must keep at least one sheet)."),
		drawingToolCommand("Model Reference", "Setup", "drawing-model",
			"Choose the model this drawing documents; its iProperties fill the title-block fields.",
			func() Tool { return NewModelReferenceTool() }),
		drawingToolCommand("Drafting Standard", "Setup", "drawing-standard",
			"Set the drawing's drafting standard (ISO or ANSI); it governs dimension, text and line appearance.",
			func() Tool { return NewDraftingStandardTool() }),
		drawingToolCommand("Base View", "Views", "drawing-base-view",
			"Project a base view of the referenced model onto the sheet (orientation, scale, hidden-line style).",
			func() Tool { return NewBaseViewTool() }),
		drawingToolCommand("Projected View", "Views", "drawing-projected-view",
			"Project an orthographic view off an existing base view (right/left/up/down).",
			func() Tool { return NewProjectedViewTool() }),
		drawingToolCommand("Auxiliary View", "Views", "drawing-auxiliary-view",
			"Fold an auxiliary view off a base view about a line at a chosen angle, to show an inclined face true-size.",
			func() Tool { return NewAuxiliaryViewTool() }),
		drawingToolCommand("Section View", "Views", "drawing-section-view",
			"Cut a section view through a base view (horizontal or vertical centreline): the near half is removed and the exposed faces are hatched.",
			func() Tool { return NewSectionViewTool() }),
		drawingToolCommand("Detail View", "Views", "drawing-detail-view",
			"Magnify a circular region of a base view at a larger scale.",
			func() Tool { return NewDetailViewTool() }),
		drawingToolCommand("Break View", "Views", "drawing-break-view",
			"Compress a base view by removing a band (horizontal or vertical) with break lines at the join.",
			func() Tool { return NewBreakViewTool() }),
		NewCommand("Drawing.ExportDXF", "Export DXF", "Output", requestDrawingDXFExport).
			WithTab("Drawing").WithRibbons(DrawingRibbon).WithEnable(hasActiveDrawing).
			WithIcon("drawing-export-dxf").WithButtonStyle(LargeIconButton).
			WithTooltip("Export the active sheet to a DXF file — view edges on Visible/Hidden layers, plus the border and title block."),
	}
}

// drawingToolCommand builds a Drawing-tab command that starts the given tool, enabled only
// when a drawing is active (mirrors sheetMetalToolCommand).
func drawingToolCommand(name, panel, icon, tip string, newTool func() Tool) *CommandDefinition {
	return NewCommand("Drawing."+stripSpaces(name), name, panel, func(s *Session) error {
		s.StartTool(newTool())
		return nil
	}).WithTab("Drawing").WithRibbons(DrawingRibbon).WithEnable(hasActiveDrawing).
		WithIcon(icon).WithButtonStyle(LargeIconButton).WithTooltip(tip)
}

// deleteActiveSheet removes the active sheet from the active drawing.
func deleteActiveSheet(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	active := c.Sheets().Active()
	if active == nil {
		return nil
	}
	if err := c.Sheets().Remove(active.Name()); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// canDeleteSheet reports whether the active drawing has more than one sheet (a drawing
// must keep at least one), so the Delete Sheet button is inert on a single-sheet drawing.
func canDeleteSheet(s *Session) bool {
	c, err := ActiveDrawing(s)
	return err == nil && c.Sheets().Count() > 1
}
