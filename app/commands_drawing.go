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
		drawingToolCommand("Slice View", "Views", "drawing-slice-view",
			"Show only the zero-thickness slice outline at a section line through a base view.",
			func() Tool { return NewSliceViewTool() }),
		drawingToolCommand("Breakout View", "Views", "drawing-breakout-view",
			"Reveal the interior within a region of a base view (a local cut-away).",
			func() Tool { return NewBreakoutViewTool() }),
		drawingToolCommand("Draft View", "Views", "drawing-draft-view",
			"Place a model-less framed view for manually-drawn 2D geometry.",
			func() Tool { return NewDraftViewTool() }),
		drawingToolCommand("Linear Dimension", "Dimension", "drawing-linear-dimension",
			"Dimension a base view's overall size (horizontal, vertical or aligned). The value is the true model size and updates with the model.",
			func() Tool { return NewLinearDimensionTool() }),
		drawingToolCommand("Radial Dimension", "Dimension", "drawing-radial-dimension",
			"Dimension every hole/circular edge in a base view as a radius or diameter callout. The value is the true model size and updates with the model.",
			func() Tool { return NewRadialDimensionTool() }),
		drawingToolCommand("Angular Dimension", "Dimension", "drawing-angular-dimension",
			"Dimension the corner angle between the first two non-parallel edges in a base view. The angle updates with the model.",
			func() Tool { return NewAngularDimensionTool() }),
		drawingToolCommand("Dimension Set", "Dimension", "drawing-dimension-set",
			"Dimension a base view's corners as a baseline (from one datum) or chain (running) set of linear dimensions. The values update with the model.",
			func() Tool { return NewDimensionSetTool() }),
		drawingToolCommand("Ordinate Dimension", "Dimension", "drawing-ordinate-dimension",
			"Dimension a base view's corners as an ordinate set: each corner's horizontal or vertical offset from a datum, shown as a leader to its value. The values update with the model.",
			func() Tool { return NewOrdinateDimensionTool() }),
		drawingToolCommand("Arc Length Dimension", "Dimension", "drawing-arc-length-dimension",
			"Dimension the swept length of a base view's first circular or arc edge (a hole's circumference or a fillet's arc). The value is the true model size and updates with the model.",
			func() Tool { return NewArcLengthDimensionTool() }),
		drawingToolCommand("Center Mark", "Annotate", "drawing-center-mark",
			"Add a centre mark (crosshair) at every hole/circular edge's centre in a base view. Each mark re-projects with the model.",
			func() Tool { return NewCenterMarkTool() }),
		drawingToolCommand("Centerline", "Annotate", "drawing-centerline",
			"Add the horizontal and vertical dash-dot symmetry centerlines through a view's centre, spanning its extent. They re-derive with the model.",
			func() Tool { return NewCenterlineTool() }),
		drawingToolCommand("Feature Control Frame", "Annotate", "drawing-feature-control-frame",
			"Add a GD&T feature control frame: a boxed geometric-tolerance callout (characteristic symbol, tolerance value and datum references).",
			func() Tool { return NewFeatureControlFrameTool() }),
		drawingToolCommand("Datum Feature", "Annotate", "drawing-datum-feature",
			"Add a GD&T datum feature symbol: a datum letter in a box with a filled datum triangle, marking a datum that feature control frames reference.",
			func() Tool { return NewDatumFeatureTool() }),
		drawingToolCommand("Surface Texture", "Annotate", "drawing-surface-texture",
			"Add an ISO surface texture symbol: the roughness checkmark with a finish value (any / machining required / as-cast).",
			func() Tool { return NewSurfaceTextureTool() }),
		drawingToolCommand("Parts List", "Table", "drawing-parts-list",
			"Add a parts list table sourced from the referenced assembly's BOM (item number, part number, description, quantity). It updates with the assembly.",
			func() Tool { return NewPartsListTool() }),
		drawingToolCommand("Balloon", "Table", "drawing-balloon",
			"Add a balloon: a circled parts-list item number with an optional leader to the component it tags.",
			func() Tool { return NewBalloonTool() }),
		drawingToolCommand("Hole Table", "Table", "drawing-hole-table",
			"Add a hole table listing a base view's holes with their X/Y from a datum origin and their diameter. It updates with the model.",
			func() Tool { return NewHoleTableTool() }),
		drawingToolCommand("Revision Table", "Table", "drawing-revision-table",
			"Add a revision table recording the drawing's change history (revision, date, description). It is seeded with the first revision; the API places multi-row tables.",
			func() Tool { return NewRevisionTableTool() }),
		drawingToolCommand("Revision Tag", "Table", "drawing-revision-tag",
			"Add a revision tag: a triangle holding a revision letter, flagging where that revision changed the drawing.",
			func() Tool { return NewRevisionTagTool() }),
		drawingToolCommand("Custom Table", "Table", "drawing-custom-table",
			"Add a general-purpose table with your own column headers and rows. It is seeded with two columns; the API places tables with arbitrary headers and rows.",
			func() Tool { return NewCustomTableTool() }),
		drawingToolCommand("Note", "Annotate", "drawing-note",
			"Add a free text note on the sheet, with an optional leader to the feature it annotates.",
			func() Tool { return NewNoteTool() }),
		drawingToolCommand("Hole Notes", "Annotate", "drawing-hole-notes",
			"Annotate every hole in a base view with a leadered diameter callout. The callouts are computed from the holes and re-resolve when the model changes.",
			func() Tool { return NewHoleNotesTool() }),
		drawingToolCommand("Sketch Rectangle", "Sketch", "drawing-sketch-rectangle",
			"Draw a rectangle in sheet space as a 2D sketch. The API places sketches with arbitrary lines, circles and rectangles.",
			func() Tool { return NewSketchRectangleTool() }),
		drawingToolCommand("Sketch Circle", "Sketch", "drawing-sketch-circle",
			"Draw a circle in sheet space as a 2D sketch.",
			func() Tool { return NewSketchCircleTool() }),
		drawingToolCommand("Hatch Region", "Sketch", "drawing-hatch-region",
			"Fill a rectangular region with a hatch pattern (general, cross-hatch or ANSI31).",
			func() Tool { return NewHatchRegionTool() }),
		drawingToolCommand("Center of Gravity", "Annotate", "drawing-cog-marker",
			"Mark a view's centre of gravity, positioned at the referenced model's centre of mass.",
			func() Tool { return NewCoGMarkerTool() }),
		drawingToolCommand("Revision Cloud", "Annotate", "drawing-revision-cloud",
			"Draw a scalloped revision cloud over a region of the sheet to highlight a change.",
			func() Tool { return NewRevisionCloudTool() }),
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
