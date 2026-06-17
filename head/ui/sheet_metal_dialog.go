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
	// Style editor (rule) buffers.
	thickness, bendRadius, kFactor    float32
	reliefWidth, reliefDepth, ruleGap float32
	reliefShape                       int
}

// reliefShapeNames labels the relief-shape combo in types.ReliefShape order (round, square, tear).
var reliefShapeNames = []string{"Round", "Square", "Tear"}

// drawSheetMetalDialogs shows the property panel for whichever Sheet Metal tool is active. The
// dispatch is split across the Setup (Style), Create (wall), and Modify panels' tools so each
// router stays within the statement budget.
func drawSheetMetalDialogs(s *app.Session) {
	if t := s.ActiveSheetMetalStyle(); t != nil {
		drawSheetMetalStyle(s, t)
		return
	}
	if drawSheetMetalWallDialogs(s) {
		return
	}
	drawSheetMetalModifyDialogs(s)
}

// drawSheetMetalWallDialogs routes the Create-panel wall tools, returning true when it drew one.
func drawSheetMetalWallDialogs(s *app.Session) bool {
	if t := s.ActiveSheetMetalFace(); t != nil {
		sheetMetalPanel(s, "Face", "Profile", "sm-face", "Click a closed sketch profile", t.PickCount(), t.ClearPicks, t.CanCommit(), nil)
		return true
	}
	if t := s.ActiveSheetMetalFlange(); t != nil {
		drawSheetMetalFlange(s, t)
		return true
	}
	if t := s.ActiveSheetMetalHem(); t != nil {
		drawSheetMetalHem(s, t)
		return true
	}
	if t := s.ActiveSheetMetalContourFlange(); t != nil {
		sheetMetalPanel(s, "Contour Flange", "Edge + Profile", "sm-contour", "Click a sheet edge and an open profile", t.PickCount(), t.ClearPicks, t.CanCommit(), nil)
		return true
	}
	if t := s.ActiveSheetMetalLoftedFlange(); t != nil {
		sheetMetalPanel(s, "Lofted Flange", "Profiles", "sm-lofted", "Click two sketch profiles", t.PickCount(), t.ClearPicks, t.CanCommit(), nil)
		return true
	}
	if t := s.ActiveSheetMetalContourRoll(); t != nil {
		drawSheetMetalContourRoll(s, t)
		return true
	}
	return false
}

// drawSheetMetalModifyDialogs routes the Modify- and Flat-Pattern-panel tools across the F02,
// F03, and flat-pattern sub-routers (split so each stays within the statement budget).
func drawSheetMetalModifyDialogs(s *app.Session) {
	if drawSheetMetalF03Dialogs(s) {
		return
	}
	if drawSheetMetalBendCutDialogs(s) {
		return
	}
	drawSheetMetalFlatDialogs(s)
}

// drawSheetMetalBendCutDialogs routes the F02 Modify-panel tools (bend/fold/corner/seam/cut).
func drawSheetMetalBendCutDialogs(s *app.Session) bool {
	if t := s.ActiveSheetMetalBend(); t != nil {
		sheetMetalPanel(s, "Bend", "Bend Line", "sm-bend", "Click a sketch line crossing the sheet", t.PickCount(), t.ClearPicks, t.CanCommit(), nil)
		return true
	}
	if t := s.ActiveSheetMetalFold(); t != nil {
		sheetMetalPanel(s, "Fold", "Fold Line", "sm-fold", "Click a sketch line crossing the sheet", t.PickCount(), t.ClearPicks, t.CanCommit(), nil)
		return true
	}
	if t := s.ActiveSheetMetalCorner(); t != nil {
		drawSheetMetalCorner(s, t)
		return true
	}
	if t := s.ActiveSheetMetalCornerSeam(); t != nil {
		drawSheetMetalCornerSeam(s, t)
		return true
	}
	if t := s.ActiveSheetMetalCut(); t != nil {
		sheetMetalPanel(s, "Cut", "Profile", "sm-cut", "Click a closed sketch profile to cut", t.PickCount(), t.ClearPicks, t.CanCommit(), nil)
		return true
	}
	return false
}

// drawSheetMetalFlatDialogs routes the Flat-Pattern-panel tools (create flat pattern / refold).
func drawSheetMetalFlatDialogs(s *app.Session) {
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
		lengthCmRow(s, "Height", "sm-flange-height", &smUI.height)
		t.SetHeight(float64(smUI.height))
		angleDegRow(s, "Angle", "sm-flange-angle", &smUI.angle)
		t.SetAngle(float64(smUI.angle) / degPerRad)
	})
}

func drawSheetMetalHem(s *app.Session, t *app.SheetMetalHemTool) {
	seedSheetMetal(t, func() { smUI.length = float32(t.Length()) })
	sheetMetalPanel(s, "Hem", "Edge", "sm-hem", "Click a straight sheet edge", t.PickCount(), t.ClearPicks, t.CanCommit(), func() {
		lengthCmRow(s, "Length", "sm-hem-length", &smUI.length)
		t.SetLength(float64(smUI.length))
	})
}

func drawSheetMetalContourRoll(s *app.Session, t *app.SheetMetalContourRollTool) {
	seedSheetMetal(t, func() { smUI.roll = float32(t.Angle() * degPerRad) })
	sheetMetalPanel(s, "Contour Roll", "Profile + Axis", "sm-roll", "Click an open profile and its axis line", t.PickCount(), t.ClearPicks, t.CanCommit(), func() {
		angleDegRow(s, "Angle", "sm-roll-angle", &smUI.roll)
		t.SetAngle(float64(smUI.roll) / degPerRad)
	})
}

func drawSheetMetalCorner(s *app.Session, t *app.SheetMetalCornerTool) {
	seedSheetMetal(t, func() { smUI.size = float32(t.Size()) })
	sheetMetalPanel(s, "Corner", "Corner Edges", "sm-corner", "Click corner edges to chamfer", t.PickCount(), t.ClearPicks, t.CanCommit(), func() {
		lengthCmRow(s, "Size", "sm-corner-size", &smUI.size)
		t.SetSize(float64(smUI.size))
	})
}

func drawSheetMetalCornerSeam(s *app.Session, t *app.SheetMetalCornerSeamTool) {
	seedSheetMetal(t, func() { smUI.gap = float32(t.Gap()) })
	sheetMetalPanel(s, "Corner Seam", "Seam Edges", "sm-seam", "Click the seam edges", t.PickCount(), t.ClearPicks, t.CanCommit(), func() {
		lengthCmRow(s, "Gap", "sm-seam-gap", &smUI.gap)
		t.SetGap(float64(smUI.gap))
	})
}

// drawSheetMetalStyle edits the active part's rule (no geometry pick — a settings panel). It
// seeds the rule's gauge/radius/K-factor/relief into the buffers once, then writes every edited
// row back to the tool each frame; Commit re-authors the rule and recomputes the part.
func drawSheetMetalStyle(s *app.Session, t *app.SheetMetalStyleTool) {
	seedSheetMetal(t, func() { seedStyleBuffers(t) })
	sheetMetalPanel(s, "Sheet Metal Style", "", "", "", 0, nil, t.CanCommit(), func() {
		drawStyleRows(s, t)
	})
}

// seedStyleBuffers copies the tool's rule values into the panel buffers (first frame only).
func seedStyleBuffers(t *app.SheetMetalStyleTool) {
	smUI.thickness = float32(t.Thickness())
	smUI.bendRadius = float32(t.BendRadius())
	smUI.kFactor = float32(t.KFactor())
	smUI.reliefWidth = float32(t.ReliefWidth())
	smUI.reliefDepth = float32(t.ReliefDepth())
	smUI.ruleGap = float32(t.Gap())
	smUI.reliefShape = t.ReliefShapeIndex()
}

// drawStyleRows draws the rule's editable rows and writes each back to the tool every frame.
func drawStyleRows(s *app.Session, t *app.SheetMetalStyleTool) {
	unit := s.LengthUnitName()
	propertyFloatRow("Thickness", "sm-style-thickness", unit, &smUI.thickness)
	t.SetThickness(float64(smUI.thickness))
	propertyFloatRow("Bend Radius", "sm-style-radius", unit, &smUI.bendRadius)
	t.SetBendRadius(float64(smUI.bendRadius))
	propertyFloatRow("K-Factor", "sm-style-kfactor", "", &smUI.kFactor)
	t.SetKFactor(float64(smUI.kFactor))
	if i := propertyComboRow("Relief Shape", "sm-style-relief", reliefShapeNames, smUI.reliefShape); i >= 0 {
		smUI.reliefShape = i
	}
	t.SetReliefShapeIndex(smUI.reliefShape)
	propertyFloatRow("Relief Width", "sm-style-relief-w", unit, &smUI.reliefWidth)
	t.SetReliefWidth(float64(smUI.reliefWidth))
	propertyFloatRow("Relief Depth", "sm-style-relief-d", unit, &smUI.reliefDepth)
	t.SetReliefDepth(float64(smUI.reliefDepth))
	propertyFloatRow("Min Gap", "sm-style-gap", unit, &smUI.ruleGap)
	t.SetGap(float64(smUI.ruleGap))
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
