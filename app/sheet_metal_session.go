// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridges for the sheet-metal tools' property windows (M13 UI). Each returns the
// running tool typed, or nil when the active tool is not that one — the head's dialog draw
// loop calls them to show the matching property panel.

// activeSheetMetalTool returns the running tool as T, or the zero value (nil) when the active
// tool is not of that type. A generic method (Go 1.27), private to this file's own use.
func (s *Session) activeSheetMetalTool[T Tool]() T {
	var zero T
	if s.tool == nil {
		return zero
	}
	t, _ := s.tool.tool.(T)
	return t
}

// ActiveSheetMetalFace returns the running Sheet Metal Face tool, or nil.
func (s *Session) ActiveSheetMetalFace() *SheetMetalFaceTool {
	return s.activeSheetMetalTool[*SheetMetalFaceTool]()
}

// ActiveSheetMetalFlange returns the running Sheet Metal Flange tool, or nil.
func (s *Session) ActiveSheetMetalFlange() *SheetMetalFlangeTool {
	return s.activeSheetMetalTool[*SheetMetalFlangeTool]()
}

// ActiveSheetMetalHem returns the running Sheet Metal Hem tool, or nil.
func (s *Session) ActiveSheetMetalHem() *SheetMetalHemTool {
	return s.activeSheetMetalTool[*SheetMetalHemTool]()
}

// ActiveSheetMetalContourFlange returns the running Sheet Metal Contour Flange tool, or nil.
func (s *Session) ActiveSheetMetalContourFlange() *SheetMetalContourFlangeTool {
	return s.activeSheetMetalTool[*SheetMetalContourFlangeTool]()
}

// ActiveSheetMetalLoftedFlange returns the running Sheet Metal Lofted Flange tool, or nil.
func (s *Session) ActiveSheetMetalLoftedFlange() *SheetMetalLoftedFlangeTool {
	return s.activeSheetMetalTool[*SheetMetalLoftedFlangeTool]()
}

// ActiveSheetMetalContourRoll returns the running Sheet Metal Contour Roll tool, or nil.
func (s *Session) ActiveSheetMetalContourRoll() *SheetMetalContourRollTool {
	return s.activeSheetMetalTool[*SheetMetalContourRollTool]()
}

// ActiveSheetMetalBend returns the running Sheet Metal Bend tool, or nil.
func (s *Session) ActiveSheetMetalBend() *SheetMetalBendTool {
	return s.activeSheetMetalTool[*SheetMetalBendTool]()
}

// ActiveSheetMetalFold returns the running Sheet Metal Fold tool, or nil.
func (s *Session) ActiveSheetMetalFold() *SheetMetalFoldTool {
	return s.activeSheetMetalTool[*SheetMetalFoldTool]()
}

// ActiveSheetMetalCorner returns the running Sheet Metal Corner tool, or nil.
func (s *Session) ActiveSheetMetalCorner() *SheetMetalCornerTool {
	return s.activeSheetMetalTool[*SheetMetalCornerTool]()
}

// ActiveSheetMetalCornerSeam returns the running Sheet Metal Corner Seam tool, or nil.
func (s *Session) ActiveSheetMetalCornerSeam() *SheetMetalCornerSeamTool {
	return s.activeSheetMetalTool[*SheetMetalCornerSeamTool]()
}

// ActiveSheetMetalCut returns the running Sheet Metal Cut tool, or nil.
func (s *Session) ActiveSheetMetalCut() *SheetMetalCutTool {
	return s.activeSheetMetalTool[*SheetMetalCutTool]()
}

// ActiveSheetMetalUnfold returns the running Sheet Metal Unfold tool, or nil.
func (s *Session) ActiveSheetMetalUnfold() *SheetMetalUnfoldTool {
	return s.activeSheetMetalTool[*SheetMetalUnfoldTool]()
}

// ActiveSheetMetalRefold returns the running Sheet Metal Refold tool, or nil.
func (s *Session) ActiveSheetMetalRefold() *SheetMetalRefoldTool {
	return s.activeSheetMetalTool[*SheetMetalRefoldTool]()
}

// ActiveSheetMetalStyle returns the running Sheet Metal Style editor tool, or nil.
func (s *Session) ActiveSheetMetalStyle() *SheetMetalStyleTool {
	return s.activeSheetMetalTool[*SheetMetalStyleTool]()
}

// ActiveSheetMetalLip returns the running Sheet Metal Lip tool, or nil.
func (s *Session) ActiveSheetMetalLip() *SheetMetalLipTool {
	return s.activeSheetMetalTool[*SheetMetalLipTool]()
}

// ActiveSheetMetalRip returns the running Sheet Metal Rip tool, or nil.
func (s *Session) ActiveSheetMetalRip() *SheetMetalRipTool {
	return s.activeSheetMetalTool[*SheetMetalRipTool]()
}

// ActiveSheetMetalPunch returns the running Sheet Metal Punch tool, or nil.
func (s *Session) ActiveSheetMetalPunch() *SheetMetalPunchTool {
	return s.activeSheetMetalTool[*SheetMetalPunchTool]()
}

// ActiveSheetMetalCosmeticBend returns the running Sheet Metal Cosmetic Bend tool, or nil.
func (s *Session) ActiveSheetMetalCosmeticBend() *SheetMetalCosmeticBendTool {
	return s.activeSheetMetalTool[*SheetMetalCosmeticBendTool]()
}
