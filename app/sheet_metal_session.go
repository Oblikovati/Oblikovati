// SPDX-License-Identifier: GPL-2.0-only

package app

// Session bridges for the sheet-metal tools' property windows (M13 UI). Each returns the
// running tool typed, or nil when the active tool is not that one — the head's dialog draw
// loop calls them to show the matching property panel.

// activeSheetMetalTool returns the running tool as T, or the zero value (nil) when the active
// tool is not of that type.
func activeSheetMetalTool[T Tool](s *Session) T {
	var zero T
	if s.tool == nil {
		return zero
	}
	t, _ := s.tool.tool.(T)
	return t
}

// ActiveSheetMetalFace returns the running Sheet Metal Face tool, or nil.
func (s *Session) ActiveSheetMetalFace() *SheetMetalFaceTool {
	return activeSheetMetalTool[*SheetMetalFaceTool](s)
}

// ActiveSheetMetalFlange returns the running Sheet Metal Flange tool, or nil.
func (s *Session) ActiveSheetMetalFlange() *SheetMetalFlangeTool {
	return activeSheetMetalTool[*SheetMetalFlangeTool](s)
}

// ActiveSheetMetalHem returns the running Sheet Metal Hem tool, or nil.
func (s *Session) ActiveSheetMetalHem() *SheetMetalHemTool {
	return activeSheetMetalTool[*SheetMetalHemTool](s)
}

// ActiveSheetMetalContourFlange returns the running Sheet Metal Contour Flange tool, or nil.
func (s *Session) ActiveSheetMetalContourFlange() *SheetMetalContourFlangeTool {
	return activeSheetMetalTool[*SheetMetalContourFlangeTool](s)
}

// ActiveSheetMetalLoftedFlange returns the running Sheet Metal Lofted Flange tool, or nil.
func (s *Session) ActiveSheetMetalLoftedFlange() *SheetMetalLoftedFlangeTool {
	return activeSheetMetalTool[*SheetMetalLoftedFlangeTool](s)
}

// ActiveSheetMetalContourRoll returns the running Sheet Metal Contour Roll tool, or nil.
func (s *Session) ActiveSheetMetalContourRoll() *SheetMetalContourRollTool {
	return activeSheetMetalTool[*SheetMetalContourRollTool](s)
}

// ActiveSheetMetalBend returns the running Sheet Metal Bend tool, or nil.
func (s *Session) ActiveSheetMetalBend() *SheetMetalBendTool {
	return activeSheetMetalTool[*SheetMetalBendTool](s)
}

// ActiveSheetMetalFold returns the running Sheet Metal Fold tool, or nil.
func (s *Session) ActiveSheetMetalFold() *SheetMetalFoldTool {
	return activeSheetMetalTool[*SheetMetalFoldTool](s)
}

// ActiveSheetMetalCorner returns the running Sheet Metal Corner tool, or nil.
func (s *Session) ActiveSheetMetalCorner() *SheetMetalCornerTool {
	return activeSheetMetalTool[*SheetMetalCornerTool](s)
}

// ActiveSheetMetalCornerSeam returns the running Sheet Metal Corner Seam tool, or nil.
func (s *Session) ActiveSheetMetalCornerSeam() *SheetMetalCornerSeamTool {
	return activeSheetMetalTool[*SheetMetalCornerSeamTool](s)
}

// ActiveSheetMetalCut returns the running Sheet Metal Cut tool, or nil.
func (s *Session) ActiveSheetMetalCut() *SheetMetalCutTool {
	return activeSheetMetalTool[*SheetMetalCutTool](s)
}

// ActiveSheetMetalUnfold returns the running Sheet Metal Unfold tool, or nil.
func (s *Session) ActiveSheetMetalUnfold() *SheetMetalUnfoldTool {
	return activeSheetMetalTool[*SheetMetalUnfoldTool](s)
}

// ActiveSheetMetalRefold returns the running Sheet Metal Refold tool, or nil.
func (s *Session) ActiveSheetMetalRefold() *SheetMetalRefoldTool {
	return activeSheetMetalTool[*SheetMetalRefoldTool](s)
}
