//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// (The package comment was promoted to doc.go — #1669, M40 audit D12.)

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
	if s.ShowMenuBar() {
		drawMenuBar(s)
	}
	activated := ""
	// The ribbon band must be drawn after the menu bar (it stacks beneath it in the
	// viewport work area) and before the dockspace (which fills what remains).
	if id := drawRibbon(s); id != "" {
		activated = id
	}
	// The status bar is the ribbon's mirror image: a fixed band claiming the BOTTOM of the
	// work area before the dockspace fills what remains (Inventor-parity 2026-08-17;
	// C-inventor-ui-reference §7).
	if s.ShowStatusBar() {
		drawStatusBar(app.BuildStatus(s))
	}
	layoutDockedPanels()
	drawViewportIfPresent(win, s)
	// QAT / Application-menu deferred actions (G design D3): the band's
	// buttons only set flags; this block consumes them in DrawChrome's ID
	// context — OpenPopup beside BeginPopup (popup ID-stack guarantee), and
	// Open/Save before drawChromeDialogs so the modal opens this frame.
	// Every flag is reset on consumption, so none survives into next frame.
	if appMenuRequested {
		appMenuRequested = false
		native.SetNextWindowPos(appMenuX, appMenuBottomY)
		native.OpenPopup("##app-menu")
	}
	if qatOpenRequested {
		qatOpenRequested = false
		openViaHookOrDialog(s)
	}
	if qatSaveRequested {
		qatSaveRequested = false
		saveActive(s)
	}
	if native.BeginPopup("##app-menu") {
		drawFileMenu(s)
		native.EndPopup()
	}
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
	applyUIScale(win, s)                                 // push the user's font/icon scale before the ribbon sizes itself
	icons.beginFrame(s.ThemeRevision(), s.UIIconScale()) // free retired textures; reflush on theme or icon-scale change
	applyThemeIfChanged(win, s)                          // restyle ImGui + overlays when the theme changed (live preview)
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

// drawChromeDialogs draws the tool property panels. The per-tool feature dialogs
// (extrude, revolve, chamfer, sheet-metal, surface edits, …) self-register from their own
// files and are drawn by drawRegisteredToolDialogs — no hand-maintained roll-call, so a new
// tool minus its dialog fails the completeness check, not a live session (audit I4). The
// remaining entries are non-feature popups/editors that are not keyed on a single tool.
func drawChromeDialogs(s *app.Session) {
	drawDimensionPopup(s)
	drawToolParamsDialog(s) // generic panel for any ParameterizedTool (the default path)
	drawSketch3DSettings(s) // 3D-sketch settings while editing one
	drawRegisteredToolDialogs(s)
	drawOffsetPlaneDialog(s)
	drawFeatureEditDialog(s)
	drawWorkPlaneEditDialog(s)
	drawPlaceComponentDialog(s) // Assemble ▸ Place: arms the component file picker (#763)
}

func drawChromeWindows(s *app.Session) {
	// All built-in dockable windows (Model browser, Parameters, Materials, Lighting, the Command
	// REPL, …) render through the one shared closable path, so each gets a uniform close 'X' and a
	// View-menu toggle (#1473). The prompt/status mirror runs every frame, even while the Command
	// panel is hidden, so feedback is never lost.
	pumpCommandFeedback(s)
	drawDockablePanels(s)
	drawScriptConsole(s)
	drawKeymapEditor(s)         // Tools ▸ Customize Keyboard (M05-F17)
	drawMarkingMenuEditor(s)    // Tools ▸ Customize Marking Menu (REQ-005)
	drawCommandInput(s)         // command-alias input box (M05-F17)
	drawUpdateWindow(s)         // Help ▸ Check for Updates notification
	drawReportBugDialog(s)      // Help ▸ Report Bug
	drawAddInCatalogueWindow(s) // Tools ▸ Get Add-Ins… (#1164)
	drawAddInPanels(s)          // add-in dockable windows (M05-F03)
	// M26 F03: toasts / prompt modal / message-center windows are retired — every message
	// now funnels into the docked Command Window, and prompts are answered inline there.
	drawWebViews(s)          // web dialogs/views (M05-F08)
	drawMarkingMenu(s)       // radial marking menu popup (M05-F12)
	drawSelectOtherWidget(s) // Select Other cycle control over stacked geometry (#910)
	serviceFileModalRequests(s)
	serviceWindowOpenRequests(s)
	serviceFitViewRequest(s)
}

// fitViewSession is the two-method slice serviceFitViewRequest consumes (audit I5, the
// arrowSession pattern): the one-shot fit-view intent and the camera fit. *app.Session
// satisfies it implicitly, so the head does not hand the whole session to the fit widget.
type fitViewSession interface {
	TakeFitViewRequest() bool
	FitView()
}

// serviceFitViewRequest fits the camera once after an import that added visible geometry (#1645).
// The core raises the intent (Session.RequestFitView) rather than driving the camera, so a
// headless/CLI import is unaffected; the head consumes the one-shot here and calls FitView, which
// frames the UNION of all visible geometry (a small import into a large model is not yanked away).
func serviceFitViewRequest(s fitViewSession) {
	if s.TakeFitViewRequest() {
		s.FitView()
	}
}

// serviceWindowOpenRequests opens head-only windows that a ribbon command asked for from the
// core (the Get Started ▸ Manage buttons). Each request is one-shot, consumed here.
func serviceWindowOpenRequests(s *app.Session) {
	if s.TakeAddInCatalogueRequest() { // Get Started ▸ AddIn Catalogue
		OpenAddInCatalogue(s)
	}
	if s.TakePreferencesRequest() { // Get Started ▸ Preferences
		showPreferences = true
	}
}

// serviceFileModalRequests arms the file modal for ribbon buttons that request a file picker.
func serviceFileModalRequests(s *app.Session) {
	if s.TakeLoadEnvironmentRequest() { // the View ▸ Load HDR ribbon button arms the file modal
		fileModal.openFor(dialogLoadHDR)
	}
	if s.TakeImportMeshRequest() { // the Mesh ▸ Place Mesh ribbon button arms the file modal (#700)
		fileModal.openFor(dialogMeshRef)
	}
	if s.TakeImportPointCloudRequest() { // the 3D Model ▸ Import Point Cloud button arms the file modal (#645)
		fileModal.openFor(dialogPointCloud)
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
	// While either sketch entry surface has a started entry it owns plain keyboard (#790, #2014):
	// Enter/Esc/Backspace/typing are processed in the viewport (handleSketchHUD), so skip the
	// tool-level Esc/shortcut handling here. Ctrl/Alt chords still fire so Ctrl+S works mid-entry.
	if sketchEntryEngaged(s) && !mods.Has(app.CtrlMod) && !mods.Has(app.AltMod) {
		return
	}
	if native.EscapePressed() {
		handleEscape(s, mods)
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

// handleEscape routes an Escape press: it disarms whichever transient interaction owns Esc
// (Select Other, Zoom Window, Constrained Orbit, SteeringWheels), else forwards Esc to the
// session's binding engine (cancel the active tool/selection).
func handleEscape(s *app.Session, mods app.Modifier) {
	switch {
	case s.SelectOtherActive():
		s.CancelSelectOther() // Esc ends the cycle, keeping the highlighted candidate (#910)
	case s.ZoomWindowArmed():
		s.DisarmZoomWindow() // Esc cancels an armed Zoom Window before/while dragging (#913 N16)
	case s.ConstrainedOrbitActive():
		s.DisarmConstrainedOrbit() // Esc exits the Constrained Orbit tool (#913 N10)
	case s.SteeringWheelActive():
		s.DisarmSteeringWheel() // Esc dismisses the SteeringWheels menu (#913 N26)
	default:
		_ = s.PressKey(app.KeyEvent{Key: "Escape", Mods: mods})
	}
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
// It takes the document source rather than the whole session (audit I5): every caller only
// needs the active document, and the narrower parameter keeps new call sites honest.
func activePart(s activeDocumentSource) *compdef.PartComponentDefinition {
	d := s.ActiveDocument()
	if d == nil {
		return nil
	}
	part, _ := d.Content().(*compdef.PartComponentDefinition)
	return part
}
