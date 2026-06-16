// SPDX-License-Identifier: GPL-2.0-only

package app

// Pick-status accessors for the sheet-metal tools — the count of geometry picked and a clear,
// for the property panels' selection chips. Single-pick tools report 0 or 1.

func boolCount(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Face
func (t *SheetMetalFaceTool) PickCount() int { return len(t.profiles) }
func (t *SheetMetalFaceTool) ClearPicks()    { t.profiles = nil }

// Flange
func (t *SheetMetalFlangeTool) PickCount() int { return boolCount(t.edge != nil) }
func (t *SheetMetalFlangeTool) ClearPicks()    { t.edge = nil }

// Hem
func (t *SheetMetalHemTool) PickCount() int { return boolCount(t.edge != nil) }
func (t *SheetMetalHemTool) ClearPicks()    { t.edge = nil }

// Contour Flange (edge + profile)
func (t *SheetMetalContourFlangeTool) PickCount() int {
	return boolCount(t.edge != nil) + boolCount(t.profile != nil)
}
func (t *SheetMetalContourFlangeTool) ClearPicks() { t.edge, t.profile = nil, nil }

// Lofted Flange (two profiles)
func (t *SheetMetalLoftedFlangeTool) PickCount() int { return len(t.profiles) }
func (t *SheetMetalLoftedFlangeTool) ClearPicks()    { t.profiles = nil }

// Contour Roll (profile + axis)
func (t *SheetMetalContourRollTool) PickCount() int {
	return boolCount(t.profile != nil) + boolCount(t.axis != nil)
}
func (t *SheetMetalContourRollTool) ClearPicks() { t.profile, t.axis = nil, nil }

// Bend
func (t *SheetMetalBendTool) PickCount() int { return boolCount(t.line != nil) }
func (t *SheetMetalBendTool) ClearPicks()    { t.line = nil }

// Fold
func (t *SheetMetalFoldTool) PickCount() int { return boolCount(t.line != nil) }
func (t *SheetMetalFoldTool) ClearPicks()    { t.line = nil }

// Corner
func (t *SheetMetalCornerTool) PickCount() int { return len(t.edges) }
func (t *SheetMetalCornerTool) ClearPicks()    { t.edges = nil }

// Corner Seam
func (t *SheetMetalCornerSeamTool) PickCount() int { return len(t.edges) }
func (t *SheetMetalCornerSeamTool) ClearPicks()    { t.edges = nil }

// Cut
func (t *SheetMetalCutTool) PickCount() int { return boolCount(t.profile != nil) }
func (t *SheetMetalCutTool) ClearPicks()    { t.profile = nil }
