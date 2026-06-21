//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// drawMenuBar renders the top menu bar. Only the few items that map to real session
// verbs are wired; the rest grow with the feature set.
func drawMenuBar(s *app.Session) {
	if !native.BeginMainMenuBar() {
		return
	}
	drawFileMenu(s)
	drawEditMenu(s)
	drawToolsMenu(s)
	drawHelpMenu(s)
	drawCommandSearch(s) // the command search box (M05-F12)
	native.EndMainMenuBar()
}

// drawHelpMenu is the Help menu: product documentation (F1) and a manual update check.
// "Check for Updates" requests the (network) check, which the head runs off-thread and
// reports back through the Software Update window — so it works even while the
// application is running, per the manual-check requirement.
func drawHelpMenu(s *app.Session) {
	if native.BeginMenu("Help") {
		if native.MenuItem("Documentation") {
			_ = s.DisplayHelpTopic("", "")
		}
		native.Separator()
		if native.MenuItem("Check for Updates") {
			s.RequestUpdateCheck()
		}
		if native.MenuItem("Report Bug") { // opens the report window; submit happens off-thread
			reportBugUI.open = true
		}
		native.EndMenu()
	}
}

func drawFileMenu(s *app.Session) {
	if native.BeginMenu("File") {
		if native.MenuItem("New Part") {
			_, _ = s.NewPart()
		}
		if native.MenuItem("New Assembly") { // #762 — start an assembly to place and constrain components
			_, _ = s.NewAssembly()
		}
		if native.MenuItem("New Drawing") { // M14-F01 — start a drawing of a model's views
			_, _ = s.NewDrawing()
		}
		if native.MenuItem("Open") {
			openViaHookOrDialog(s)
		}
		drawOpenRecentMenu(s)
		if native.MenuItem("Save") {
			saveActive(s)
		}
		if native.MenuItem("Save As") {
			saveAsViaHookOrDialog(s)
		}
		native.Separator()
		if native.MenuItem("Import") { // STL / OBJ / 3MF / STEP → an imported body
			fileModal.openFor(dialogImport)
		}
		if native.MenuItem("Export") { // part bodies → STL / OBJ / 3MF / STEP
			fileModal.openFor(dialogExport)
		}
		native.EndMenu()
	}
}

// openViaHookOrDialog consults the FileOpenDialog hook before presenting the
// explorer (M04-F05): an add-in that supplies a path replaces the dialog.
func openViaHookOrDialog(s *app.Session) {
	if path, ok := s.HookFileOpenDialog(); ok {
		applyFileAction(s, fileAction{Kind: dialogOpen, Path: path})
		return
	}
	fileModal.openFor(dialogOpen)
}

// saveAsViaHookOrDialog consults the FileSaveAsDialog hook before presenting
// the explorer; a supplied destination saves directly.
func saveAsViaHookOrDialog(s *app.Session) {
	if path, ok := s.HookFileSaveAsDialog(false); ok {
		applyFileAction(s, fileAction{Kind: dialogSaveAs, Path: path})
		return
	}
	armSaveAs(s)
}

// drawOpenRecentMenu lists the session's recent documents (File ▸ Open Recent);
// each entry opens behind the vetoable FileOpenFromMRU hook.
func drawOpenRecentMenu(s *app.Session) {
	recent := s.RecentDocuments()
	if len(recent) == 0 {
		return
	}
	if !native.BeginMenu("Open Recent") {
		return
	}
	for _, path := range recent {
		if native.MenuItem(path) {
			_, _ = s.OpenDocumentFromMRU(path)
		}
	}
	native.EndMenu()
}

func drawEditMenu(s *app.Session) {
	if native.BeginMenu("Edit") {
		// Undo/Redo name the step they act on (Inventor's "Undo Extrude") and grey out
		// when the stream cursor is at an end. Keyboard equivalents: Ctrl+Z / Ctrl+Y.
		if native.MenuItemEx(undoLabel("Undo", s.UndoLabel()), "Ctrl+Z", s.CanUndo()) {
			_ = s.Undo()
		}
		if native.MenuItemEx(undoLabel("Redo", s.RedoLabel()), "Ctrl+Y", s.CanRedo()) {
			_ = s.Redo()
		}
		native.Separator()
		// History Browser complements undo/redo for long histories and multi-document
		// assemblies: jump to any point on each document's timeline, side by side.
		if native.MenuItem(checkLabel("History Browser", s.HistoryBrowserOpen())) {
			s.ToggleHistoryBrowser()
		}
		native.Separator()
		if native.MenuItem("Cancel Tool (Esc)") && s.ActiveTool() != nil {
			s.CancelTool()
		}
		native.EndMenu()
	}
}

func drawToolsMenu(s *app.Session) {
	if native.BeginMenu("Tools") {
		if native.MenuItem("Materials") {
			showMaterials = !showMaterials
		}
		if native.MenuItem("Preferences") {
			showPreferences = !showPreferences
		}
		if native.MenuItem("Document Settings — Units") { // #146: per-document units & precision
			s.OpenUnitsSettings()
		}
		if native.MenuItem("Customize Keyboard") { // M05-F17: rebind shortcuts / aliases
			s.OpenKeymapEditor()
		}
		if native.MenuItem("Command Input") { // M05-F17: type an alias to run a command
			s.BeginCommandInput()
		}
		if native.MenuItem("Get Add-Ins…") { // #1164: browse/install/update add-ins from the catalogue
			OpenAddInCatalogue(s)
		}
		if native.MenuItem(checkLabel("Command Window", s.CommandWindowOpen())) { // M26 F04: docked REPL
			s.ToggleCommandWindow()
		}
		if native.MenuItem(checkLabel("Selection Filter", s.SelectionFilterWindowOpen())) { // #1222
			s.ToggleSelectionFilterWindow()
		}
		native.Separator()
		if native.MenuItem(checkLabel("Normal Debug (front green / back red)", normalDebugOn)) {
			normalDebugOn = !normalDebugOn
		}
		if native.MenuItem("Save Viewport PNG") {
			screenshotRequested = true
		}
		native.EndMenu()
	}
}

// checkLabel prefixes a menu label with a check mark when the toggle is on (Dear ImGui's
// MenuItem has no built-in checkbox variant here).
func checkLabel(label string, on bool) string {
	if on {
		return "[x] " + label
	}
	return "[ ] " + label
}

// undoLabel builds an Edit-menu label that names the step undo/redo would act on (e.g.
// "Undo Extrude"), falling back to the bare verb when the stream cursor is at an end.
func undoLabel(verb, step string) string {
	if step == "" {
		return verb
	}
	return verb + " " + step
}
