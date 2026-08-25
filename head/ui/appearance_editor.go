//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/material"
)

// Appearance editor (M46: OpenPBR Appearance Consolidation) — the single appearance
// panel, covering every parameter group parametrization.md.html defines: Base/Specular/
// Transmission/Subsurface/Coat/Fuzz/ThinFilm/Emission/Geometry. Groups are stacked
// SeparatorText sections (editMaterialProps' pattern), not nested ImGui tabs — a
// non-selected ImGui tab's content function is never called, so nested tabs would make
// the "every group renders" smoke test (this PBI's own acceptance criterion) unable to
// actually exercise most groups without extra tab-selection plumbing this package
// doesn't otherwise need. Geometry's four optional texture-mapped vector overrides
// (normal/tangent/coatNormal/coatTangent) are not editable here — textured OpenPBR
// inputs are deferred per the migration plan's scope (ADR-0022 §2).

// appearanceEditorSession is the narrow view of *app.Session this editor needs (audit
// I5, the arrowSession/cloudBounder pattern — head/ui's session-coupling ratchet).
// Mirrors realistic_render.go's realisticSession; *app.Session satisfies it implicitly.
type appearanceEditorSession interface {
	Materials() *material.Library
	DuplicateAppearance(baseID, name string) (*material.Appearance, error)
	UpdateAppearance(id string, spec material.AppearanceSpec)
	AssignAppearance(scope, key, appearanceID string) error
}

// drawAppearanceTabContent renders the appearance selector, actions, and the full
// parameter editor inside the Materials window's "Appearances" tab.
func drawAppearanceTabContent(s appearanceEditorSession) {
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
func drawAppearanceSelector(s appearanceEditorSession) {
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
func drawAppearanceActions(s appearanceEditorSession, a *material.Appearance) {
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

// editAppearancePBR renders every parameter group (disabled for built-ins). Edits
// flow to the session immediately, so the viewport recolors next frame.
func editAppearancePBR(s appearanceEditorSession, a *material.Appearance) {
	editable := a.Source().Editable()
	native.BeginDisabled(!editable)
	spec := a.Spec()
	changed := editOpenPBRBase(&spec.Base)
	changed = editOpenPBRSpecular(&spec.Specular) || changed
	changed = editOpenPBRTransmission(&spec.Transmission) || changed
	changed = editOpenPBRSubsurface(&spec.Subsurface) || changed
	changed = editOpenPBRCoat(&spec.Coat) || changed
	changed = editOpenPBRFuzz(&spec.Fuzz) || changed
	changed = editOpenPBRThinFilm(&spec.ThinFilm) || changed
	changed = editOpenPBREmission(&spec.Emission) || changed
	changed = editOpenPBRGeometry(&spec.Geometry) || changed
	native.EndDisabled()
	if changed && editable {
		s.UpdateAppearance(a.ID(), spec)
	}
}

func editOpenPBRBase(g *material.OpenPBRBase) bool {
	native.SeparatorText("Base")
	changed := native.InputFloat("Base Weight", &g.Weight)
	changed = editColor3("Base Color", &g.Color) || changed
	changed = native.InputFloat("Base Metalness", &g.Metalness) || changed
	changed = native.InputFloat("Base Diffuse Roughness", &g.DiffuseRoughness) || changed
	return changed
}

func editOpenPBRSpecular(g *material.OpenPBRSpecular) bool {
	native.SeparatorText("Specular")
	changed := native.InputFloat("Specular Weight", &g.Weight)
	changed = editColor3("Specular Color", &g.Color) || changed
	changed = native.InputFloat("Specular Roughness", &g.Roughness) || changed
	changed = native.InputFloat("Specular Roughness Anisotropy", &g.RoughnessAnisotropy) || changed
	changed = native.InputFloat("Specular IOR", &g.IOR) || changed
	return changed
}

func editOpenPBRTransmission(g *material.OpenPBRTransmission) bool {
	native.SeparatorText("Transmission")
	changed := native.InputFloat("Transmission Weight", &g.Weight)
	changed = editColor3("Transmission Color", &g.Color) || changed
	changed = native.InputFloat("Transmission Depth", &g.Depth) || changed
	changed = editColor3("Transmission Scatter", &g.Scatter) || changed
	changed = native.InputFloat("Transmission Scatter Anisotropy", &g.ScatterAnisotropy) || changed
	changed = native.InputFloat("Dispersion Scale", &g.DispersionScale) || changed
	changed = native.InputFloat("Dispersion Abbe Number", &g.DispersionAbbeNumber) || changed
	return changed
}

func editOpenPBRSubsurface(g *material.OpenPBRSubsurface) bool {
	native.SeparatorText("Subsurface")
	changed := native.InputFloat("Subsurface Weight", &g.Weight)
	changed = editColor3("Subsurface Color", &g.Color) || changed
	changed = native.InputFloat("Subsurface Radius", &g.Radius) || changed
	changed = editColor3("Subsurface Radius Scale", &g.RadiusScale) || changed
	changed = native.InputFloat("Subsurface Scatter Anisotropy", &g.ScatterAnisotropy) || changed
	return changed
}

func editOpenPBRCoat(g *material.OpenPBRCoat) bool {
	native.SeparatorText("Coat")
	changed := native.InputFloat("Coat Weight", &g.Weight)
	changed = editColor3("Coat Color", &g.Color) || changed
	changed = native.InputFloat("Coat Roughness", &g.Roughness) || changed
	changed = native.InputFloat("Coat Roughness Anisotropy", &g.RoughnessAnisotropy) || changed
	changed = native.InputFloat("Coat IOR", &g.IOR) || changed
	changed = native.InputFloat("Coat Darkening", &g.Darkening) || changed
	return changed
}

func editOpenPBRFuzz(g *material.OpenPBRFuzz) bool {
	native.SeparatorText("Fuzz")
	changed := native.InputFloat("Fuzz Weight", &g.Weight)
	changed = editColor3("Fuzz Color", &g.Color) || changed
	changed = native.InputFloat("Fuzz Roughness", &g.Roughness) || changed
	return changed
}

func editOpenPBRThinFilm(g *material.OpenPBRThinFilm) bool {
	native.SeparatorText("Thin Film")
	changed := native.InputFloat("Thin Film Weight", &g.Weight)
	changed = native.InputFloat("Thin Film Thickness (µm)", &g.Thickness) || changed
	changed = native.InputFloat("Thin Film IOR", &g.IOR) || changed
	return changed
}

func editOpenPBREmission(g *material.OpenPBREmission) bool {
	native.SeparatorText("Emission")
	changed := native.InputFloat("Emission Luminance (nits)", &g.Luminance)
	changed = editColor3("Emission Color", &g.Color) || changed
	return changed
}

// editOpenPBRGeometry edits Opacity/ThinWalled only — Normal/Tangent/CoatNormal/
// CoatTangent are texture-mapped overrides, out of scope until textured OpenPBR inputs
// land (see this file's doc comment).
func editOpenPBRGeometry(g *material.OpenPBRGeometry) bool {
	native.SeparatorText("Geometry")
	changed := native.InputFloat("Opacity", &g.Opacity)
	changed = native.Checkbox("Thin Walled", &g.ThinWalled) || changed
	return changed
}

// editColor3 draws a 3-channel color field for an OpenPBR Color3 (unbounded ACEScg, no
// alpha) via ColorEdit4 with a fixed alpha of 1, discarded on write-back.
func editColor3(label string, c *material.Color3) bool {
	v := [4]float32{c.R, c.G, c.B, 1}
	if native.ColorEdit4(label, &v) {
		c.R, c.G, c.B = v[0], v[1], v[2]
		return true
	}
	return false
}
