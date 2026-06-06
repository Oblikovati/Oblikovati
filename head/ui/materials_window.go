//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati/app"
	"oblikovati/head/internal/native"
	"oblikovati/model/material"
)

// showMaterials toggles the Materials window (Tools ▸ Materials). UI state, so it lives in
// the head. The selection ids persist across frames so the editor keeps its target.
var (
	showMaterials      bool
	selectedMaterialID string
	selectedAppearance string
	matNameBuf         [64]byte
	apprNameBuf        [64]byte
)

// drawMaterialsWindow renders the Materials browser/editor: pick or duplicate a material or
// appearance, edit a project-scoped copy (built-ins are read-only), assign it to the active
// part, and read the part's physical properties. Edits flow straight to the session, so the
// viewport recolors next frame (ADR-0022).
func drawMaterialsWindow(s *app.Session) {
	if !showMaterials {
		return
	}
	native.SetNextWindowSizeOnce(420, 560)
	if native.Begin("Materials") && native.BeginTabBar("##materials-tabs") {
		if native.BeginTabItem("Materials") {
			drawMaterialTab(s)
			native.EndTabItem()
		}
		if native.BeginTabItem("Appearances") {
			drawAppearanceTabContent(s)
			native.EndTabItem()
		}
		if native.BeginTabItem("Physical") {
			drawPhysicalReadout(s)
			native.EndTabItem()
		}
		native.EndTabBar()
	}
	native.End()
}

// drawMaterialTab renders the material selector, actions, and property editor.
func drawMaterialTab(s *app.Session) {
	lib := s.Materials()
	if _, ok := lib.Material(selectedMaterialID); !ok && len(lib.Materials()) > 0 {
		selectedMaterialID = lib.Materials()[0].ID()
	}
	drawMaterialSelector(s)
	m, ok := lib.Material(selectedMaterialID)
	if !ok {
		return
	}
	drawMaterialActions(s, m)
	native.Separator()
	editMaterialProps(s, m)
}

// drawMaterialSelector renders the material dropdown.
func drawMaterialSelector(s *app.Session) {
	current := selectedMaterialID
	if m, ok := s.Materials().Material(selectedMaterialID); ok {
		current = m.DisplayName()
	}
	if !native.BeginCombo("Material", current) {
		return
	}
	for _, m := range s.Materials().Materials() {
		if native.Selectable(m.DisplayName(), m.ID() == selectedMaterialID) {
			selectedMaterialID = m.ID()
		}
	}
	native.EndCombo()
}

// drawMaterialActions renders duplicate + assign-to-part for the selected material.
func drawMaterialActions(s *app.Session, m *material.Material) {
	native.InputText("Copy name", matNameBuf[:])
	native.SameLine()
	if native.Button("Duplicate##mat") {
		if dup, err := s.DuplicateMaterial(m.ID(), bufString(matNameBuf[:])); err == nil {
			selectedMaterialID = dup.ID()
			clearBuf(matNameBuf[:])
		}
	}
	if native.Button("Assign to part") {
		_ = s.AssignMaterial("", m.ID())
	}
}

// editMaterialProps renders the editable property fields (disabled for built-ins).
func editMaterialProps(s *app.Session, m *material.Material) {
	editable := m.Source().Editable()
	native.BeginDisabled(!editable)
	spec := m.Spec()
	changed := false
	dbl("Density (g/cm³)", &spec.Density, &changed)
	native.SeparatorText("Mechanical")
	dbl("Young's modulus (GPa)", &spec.Mechanical.YoungsModulus, &changed)
	dbl("Poisson's ratio", &spec.Mechanical.PoissonsRatio, &changed)
	dbl("Yield strength (MPa)", &spec.Mechanical.YieldStrength, &changed)
	dbl("Tensile strength (MPa)", &spec.Mechanical.UltimateTensileStrength, &changed)
	native.SeparatorText("Thermal")
	dbl("Conductivity (W/m·K)", &spec.Thermal.Conductivity, &changed)
	dbl("Expansion (1/K)", &spec.Thermal.ExpansionCoeff, &changed)
	dbl("Specific heat (J/kg·K)", &spec.Thermal.SpecificHeat, &changed)
	native.SeparatorText("Electrical")
	dbl("Resistivity (Ω·m)", &spec.Electrical.Resistivity, &changed)
	dbl("Rel. permittivity", &spec.Electrical.RelativePermittivity, &changed)
	native.EndDisabled()
	if changed && editable {
		s.UpdateMaterial(m.ID(), spec)
	}
}

// dbl draws a float64 field, OR-ing its change into changed.
func dbl(label string, v *float64, changed *bool) {
	if native.InputDouble(label, v) {
		*changed = true
	}
}

// drawPhysicalReadout shows the active part's computed mass properties.
func drawPhysicalReadout(s *app.Session) {
	props, ok := s.PhysicalProperties()
	if !ok {
		native.Text("No active part.")
		return
	}
	native.Text(fmt.Sprintf("Mass:     %.3f g", props.Mass))
	native.Text(fmt.Sprintf("Volume:   %.3f cm³", props.Volume))
	native.Text(fmt.Sprintf("Area:     %.3f cm²", props.Area))
	native.Text(fmt.Sprintf("Density:  %.3f g/cm³", props.Density))
	native.Text(fmt.Sprintf("Centroid: (%.2f, %.2f, %.2f) cm",
		props.Centroid[0], props.Centroid[1], props.Centroid[2]))
}
