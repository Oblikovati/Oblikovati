//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Package ui composes the Inventor-style chrome — menu bar, ribbon, model browser,
// and the viewport panel — from the live application Session each frame. There is no
// retained widget tree: every frame reads app.BuildRibbon / app.BuildBrowser and the
// current tool/selection, and Dear ImGui draws that (ADR-0004/0009). All layout lives
// here in Go; the native package only exposes ImGui verbs.
//
// The chrome is split by region across sibling files in this package:
//   - chrome.go         — DrawChrome orchestration, keyboard, shared part accessors
//   - chrome_menubar.go — the top menu bar and New Part
//   - chrome_ribbon.go  — the ribbon (tabs, panels, command buttons)
//   - chrome_doctabs.go — the document tab strip and active-document follow
//   - chrome_viewport.go— the Vulkan viewport panel, picking and overlays
//   - chrome_statusbar.go — the status bar
package ui

import (
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/model/compdef"
)

// DrawChrome renders one frame of chrome for the session. Call it between
// Window.BeginFrame and Window.EndFrame. It returns the id of a command the user
// activated this frame (via ribbon or menu), or "" — the caller executes it, so this
// function stays free of side effects on the model. The window is needed to render the
// 3D viewport into its offscreen target.
func DrawChrome(win *native.Window, s *app.Session) string {
	if icons == nil {
		icons = newIconCache(win) // lazily bind the icon cache to this window
	}
	applyThemeIfChanged(win, s) // restyle ImGui + overlays when the theme changed (live preview)
	handleKeyboard(s)
	activated := drawMenuBar(s)
	dockID := native.DockSpaceOverMain()
	if !dockLaidOut {
		native.DockDefaultLayout(dockID, "Ribbon", "Model", "Viewport", "Status")
		dockLaidOut = true
	}
	followActiveDocument(s)
	if id := drawRibbon(s); id != "" {
		activated = id
	}
	drawBrowser(s)
	drawViewportPanel(win, s)
	drawDimensionPopup(s)
	drawExtrudeDialog(s)
	drawRevolveDialog(s)
	drawCoilDialog(s)
	drawLoftDialog(s)
	drawSweepDialog(s)
	drawHoleDialog(s)
	drawChamferDialog(s)
	drawFilletDialog(s)
	drawShellDialog(s)
	drawFaceOffsetDialog(s)
	drawDraftDialog(s)
	drawDeleteFaceDialog(s)
	drawReplaceFaceDialog(s)
	drawThickenDialog(s)
	drawOffsetPlaneDialog(s)
	drawFeatureEditDialog(s)
	drawStatusBar(s)
	drawParametersWindow(s)
	drawPreferencesWindow(s)
	drawMaterialsWindow(s)
	drawLightingWindow(s)
	if s.TakeLoadEnvironmentRequest() { // the View ▸ Load HDR ribbon button arms the file modal
		fileModal.openFor(dialogLoadHDR)
	}
	drawFileDialog(s)
	return activated
}

// handleKeyboard routes global shortcuts to the session. Esc cancels the active tool at
// any point (or clears the selection when idle) — Inventor's universal cancel.
func handleKeyboard(s *app.Session) {
	if native.EscapePressed() {
		_ = s.PressKey(app.KeyEvent{Key: "Escape"})
	}
}

// dockLaidOut guards the one-time default panel arrangement (the dockspace persists
// across frames; the layout is only built once so the user can rearrange afterwards).
var dockLaidOut bool

// clampDim keeps the offscreen target at least 1px (ImGui can report a zero/negative
// content region on the first frame or when the panel is collapsed).
func clampDim(v float32) int {
	if v < 1 {
		return 1
	}
	return int(v)
}

// activeBodies returns the surface bodies of the active part, or nil.
func activeBodies(s *app.Session) []*topo.Body {
	part := activePart(s)
	if part == nil {
		return nil
	}
	return part.SurfaceBodies().All()
}

// activePart returns the active document's part component definition, or nil.
func activePart(s *app.Session) *compdef.PartComponentDefinition {
	d := s.ActiveDocument()
	if d == nil {
		return nil
	}
	part, _ := d.Content().(*compdef.PartComponentDefinition)
	return part
}
