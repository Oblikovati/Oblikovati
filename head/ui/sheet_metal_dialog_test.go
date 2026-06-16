//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// TestSheetMetalDialogsRender opens a real window and renders each Sheet Metal tool's property
// panel (and the no-op early return when no sheet-metal tool runs). It skips cleanly where no
// display/Vulkan is available.
func TestSheetMetalDialogsRender(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	icons = nil // rebind the icon cache to this fresh window/context

	// Early return: no sheet-metal tool active → the panels draw nothing.
	win.BeginFrame()
	drawSheetMetalDialogs(app.NewSession())
	win.EndFrame(0.1, 0.1, 0.1)

	// Each tool active → its property panel renders.
	starts := []func(*app.Session){
		func(s *app.Session) { s.StartTool(app.NewSheetMetalStyleTool()) },
		func(s *app.Session) { s.StartTool(app.NewSheetMetalFaceTool()) },
		func(s *app.Session) { s.StartTool(app.NewSheetMetalFlangeTool()) },
		func(s *app.Session) { s.StartTool(app.NewSheetMetalHemTool()) },
		func(s *app.Session) { s.StartTool(app.NewSheetMetalContourFlangeTool()) },
		func(s *app.Session) { s.StartTool(app.NewSheetMetalLoftedFlangeTool()) },
		func(s *app.Session) { s.StartTool(app.NewSheetMetalContourRollTool()) },
		func(s *app.Session) { s.StartTool(app.NewSheetMetalBendTool()) },
		func(s *app.Session) { s.StartTool(app.NewSheetMetalFoldTool()) },
		func(s *app.Session) { s.StartTool(app.NewSheetMetalCornerTool()) },
		func(s *app.Session) { s.StartTool(app.NewSheetMetalCornerSeamTool()) },
		func(s *app.Session) { s.StartTool(app.NewSheetMetalCutTool()) },
		func(s *app.Session) { s.StartTool(app.NewSheetMetalUnfoldTool()) },
		func(s *app.Session) { s.StartTool(app.NewSheetMetalRefoldTool()) },
	}
	for _, start := range starts {
		s := app.NewSession()
		start(s)
		win.BeginFrame()
		drawSheetMetalDialogs(s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
}
