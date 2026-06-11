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
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
)

// DrawChrome renders one frame of chrome for the session. Call it between
// Window.BeginFrame and Window.EndFrame. It returns the id of a command the user
// activated this frame (via ribbon or menu), or "" — the caller executes it, so this
// function stays free of side effects on the model. The window is needed to render the
// 3D viewport into its offscreen target.
func DrawChrome(win *native.Window, s *app.Session) string {
	prepareChromeFrame(win, s)
	drawMenuBar(s)
	activated := ""
	// The ribbon band must be drawn after the menu bar (it stacks beneath it in the
	// viewport work area) and before the dockspace (which fills what remains).
	if id := drawRibbon(s); id != "" {
		activated = id
	}
	layoutDockedPanels()
	drawBrowser(s)
	drawViewportIfPresent(win, s)
	drawChromeDialogs(s)
	drawStatusBar(s)
	drawChromeWindows(s)
	drawDocumentClosePrompt(s)
	drawFileDialog(s)
	return activated
}

func prepareChromeFrame(win *native.Window, s *app.Session) {
	if icons == nil {
		icons = newIconCache(win) // lazily bind the icon cache to this window
	}
	icons.beginFrame(s.ThemeRevision()) // free retired textures; flush composes on theme change
	applyThemeIfChanged(win, s)         // restyle ImGui + overlays when the theme changed (live preview)
	handleKeyboard(s)
	followActiveDocument(s)
}

// layoutDockedPanels hosts the dockspace under the fixed chrome (menu bar + ribbon
// band) and builds the one-time default arrangement of the dockable panels. The ribbon
// is intentionally absent: it is fixed chrome, not a dockable panel.
func layoutDockedPanels() {
	dockID := native.DockSpaceOverMain()
	if !dockLaidOut {
		native.DockDefaultLayout(dockID, "Model", "Viewport", "Status")
		dockLaidOut = true
	}
}

func drawViewportIfPresent(win *native.Window, s *app.Session) {
	if shouldDrawViewport(s) {
		drawViewportPanel(win, s)
	}
}

func drawChromeDialogs(s *app.Session) {
	drawDimensionPopup(s)
	drawToolParamsDialog(s) // generic dialog for parameterized sketch tools
	drawSketch3DSettings(s) // 3D-sketch settings while editing one
	drawSolidFeatureDialogs(s)
	drawSurfaceFeatureDialogs(s)
	drawOffsetPlaneDialog(s)
	drawFeatureEditDialog(s)
	drawWorkPlaneEditDialog(s)
}

func drawSolidFeatureDialogs(s *app.Session) {
	drawExtrudeDialog(s)
	drawRevolveDialog(s)
	drawCoilDialog(s)
	drawLoftDialog(s)
	drawSweepDialog(s)
	drawHoleDialog(s)
	drawChamferDialog(s)
	drawThreadDialog(s)
	drawFilletDialog(s)
	drawShellDialog(s)
	drawSplitDialog(s)
}

func drawSurfaceFeatureDialogs(s *app.Session) {
	drawFaceOffsetDialog(s)
	drawDraftDialog(s)
	drawDeleteFaceDialog(s)
	drawReplaceFaceDialog(s)
	drawThickenDialog(s)
}

func drawChromeWindows(s *app.Session) {
	drawParametersWindow(s)
	drawPreferencesWindow(s)
	drawMaterialsWindow(s)
	drawLightingWindow(s)
	drawScriptConsole(s)
	if s.TakeLoadEnvironmentRequest() { // the View ▸ Load HDR ribbon button arms the file modal
		fileModal.openFor(dialogLoadHDR)
	}
}

// handleKeyboard routes global shortcuts to the session. Esc cancels the active tool at
// any point (or clears the selection when idle) — Inventor's universal cancel. Ctrl+Z /
// Ctrl+Y (Ctrl+Shift+Z) navigate the undo stream, but only when no text field has focus,
// so a field's own editing keeps the keystroke.
func handleKeyboard(s *app.Session) {
	if native.EscapePressed() {
		_ = s.PressKey(app.KeyEvent{Key: "Escape"})
	}
	if !native.KeyCtrl() || native.WantTextInput() {
		return
	}
	mods := app.CtrlMod
	if native.KeyShift() {
		mods |= app.ShiftMod
	}
	switch {
	case native.UndoPressed():
		_ = s.PressKey(app.KeyEvent{Key: "z", Mods: mods})
	case native.RedoPressed():
		_ = s.PressKey(app.KeyEvent{Key: "y", Mods: mods})
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
	return s.VisibleBodies()
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
