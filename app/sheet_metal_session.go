// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridges for the sheet-metal tools' property windows (M13 UI). Each returns the
// running tool typed, or nil when the active tool is not that one — the head's dialog draw
// loop calls them to show the matching property panel. All route through activeTool
// (app/tool.go), the package-general form of what this file originally introduced.

// ActiveSheetMetalFace returns the running Sheet Metal Face tool, or nil.
func (s *Session) ActiveSheetMetalFace() *SheetMetalFaceTool {
	return s.activeTool[*SheetMetalFaceTool]()
}

// ActiveSheetMetalFlange returns the running Sheet Metal Flange tool, or nil.
func (s *Session) ActiveSheetMetalFlange() *SheetMetalFlangeTool {
	return s.activeTool[*SheetMetalFlangeTool]()
}

// ActiveSheetMetalHem returns the running Sheet Metal Hem tool, or nil.
func (s *Session) ActiveSheetMetalHem() *SheetMetalHemTool {
	return s.activeTool[*SheetMetalHemTool]()
}

// ActiveSheetMetalContourFlange returns the running Sheet Metal Contour Flange tool, or nil.
func (s *Session) ActiveSheetMetalContourFlange() *SheetMetalContourFlangeTool {
	return s.activeTool[*SheetMetalContourFlangeTool]()
}

// ActiveSheetMetalLoftedFlange returns the running Sheet Metal Lofted Flange tool, or nil.
func (s *Session) ActiveSheetMetalLoftedFlange() *SheetMetalLoftedFlangeTool {
	return s.activeTool[*SheetMetalLoftedFlangeTool]()
}

// ActiveSheetMetalContourRoll returns the running Sheet Metal Contour Roll tool, or nil.
func (s *Session) ActiveSheetMetalContourRoll() *SheetMetalContourRollTool {
	return s.activeTool[*SheetMetalContourRollTool]()
}

// ActiveSheetMetalBend returns the running Sheet Metal Bend tool, or nil.
func (s *Session) ActiveSheetMetalBend() *SheetMetalBendTool {
	return s.activeTool[*SheetMetalBendTool]()
}

// ActiveSheetMetalFold returns the running Sheet Metal Fold tool, or nil.
func (s *Session) ActiveSheetMetalFold() *SheetMetalFoldTool {
	return s.activeTool[*SheetMetalFoldTool]()
}

// ActiveSheetMetalCorner returns the running Sheet Metal Corner tool, or nil.
func (s *Session) ActiveSheetMetalCorner() *SheetMetalCornerTool {
	return s.activeTool[*SheetMetalCornerTool]()
}

// ActiveSheetMetalCornerSeam returns the running Sheet Metal Corner Seam tool, or nil.
func (s *Session) ActiveSheetMetalCornerSeam() *SheetMetalCornerSeamTool {
	return s.activeTool[*SheetMetalCornerSeamTool]()
}

// ActiveSheetMetalCut returns the running Sheet Metal Cut tool, or nil.
func (s *Session) ActiveSheetMetalCut() *SheetMetalCutTool {
	return s.activeTool[*SheetMetalCutTool]()
}

// ActiveSheetMetalUnfold returns the running Sheet Metal Unfold tool, or nil.
func (s *Session) ActiveSheetMetalUnfold() *SheetMetalUnfoldTool {
	return s.activeTool[*SheetMetalUnfoldTool]()
}

// ActiveSheetMetalRefold returns the running Sheet Metal Refold tool, or nil.
func (s *Session) ActiveSheetMetalRefold() *SheetMetalRefoldTool {
	return s.activeTool[*SheetMetalRefoldTool]()
}

// ActiveSheetMetalStyle returns the running Sheet Metal Style editor tool, or nil.
func (s *Session) ActiveSheetMetalStyle() *SheetMetalStyleTool {
	return s.activeTool[*SheetMetalStyleTool]()
}

// ActiveSheetMetalLip returns the running Sheet Metal Lip tool, or nil.
func (s *Session) ActiveSheetMetalLip() *SheetMetalLipTool {
	return s.activeTool[*SheetMetalLipTool]()
}

// ActiveSheetMetalRip returns the running Sheet Metal Rip tool, or nil.
func (s *Session) ActiveSheetMetalRip() *SheetMetalRipTool {
	return s.activeTool[*SheetMetalRipTool]()
}

// ActiveSheetMetalPunch returns the running Sheet Metal Punch tool, or nil.
func (s *Session) ActiveSheetMetalPunch() *SheetMetalPunchTool {
	return s.activeTool[*SheetMetalPunchTool]()
}

// ActiveSheetMetalCosmeticBend returns the running Sheet Metal Cosmetic Bend tool, or nil.
func (s *Session) ActiveSheetMetalCosmeticBend() *SheetMetalCosmeticBendTool {
	return s.activeTool[*SheetMetalCosmeticBendTool]()
}
