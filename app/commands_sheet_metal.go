// SPDX-License-Identifier: GPL-2.0-only

package app

// Sheet Metal ribbon tab (M13). The tab shows on a part document and its commands enable only
// when the active part is in the sheet-metal environment (created with the sheet-metal
// subtype). Each command starts an interactive tool that picks geometry and commits a
// sheet-metal feature — the GUI counterpart of the features.add / sheetMetal.* / flatPattern.*
// wire surface. Commands are grouped into the Create, Modify and Flat Pattern panels.

// sheetMetalTabCommands returns the Sheet Metal tab commands in ribbon order.
func sheetMetalTabCommands() []*CommandDefinition {
	return []*CommandDefinition{
		// Create panel — the walls.
		sheetMetalToolCommand("Face", "Create", "sheet-metal-face",
			"Thicken a closed sketch profile into a sheet-metal wall at the active gauge.",
			func() Tool { return NewSheetMetalFaceTool() }),
		sheetMetalToolCommand("Flange", "Create", "sheet-metal-flange",
			"Fold a wall (flange) onto a straight sheet edge over a bend at the active gauge.",
			func() Tool { return NewSheetMetalFlangeTool() }),
		sheetMetalToolCommand("Hem", "Create", "sheet-metal-hem",
			"Fold a hem (a safe, reinforced edge) back on a straight sheet edge.",
			func() Tool { return NewSheetMetalHemTool() }),
		sheetMetalToolCommand("Contour Flange", "Create", "sheet-metal-contour-flange",
			"Sweep an open sketch profile along a sheet edge into a contoured wall.",
			func() Tool { return NewSheetMetalContourFlangeTool() }),
		sheetMetalToolCommand("Lofted Flange", "Create", "sheet-metal-lofted-flange",
			"Loft a transition wall between two sketch profiles.",
			func() Tool { return NewSheetMetalLoftedFlangeTool() }),
		sheetMetalToolCommand("Contour Roll", "Create", "sheet-metal-contour-roll",
			"Revolve an open profile about an axis line into a rolled wall.",
			func() Tool { return NewSheetMetalContourRollTool() }),
		// Modify panel — bends, corners and cuts.
		sheetMetalToolCommand("Bend", "Modify", "sheet-metal-bend",
			"Fold the sheet along a sketch line at the active bend radius.",
			func() Tool { return NewSheetMetalBendTool() }),
		sheetMetalToolCommand("Fold", "Modify", "sheet-metal-fold",
			"Fold the sheet along a sketch line, hinged at its centerline.",
			func() Tool { return NewSheetMetalFoldTool() }),
		sheetMetalToolCommand("Corner", "Modify", "sheet-metal-corner",
			"Chamfer or round a sheet-metal corner edge.",
			func() Tool { return NewSheetMetalCornerTool() }),
		sheetMetalToolCommand("Corner Seam", "Modify", "sheet-metal-corner-seam",
			"Open a gap or overlap seam where two flanges meet at a corner.",
			func() Tool { return NewSheetMetalCornerSeamTool() }),
		sheetMetalToolCommand("Cut", "Modify", "sheet-metal-cut",
			"Cut a closed sketch profile through the sheet (through all).",
			func() Tool { return NewSheetMetalCutTool() }),
		// Flat Pattern panel — develop and re-fold.
		sheetMetalToolCommand("Create Flat Pattern", "Flat Pattern", "sheet-metal-unfold",
			"Develop the part flat (unfold every bend) so it can be cut while flat or exported.",
			func() Tool { return NewSheetMetalUnfoldTool() }),
		sheetMetalToolCommand("Refold", "Flat Pattern", "sheet-metal-refold",
			"Re-fold the bends an earlier unfold flattened, restoring the folded part.",
			func() Tool { return NewSheetMetalRefoldTool() }),
	}
}

// sheetMetalToolCommand builds a Sheet Metal tab command that starts the given tool, gated on
// the active part being sheet metal. The command id is "SheetMetal.<name without spaces>".
func sheetMetalToolCommand(name, panel, icon, tip string, newTool func() Tool) *CommandDefinition {
	return NewCommand("SheetMetal."+stripSpaces(name), name, panel, func(s *Session) error {
		s.StartTool(newTool())
		return nil
	}).WithTab("Sheet Metal").WithRibbons(PartRibbon).WithEnable(hasActiveSheetMetalPart).
		WithIcon(icon).WithButtonStyle(LargeIconButton).WithTooltip(tip)
}

// stripSpaces removes spaces from a command display name to form its id.
func stripSpaces(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r != ' ' {
			out = append(out, r)
		}
	}
	return string(out)
}
