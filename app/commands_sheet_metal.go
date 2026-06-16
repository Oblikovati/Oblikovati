// SPDX-License-Identifier: GPL-2.0-only

package app

// Sheet Metal ribbon tab (M13). The tab shows on a part document and its commands enable only
// when the active part is in the sheet-metal environment (created with the sheet-metal
// subtype). Each command starts an interactive tool that picks geometry and commits a
// sheet-metal feature — the GUI counterpart of the features.add / sheetMetal.* wire surface.

// sheetMetalTabCommands returns the Sheet Metal tab commands in ribbon order, grouped into
// panels (Create, …). Further panels (Modify, Flat Pattern) are added as their tools land.
func sheetMetalTabCommands() []*CommandDefinition {
	return []*CommandDefinition{
		sheetMetalFaceCommand(),
		sheetMetalFlangeCommand(),
	}
}

// sheetMetalFaceCommand thickens a sketch profile into the base (or a secondary) wall.
func sheetMetalFaceCommand() *CommandDefinition {
	return NewCommand("SheetMetal.Face", "Face", "Create", func(s *Session) error {
		s.StartTool(NewSheetMetalFaceTool())
		return nil
	}).WithTab("Sheet Metal").WithRibbons(PartRibbon).WithEnable(hasActiveSheetMetalPart).
		WithIcon("sheet-metal-face").WithButtonStyle(LargeIconButton).
		WithTooltip("Thicken a closed sketch profile into a sheet-metal wall at the active gauge.")
}

// sheetMetalFlangeCommand folds a wall up on a straight sheet edge.
func sheetMetalFlangeCommand() *CommandDefinition {
	return NewCommand("SheetMetal.Flange", "Flange", "Create", func(s *Session) error {
		s.StartTool(NewSheetMetalFlangeTool())
		return nil
	}).WithTab("Sheet Metal").WithRibbons(PartRibbon).WithEnable(hasActiveSheetMetalPart).
		WithIcon("sheet-metal-flange").WithButtonStyle(LargeIconButton).
		WithTooltip("Fold a wall (flange) onto a straight sheet edge over a bend at the active gauge.")
}
