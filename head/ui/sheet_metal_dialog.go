//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Sheet-metal property panels (M13 UI). While a Sheet Metal tool runs, a modeless panel shows
// its picked-geometry chip and its dimensions, then OK/Cancel. One dispatcher routes to the
// active tool's panel; a shared sheetMetalPanel draws the common frame.

const degPerRad = 57.29577951308232

// smUI holds the panels' edit buffers and the tool they were seeded from (so a fresh tool
// re-seeds its defaults the first frame).
var smUI struct {
	seeded                                 any
	height, angle, length, size, gap, roll float32
}

// drawSheetMetalDialogs shows the property panel for whichever Sheet Metal tool is active.
func drawSheetMetalDialogs(s *app.Session) {
	if t := s.ActiveSheetMetalFace(); t != nil {
		sheetMetalPanel(s, "Face", "Profile", "sm-face", "Click a closed sketch profile", t.PickCount(), t.ClearPicks, t.CanCommit(), nil)
		return
	}
	if t := s.ActiveSheetMetalFlange(); t != nil {
		drawSheetMetalFlange(s, t)
		return
	}
	if t := s.ActiveSheetMetalHem(); t != nil {
		drawSheetMetalHem(s, t)
		return
	}
	if t := s.ActiveSheetMetalContourFlange(); t != nil {
		sheetMetalPanel(s, "Contour Flange", "Edge + Profile", "sm-contour", "Click a sheet edge and an open profile", t.PickCount(), t.ClearPicks, t.CanCommit(), nil)
		return
	}
	if t := s.ActiveSheetMetalLoftedFlange(); t != nil {
		sheetMetalPanel(s, "Lofted Flange", "Profiles", "sm-lofted", "Click two sketch profiles", t.PickCount(), t.ClearPicks, t.CanCommit(), nil)
		return
	}
	if t := s.ActiveSheetMetalContourRoll(); t != nil {
		drawSheetMetalContourRoll(s, t)
		return
	}
	drawSheetMetalModifyDialogs(s)
}

// drawSheetMetalModifyDialogs routes the Modify- and Flat-Pattern-panel tools.
func drawSheetMetalModifyDialogs(s *app.Session) {
	if t := s.ActiveSheetMetalBend(); t != nil {
		sheetMetalPanel(s, "Bend", "Bend Line", "sm-bend", "Click a sketch line crossing the sheet", t.PickCount(), t.ClearPicks, t.CanCommit(), nil)
		return
	}
	if t := s.ActiveSheetMetalFold(); t != nil {
		sheetMetalPanel(s, "Fold", "Fold Line", "sm-fold", "Click a sketch line crossing the sheet", t.PickCount(), t.ClearPicks, t.CanCommit(), nil)
		return
	}
	if t := s.ActiveSheetMetalCorner(); t != nil {
		drawSheetMetalCorner(s, t)
		return
	}
	if t := s.ActiveSheetMetalCornerSeam(); t != nil {
		drawSheetMetalCornerSeam(s, t)
		return
	}
	if t := s.ActiveSheetMetalCut(); t != nil {
		sheetMetalPanel(s, "Cut", "Profile", "sm-cut", "Click a closed sketch profile to cut", t.PickCount(), t.ClearPicks, t.CanCommit(), nil)
		return
	}
	if t := s.ActiveSheetMetalUnfold(); t != nil {
		sheetMetalPanel(s, "Create Flat Pattern", "", "", "", 0, nil, t.CanCommit(), nil)
		return
	}
	if t := s.ActiveSheetMetalRefold(); t != nil {
		sheetMetalPanel(s, "Refold", "", "", "", 0, nil, t.CanCommit(), nil)
	}
}

func drawSheetMetalFlange(s *app.Session, t *app.SheetMetalFlangeTool) {
	seedSheetMetal(t, func() {
		smUI.height = float32(t.Height())
		smUI.angle = float32(t.Angle() * degPerRad)
	})
	sheetMetalPanel(s, "Flange", "Edge", "sm-flange", "Click a straight sheet edge", t.PickCount(), t.ClearPicks, t.CanCommit(), func() {
		propertyFloatRow("Height", "sm-flange-height", s.LengthUnitName(), &smUI.height)
		t.SetHeight(float64(smUI.height))
		propertyFloatRow("Angle", "sm-flange-angle", "deg", &smUI.angle)
		t.SetAngle(float64(smUI.angle) / degPerRad)
	})
}

func drawSheetMetalHem(s *app.Session, t *app.SheetMetalHemTool) {
	seedSheetMetal(t, func() { smUI.length = float32(t.Length()) })
	sheetMetalPanel(s, "Hem", "Edge", "sm-hem", "Click a straight sheet edge", t.PickCount(), t.ClearPicks, t.CanCommit(), func() {
		propertyFloatRow("Length", "sm-hem-length", s.LengthUnitName(), &smUI.length)
		t.SetLength(float64(smUI.length))
	})
}

func drawSheetMetalContourRoll(s *app.Session, t *app.SheetMetalContourRollTool) {
	seedSheetMetal(t, func() { smUI.roll = float32(t.Angle() * degPerRad) })
	sheetMetalPanel(s, "Contour Roll", "Profile + Axis", "sm-roll", "Click an open profile and its axis line", t.PickCount(), t.ClearPicks, t.CanCommit(), func() {
		propertyFloatRow("Angle", "sm-roll-angle", "deg", &smUI.roll)
		t.SetAngle(float64(smUI.roll) / degPerRad)
	})
}

func drawSheetMetalCorner(s *app.Session, t *app.SheetMetalCornerTool) {
	seedSheetMetal(t, func() { smUI.size = float32(t.Size()) })
	sheetMetalPanel(s, "Corner", "Corner Edges", "sm-corner", "Click corner edges to chamfer", t.PickCount(), t.ClearPicks, t.CanCommit(), func() {
		propertyFloatRow("Size", "sm-corner-size", s.LengthUnitName(), &smUI.size)
		t.SetSize(float64(smUI.size))
	})
}

func drawSheetMetalCornerSeam(s *app.Session, t *app.SheetMetalCornerSeamTool) {
	seedSheetMetal(t, func() { smUI.gap = float32(t.Gap()) })
	sheetMetalPanel(s, "Corner Seam", "Seam Edges", "sm-seam", "Click the seam edges", t.PickCount(), t.ClearPicks, t.CanCommit(), func() {
		propertyFloatRow("Gap", "sm-seam-gap", s.LengthUnitName(), &smUI.gap)
		t.SetGap(float64(smUI.gap))
	})
}

// seedSheetMetal loads the panel buffers from a tool the first frame it appears.
func seedSheetMetal(tool any, seed func()) {
	if smUI.seeded != tool {
		seed()
		smUI.seeded = tool
	}
}

// sheetMetalPanel draws the common Sheet Metal property-panel frame: a breadcrumb, an optional
// pick chip, an optional behavior section, and OK/Cancel.
func sheetMetalPanel(s *app.Session, title, pickLabel, pickID, hint string, pickCount int, clear func(), canCommit bool, body func()) {
	native.SetNextWindowSizeOnce(340, 220)
	if native.Begin(title) {
		drawFeatureBreadcrumb(title, "")
		if pickLabel != "" && propertySection("Input Geometry") {
			drawPickChipRow(pickLabel, pickID, countChipText(pickCount, pickLabel, "Select "+pickLabel), pickCount > 0, hint, clear)
		}
		if body != nil && propertySection("Behavior") {
			body()
		}
		native.Separator()
		drawCommitCancelButtons(s, canCommit)
	}
	native.End()
}
