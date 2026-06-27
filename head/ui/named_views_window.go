//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// namedViewNameBuf backs the "save view" name field (NUL-terminated, ImGui-owned).
var namedViewNameBuf [64]byte

// drawNamedViewsWindow renders the Named Views panel while it is open (M16-F03 #404): a name
// field + Save button to capture the current camera, and a row per saved view with Restore and
// Delete. The capture/restore/delete go through the session (the same surface the API uses).
func drawNamedViewsBody(s *app.Session) {
	native.InputText("Name", namedViewNameBuf[:])
	native.SameLine()
	if native.Button("Save") {
		if name := bufString(namedViewNameBuf[:]); name != "" {
			if _, err := s.CaptureNamedView(name); err == nil {
				clearBuf(namedViewNameBuf[:])
			}
		}
	}
	native.Separator()
	drawNamedViewRows(s)
	native.Separator()
	if native.Button("Done") {
		s.CloseNamedViewsPanel()
	}
}

// drawNamedViewRows lists each saved named view with Restore and Delete actions.
func drawNamedViewRows(s *app.Session) {
	views := s.NamedViews()
	if len(views) == 0 {
		native.Text("No saved views — type a name and Save.")
		return
	}
	for _, nv := range views {
		native.Text(nv.Name)
		native.SameLine()
		if native.Button(fmt.Sprintf("Restore##%s", nv.Name)) {
			_ = s.RestoreNamedView(nv.Name)
		}
		native.SameLine()
		if native.Button(fmt.Sprintf("Delete##%s", nv.Name)) {
			_ = s.DeleteNamedView(nv.Name)
		}
	}
}
