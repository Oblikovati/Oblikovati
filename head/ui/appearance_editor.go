//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/material"
)

// drawAppearanceTabContent renders the appearance selector, actions, and PBR editor inside
// the Materials window (distinct from the theme Appearance tab in appearance_tab.go).
func drawAppearanceTabContent(s *app.Session) {
	lib := s.Materials()
	if _, ok := lib.Appearance(selectedAppearance); !ok && len(lib.Appearances()) > 0 {
		selectedAppearance = lib.Appearances()[0].ID()
	}
	drawAppearanceSelector(s)
	a, ok := lib.Appearance(selectedAppearance)
	if !ok {
		return
	}
	drawAppearanceActions(s, a)
	native.Separator()
	editAppearancePBR(s, a)
}

// drawAppearanceSelector renders the appearance dropdown.
func drawAppearanceSelector(s *app.Session) {
	current := selectedAppearance
	if a, ok := s.Materials().Appearance(selectedAppearance); ok {
		current = a.DisplayName()
	}
	if !native.BeginCombo("Appearance", current) {
		return
	}
	for _, a := range s.Materials().Appearances() {
		if native.Selectable(a.DisplayName(), a.ID() == selectedAppearance) {
			selectedAppearance = a.ID()
		}
	}
	native.EndCombo()
}

// drawAppearanceActions renders duplicate + assign-to-part for the selected appearance.
func drawAppearanceActions(s *app.Session, a *material.Appearance) {
	native.InputText("Copy name", apprNameBuf[:])
	native.SameLine()
	if native.Button("Duplicate##appr") {
		if dup, err := s.DuplicateAppearance(a.ID(), bufString(apprNameBuf[:])); err == nil {
			selectedAppearance = dup.ID()
			clearBuf(apprNameBuf[:])
		}
	}
	if native.Button("Assign to part") {
		_ = s.AssignAppearance(app.ScopePart, "", a.ID())
	}
}

// editAppearancePBR renders the metallic-roughness controls (disabled for built-ins). Edits
// flow to the session immediately, so the viewport recolors next frame.
func editAppearancePBR(s *app.Session, a *material.Appearance) {
	editable := a.Source().Editable()
	native.BeginDisabled(!editable)
	spec := a.Spec()
	changed := false
	albedo := spec.Albedo.Array()
	if native.ColorEdit4("Albedo", &albedo) {
		spec.Albedo = rgbaOf(albedo)
		changed = true
	}
	changed = native.InputFloat("Metallic", &spec.Metallic) || changed
	changed = native.InputFloat("Roughness", &spec.Roughness) || changed
	changed = native.InputFloat("Opacity", &spec.Opacity) || changed
	emissive := spec.Emissive.Array()
	if native.ColorEdit4("Emissive", &emissive) {
		spec.Emissive = rgbaOf(emissive)
		changed = true
	}
	native.EndDisabled()
	if changed && editable {
		s.UpdateAppearance(a.ID(), spec)
	}
}

// rgbaOf converts an ImGui [4]float32 color into a material color.
func rgbaOf(c [4]float32) material.Rgba {
	return material.Rgba{R: c[0], G: c[1], B: c[2], A: c[3]}
}
