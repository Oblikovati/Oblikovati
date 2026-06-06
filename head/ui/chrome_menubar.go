//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// drawMenuBar renders the top menu bar. Only the few items that map to real session
// verbs are wired; the rest grow with the feature set.
func drawMenuBar(s *app.Session) string {
	if !native.BeginMainMenuBar() {
		return ""
	}
	if native.BeginMenu("File") {
		if native.MenuItem("New Part") {
			_, _ = s.NewPart()
		}
		if native.MenuItem("Open") {
			fileModal.openFor(dialogOpen)
		}
		if native.MenuItem("Save") {
			saveActive(s)
		}
		if native.MenuItem("Save As") {
			fileModal.openFor(dialogSaveAs)
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
		if native.MenuItem("Cancel Tool (Esc)") && s.ActiveTool() != nil {
			s.CancelTool()
		}
		native.EndMenu()
	}
	if native.BeginMenu("Tools") {
		if native.MenuItem("Materials") {
			showMaterials = !showMaterials
		}
		if native.MenuItem("Preferences") {
			showPreferences = !showPreferences
		}
		native.EndMenu()
	}
	native.EndMainMenuBar()
	return ""
}

// undoLabel builds an Edit-menu label that names the step undo/redo would act on (e.g.
// "Undo Extrude"), falling back to the bare verb when the stream cursor is at an end.
func undoLabel(verb, step string) string {
	if step == "" {
		return verb
	}
	return verb + " " + step
}
