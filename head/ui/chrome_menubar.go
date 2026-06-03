//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
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
		native.EndMenu()
	}
	if native.BeginMenu("Edit") {
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
