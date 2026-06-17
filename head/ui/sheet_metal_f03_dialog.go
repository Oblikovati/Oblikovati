//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import "oblikovati.org/app"

// Sheet-metal F03 modify-tool property panels (rip/lip/punch/cosmetic bend). Split from the F02
// modify dispatcher so each router stays within the statement budget; the buffers are shared
// with the other panels (re-seeded per tool by seedSheetMetal).

// drawSheetMetalF03Dialogs draws the active F03 modify tool's panel, returning true when it drew
// one.
func drawSheetMetalF03Dialogs(s *app.Session) bool {
	if t := s.ActiveSheetMetalPunch(); t != nil {
		sheetMetalPanel(s, "Punch", "Profile", "sm-punch", "Click a profile — every profile of its sketch is punched", t.PickCount(), t.ClearPicks, t.CanCommit(), nil)
		return true
	}
	if t := s.ActiveSheetMetalRip(); t != nil {
		drawSheetMetalRip(s, t)
		return true
	}
	if t := s.ActiveSheetMetalLip(); t != nil {
		drawSheetMetalLip(s, t)
		return true
	}
	if t := s.ActiveSheetMetalCosmeticBend(); t != nil {
		drawSheetMetalCosmeticBend(s, t)
		return true
	}
	return false
}

func drawSheetMetalRip(s *app.Session, t *app.SheetMetalRipTool) {
	seedSheetMetal(t, func() { smUI.gap = float32(t.Gap()) })
	sheetMetalPanel(s, "Rip", "Rip Line", "sm-rip", "Click a sketch line to rip along", t.PickCount(), t.ClearPicks, t.CanCommit(), func() {
		lengthCmRow(s, "Gap", "sm-rip-gap", &smUI.gap)
		t.SetGap(float64(smUI.gap))
	})
}

func drawSheetMetalLip(s *app.Session, t *app.SheetMetalLipTool) {
	seedSheetMetal(t, func() {
		smUI.height = float32(t.Height())
		smUI.length = float32(t.ReturnLength())
		smUI.angle = float32(t.Angle() * degPerRad)
	})
	sheetMetalPanel(s, "Lip", "Edge", "sm-lip", "Click a straight sheet edge", t.PickCount(), t.ClearPicks, t.CanCommit(), func() {
		lengthCmRow(s, "Height", "sm-lip-height", &smUI.height)
		t.SetHeight(float64(smUI.height))
		lengthCmRow(s, "Return", "sm-lip-return", &smUI.length)
		t.SetReturnLength(float64(smUI.length))
		angleDegRow(s, "Angle", "sm-lip-angle", &smUI.angle)
		t.SetAngle(float64(smUI.angle) / degPerRad)
	})
}

func drawSheetMetalCosmeticBend(s *app.Session, t *app.SheetMetalCosmeticBendTool) {
	seedSheetMetal(t, func() { smUI.angle = float32(t.Angle() * degPerRad) })
	sheetMetalPanel(s, "Cosmetic Bend", "Bend Line", "sm-cosbend", "Click a sketch line to mark", t.PickCount(), t.ClearPicks, t.CanCommit(), func() {
		angleDegRow(s, "Angle", "sm-cosbend-angle", &smUI.angle)
		t.SetAngle(float64(smUI.angle) / degPerRad)
	})
}
