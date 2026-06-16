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
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/build"
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
	drawChromeWindows(s)
	drawDocumentClosePrompt(s)
	drawFileDialog(s)
	return activated
}

func prepareChromeFrame(win *native.Window, s *app.Session) {
	if icons == nil {
		icons = newIconCache(win) // lazily bind the icon cache to this window
	}
	bindGraphicsImages(win) // client-graphics image billboards create textures on this window (M16-F05)
	reportWindowFrame(win, s)
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
		dockSideNodes = native.DockDefaultLayout(dockID, "Model", "Viewport", "Command")
		addInDockRightNode = 0 // any lazily split right band died with the old layout
		dockLaidOut = true
	}
}

// dockSideNodes remembers the default arrangement's node ids so add-in dockable
// windows can be docked beside the built-in panels on first show (M05-F03).
var dockSideNodes native.DockSideNodes

// reportWindowFrame mirrors the GLFW window's live state into the session so
// windows.listFrames serves real geometry (M05-F10). The caption follows the
// active document, like a document-centric title bar.
func reportWindowFrame(win *native.Window, s *app.Session) {
	_, _, w, h, maximized := win.WindowState()
	state := types.WindowNormal
	if maximized {
		state = types.WindowMaximized
	}
	caption := build.Title()
	if d := s.ActiveDocument(); d != nil {
		caption = d.DisplayName() + " — " + build.Title()
	}
	s.SetWindowFrameStatus(app.WindowFrameStatus{Caption: caption, State: state, Width: w, Height: h})
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
	drawPlaceComponentDialog(s) // Assemble ▸ Place: arms the component file picker (#763)
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
	drawGripSnapDialog(s)
	drawSheetMetalDialogs(s)
}

func drawSurfaceFeatureDialogs(s *app.Session) {
	drawFaceOffsetDialog(s)
	drawDraftDialog(s)
	drawDeleteFaceDialog(s)
	drawReplaceFaceDialog(s)
	drawThickenDialog(s)
}

func drawChromeWindows(s *app.Session) {
	drawBOMWindow(s) // Assemble ▸ Bill of Materials (#768)
	drawParametersWindow(s)
	drawPreferencesWindow(s)
	drawMaterialsWindow(s)
	drawLightingWindow(s)
	drawNamedViewsWindow(s)      // View ▸ Named Views (M16-F03 #404)
	drawColorStylesWindow(s)     // View ▸ Color Styles (M16-F02 #403/#408)
	drawDisplaySettingsWindow(s) // View ▸ Display Settings (M16-F07 #643)
	drawScriptConsole(s)
	drawKeymapEditor(s)  // Tools ▸ Customize Keyboard (M05-F17)
	drawCommandInput(s)  // command-alias input box (M05-F17)
	drawCommandWindow(s) // docked Command Window REPL panel (M26 F04)
	drawUpdateWindow(s)  // Help ▸ Check for Updates notification
	drawAddInPanels(s)   // add-in dockable windows (M05-F03)
	// M26 F03: toasts / prompt modal / message-center windows are retired — every message
	// now funnels into the docked Command Window, and prompts are answered inline there.
	drawWebViews(s)          // web dialogs/views (M05-F08)
	drawMarkingMenu(s)       // radial marking menu popup (M05-F12)
	drawSelectOtherWidget(s) // Select Other cycle control over stacked geometry (#910)
	serviceFileModalRequests(s)
}

// serviceFileModalRequests arms the file modal for ribbon buttons that request a file picker.
func serviceFileModalRequests(s *app.Session) {
	if s.TakeLoadEnvironmentRequest() { // the View ▸ Load HDR ribbon button arms the file modal
		fileModal.openFor(dialogLoadHDR)
	}
	if s.TakeImportMeshRequest() { // the Mesh ▸ Place Mesh ribbon button arms the file modal (#700)
		fileModal.openFor(dialogMeshRef)
	}
}

// handleKeyboard forwards keyboard input to the session's binding engine (M05-F17). Esc and
// F1 are handled first so they work even from a focused field (Esc is the universal cancel;
// F1 opens host help, not a bindable action). Otherwise, while no text widget owns the
// keyboard, every non-modifier key pressed this frame is sent as a chord carrying the held
// modifiers, so rebindable shortcuts (undo/redo, command shortcuts, …) resolve and fire.
func handleKeyboard(s *app.Session) {
	if native.F1Pressed() {
		_ = s.DisplayHelpTopic("", "")
	}
	mods := heldModifiers()
	if native.EscapePressed() {
		if s.SelectOtherActive() {
			s.CancelSelectOther() // Esc ends the cycle, keeping the highlighted candidate (#910)
		} else {
			_ = s.PressKey(app.KeyEvent{Key: "Escape", Mods: mods})
		}
	}
	// M26 F05: modifier chords (Ctrl/Alt — e.g. Ctrl+S, Ctrl+Z) fire even while the
	// command-window input is focused, so they work mid-typing; PressKey routes them through
	// the command line, echoing the command word and running it (the "autofill + Enter" path).
	ctrlOrAlt := mods.Has(app.CtrlMod) || mods.Has(app.AltMod)
	if ctrlOrAlt {
		dispatchPressedKeys(s, native.PressedKeys(), mods)
	}
	if native.WantTextInput() {
		return // a text widget (the command line, by default) owns plain typing → fills it
	}
	if ctrlOrAlt {
		return // modifier chords already handled above; don't double-dispatch them
	}
	// No text field is focused: plain shortcut keys dispatch directly (the legacy path).
	// NOTE: we deliberately do NOT force-refocus the command line here to make it a sticky
	// keyboard sink — calling SetKeyboardFocusHere while nothing is focused steals the active
	// item and disables ImGui mouse hover, which breaks viewport drag-orbit (regression guard
	// TestInWindowDockedViewportIsInteractive). True stickiness needs the viewport to decline
	// keyboard focus on click instead; see ADR-0037's follow-ups.
	dispatchPressedKeys(s, native.PressedKeys(), mods)
}

// dispatchPressedKeys sends each non-modifier key pressed this frame to the session as a
// chord carrying the held modifiers. Escape is already handled in handleKeyboard, so it is
// skipped here to avoid a double cancel. Kept pure (no native reads) so it is unit-testable.
func dispatchPressedKeys(s *app.Session, keys []string, mods app.Modifier) {
	for _, key := range keys {
		if key == "Escape" {
			continue
		}
		_ = s.PressKey(app.KeyEvent{Key: key, Mods: mods})
	}
}

// heldModifiers reads the modifier keys currently down.
func heldModifiers() app.Modifier {
	var mods app.Modifier
	if native.KeyCtrl() {
		mods |= app.CtrlMod
	}
	if native.KeyShift() {
		mods |= app.ShiftMod
	}
	if native.KeyAlt() {
		mods |= app.AltMod
	}
	return mods
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

// activeBodies returns the bodies the viewport draws for the active document — a part's own bodies
// or an assembly's placed components (Session.VisibleBodies resolves both, #769). It is nil only
// when no renderable document is active, so an assembly's components render and cache like a part's
// (a part-only gate here left the assembly viewport blank).
func activeBodies(s *app.Session) []*topo.Body {
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
