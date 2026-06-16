//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// drawColorStylesWindow renders the Color Styles panel while it is open (M16-F02 #403/#408):
// it lists the document's color styles, applies a clicked style to the selected body (so the
// body renders in that style's color), and clears the selected body's style. The assignment
// goes through the session — the same surface the API uses.
func drawColorStylesWindow(s *app.Session) {
	if !s.ColorStylesPanelOpen() {
		return
	}
	native.SetNextWindowSizeOnce(300, 340)
	if native.Begin("Color Styles") {
		key, hasBody := s.SelectedBodyKey()
		if hasBody {
			native.Text(selectedStyleLabel(s, key))
		} else {
			native.Text("Select a body, then apply a style.")
		}
		native.Separator()
		drawColorStyleRows(s, key, hasBody)
		native.Separator()
		if hasBody && native.Button("Clear style") {
			s.ClearBodyColorStyle(key)
		}
		if native.Button("Done") {
			s.CloseColorStylesPanel()
		}
	}
	native.End()
}

// selectedStyleLabel describes the style currently on the selected body.
func selectedStyleLabel(s *app.Session, key string) string {
	if name, ok := s.BodyColorStyle(key); ok {
		return "Selected body: " + name
	}
	return "Selected body: (appearance)"
}

// drawColorStyleRows lists each color style with an Apply action (enabled only with a body
// selected).
func drawColorStyleRows(s *app.Session, key string, hasBody bool) {
	for _, cs := range s.ColorStyles() {
		native.Text(cs.Name)
		if hasBody {
			native.SameLine()
			if native.Button(fmt.Sprintf("Apply##%s", cs.Name)) {
				_ = s.AssignColorStyleToBody(key, cs.Name)
			}
		}
	}
}
