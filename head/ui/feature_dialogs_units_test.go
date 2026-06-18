//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// TestFeatureDialogsRenderUnitFields renders each migrated part-feature dialog in
// a real window, flipping the toggles that reveal every length/angle field, so
// the unit-aware rows (lengthCmRow/angleDegRow) actually execute. It guards that
// the #146 dialog sweep renders without panicking and exercises the conversion
// rows. Skips cleanly where no display/Vulkan is available.
func TestFeatureDialogsRenderUnitFields(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	icons = newIconCache(win) // bind the icon cache (dialogs draw icon toggles directly)

	frame := func(draw func()) {
		win.BeginFrame()
		draw()
		win.EndFrame(0.1, 0.1, 0.1)
	}

	t.Run("hole", func(t *testing.T) {
		s := app.NewSession()
		h := app.NewHoleTool()
		s.StartTool(h)
		frame(func() { drawHoleDialog(s) }) // plain: Diameter/Depth/point angle
		h.SetCounterbore(true)
		frame(func() { drawHoleDialog(s) }) // Seat Ø / Seat Depth
		h.SetCounterbore(false)
		h.SetCountersink(true)
		frame(func() { drawHoleDialog(s) }) // Seat Ø / Seat Angle
	})
	t.Run("coil", func(t *testing.T) {
		s := app.NewSession()
		s.StartTool(app.NewCoilTool())
		frame(func() { drawCoilDialog(s) }) // seeds + opens the panel
		// Now reveal the variable-pitch rows and flat end angles, then re-render.
		coilUI.variable = true
		coilUI.rows = []coilRowUI{{pitch: 1}, {pitch: 1, revolution: 1}}
		coilUI.startFlat, coilUI.endFlat = true, true
		frame(func() { drawCoilDialog(s) })
		coilUI.variable, coilUI.startFlat, coilUI.endFlat = false, false, false
	})
	t.Run("loft", func(t *testing.T) {
		s := app.NewSession()
		s.StartTool(app.NewLoftTool())
		frame(func() { drawLoftDialog(s) })        // seed loftUI
		loftUI.first.cond, loftUI.last.cond = 1, 2 // angle / direction → angle rows
		frame(func() { drawLoftDialog(s) })
	})
	for name, draw := range map[string]func(*app.Session){
		"sweep":      drawSweepDialog,
		"fillet":     drawFilletDialog,
		"chamfer":    drawChamferDialog,
		"shell":      drawShellDialog,
		"draft":      drawDraftDialog,
		"thicken":    drawThickenDialog,
		"faceOffset": drawFaceOffsetDialog,
	} {
		t.Run(name, func(t *testing.T) {
			s := app.NewSession()
			s.StartTool(toolFor(name))
			frame(func() { draw(s) })
			if name == "fillet" {
				filletUI.seeded = nil
				f := s.ActiveFillet()
				f.SetVariable(true) // start/end radius rows
				frame(func() { drawFilletDialog(s) })
				f.AddMidPoint() // #695: render a Position/Radius intermediate-point row
				frame(func() { drawFilletDialog(s) })
			}
		})
	}
}

// toolFor returns a fresh tool for the named dialog.
func toolFor(name string) app.Tool {
	switch name {
	case "sweep":
		return app.NewSweepTool()
	case "fillet":
		return app.NewFilletTool()
	case "chamfer":
		return app.NewChamferTool()
	case "shell":
		return app.NewShellTool()
	case "draft":
		return app.NewDraftTool()
	case "thicken":
		return app.NewThickenTool()
	default:
		return app.NewFaceOffsetTool()
	}
}
