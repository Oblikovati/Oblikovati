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
		// Setup panel — enter the environment, then edit the active rule.
		// Convert is the one command lit before conversion: it turns an ordinary part into a
		// sheet-metal part so the rest of the tab enables.
		NewCommand("SheetMetal.Convert", "Convert to Sheet Metal", "Setup", func(s *Session) error {
			return s.ConvertActiveToSheetMetal()
		}).WithTab("Sheet Metal").WithRibbons(PartRibbon).WithEnable(canConvertToSheetMetal).
			WithIcon("sheet-metal-convert").WithButtonStyle(LargeIconButton).
			WithTooltip("Convert the active part into a sheet-metal part, entering the sheet-metal environment."),
		// Style is a settings dialog, not a feature commit — it stays on the plain StartTool,
		// deliberately outside the commit gate (#1626; archguard pins it in
		// allowedPlainStartTools).
		sheetMetalCommand("Style", "Setup", "sheet-metal-style",
			"Edit the active sheet-metal rule: gauge thickness, bend radius, K-factor and corner relief.",
			func(s *Session) error { s.StartTool(NewSheetMetalStyleTool()); return nil }),
		// Create panel — the walls.
		sheetMetalToolCommand("Face", "Create", "sheet-metal-face",
			"Thicken a closed sketch profile into a sheet-metal wall at the active gauge.",
			func() PartFeatureTool { return NewSheetMetalFaceTool() }),
		sheetMetalToolCommand("Flange", "Create", "sheet-metal-flange",
			"Fold a wall (flange) onto a straight sheet edge over a bend at the active gauge.",
			func() PartFeatureTool { return NewSheetMetalFlangeTool() }),
		sheetMetalToolCommand("Hem", "Create", "sheet-metal-hem",
			"Fold a hem (a safe, reinforced edge) back on a straight sheet edge.",
			func() PartFeatureTool { return NewSheetMetalHemTool() }),
		sheetMetalToolCommand("Contour Flange", "Create", "sheet-metal-contour-flange",
			"Sweep an open sketch profile along a sheet edge into a contoured wall.",
			func() PartFeatureTool { return NewSheetMetalContourFlangeTool() }),
		sheetMetalToolCommand("Lofted Flange", "Create", "sheet-metal-lofted-flange",
			"Loft a transition wall between two sketch profiles.",
			func() PartFeatureTool { return NewSheetMetalLoftedFlangeTool() }),
		sheetMetalToolCommand("Contour Roll", "Create", "sheet-metal-contour-roll",
			"Revolve an open profile about an axis line into a rolled wall.",
			func() PartFeatureTool { return NewSheetMetalContourRollTool() }),
		// Modify panel — bends, corners and cuts.
		sheetMetalToolCommand("Bend", "Modify", "sheet-metal-bend",
			"Fold the sheet along a sketch line at the active bend radius.",
			func() PartFeatureTool { return NewSheetMetalBendTool() }),
		sheetMetalToolCommand("Fold", "Modify", "sheet-metal-fold",
			"Fold the sheet along a sketch line, hinged at its centerline.",
			func() PartFeatureTool { return NewSheetMetalFoldTool() }),
		sheetMetalToolCommand("Corner", "Modify", "sheet-metal-corner",
			"Chamfer or round a sheet-metal corner edge.",
			func() PartFeatureTool { return NewSheetMetalCornerTool() }),
		sheetMetalToolCommand("Corner Seam", "Modify", "sheet-metal-corner-seam",
			"Open a gap or overlap seam where two flanges meet at a corner.",
			func() PartFeatureTool { return NewSheetMetalCornerSeamTool() }),
		sheetMetalToolCommand("Cut", "Modify", "sheet-metal-cut",
			"Cut a closed sketch profile through the sheet (through all).",
			func() PartFeatureTool { return NewSheetMetalCutTool() }),
		sheetMetalToolCommand("Punch", "Modify", "sheet-metal-punch",
			"Punch every closed profile of a sketch through the sheet (a die pattern).",
			func() PartFeatureTool { return NewSheetMetalPunchTool() }),
		sheetMetalToolCommand("Rip", "Modify", "sheet-metal-rip",
			"Rip a through-thickness slit along a sketch line to open a seam for unfolding.",
			func() PartFeatureTool { return NewSheetMetalRipTool() }),
		sheetMetalToolCommand("Lip", "Modify", "sheet-metal-lip",
			"Fold a stiffening lip (a short flange curled 180° back on itself) onto a sheet edge.",
			func() PartFeatureTool { return NewSheetMetalLipTool() }),
		sheetMetalToolCommand("Cosmetic Bend", "Modify", "sheet-metal-cosmetic-bend",
			"Mark a cosmetic bend line for the bend table without folding the model.",
			func() PartFeatureTool { return NewSheetMetalCosmeticBendTool() }),
		// Flat Pattern panel — develop and re-fold.
		sheetMetalToolCommand("Create Flat Pattern", "Flat Pattern", "sheet-metal-unfold",
			"Develop the part flat (unfold every bend) so it can be cut while flat or exported.",
			func() PartFeatureTool { return NewSheetMetalUnfoldTool() }),
		sheetMetalToolCommand("Refold", "Flat Pattern", "sheet-metal-refold",
			"Re-fold the bends an earlier unfold flattened, restoring the folded part.",
			func() PartFeatureTool { return NewSheetMetalRefoldTool() }),
	}
}

// sheetMetalToolCommand builds a Sheet Metal tab command that starts the given feature tool
// through StartFeatureTool — the PartFeatureTool parameter makes the compiler prove every
// tool on this seam carries a DraftFeature, so the sick-config commit gate can never be
// skipped by omission (#1626, audit I3).
func sheetMetalToolCommand(name, panel, icon, tip string, newTool func() PartFeatureTool) *CommandDefinition {
	return sheetMetalCommand(name, panel, icon, tip, func(s *Session) error {
		s.StartFeatureTool(newTool())
		return nil
	})
}

// sheetMetalCommand builds a Sheet Metal tab command gated on the active part being sheet
// metal. The command id is "SheetMetal.<name without spaces>".
func sheetMetalCommand(name, panel, icon, tip string, run func(*Session) error) *CommandDefinition {
	return NewCommand("SheetMetal."+stripSpaces(name), name, panel, run).
		WithTab("Sheet Metal").WithRibbons(PartRibbon).WithEnable(hasActiveSheetMetalPart).
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
