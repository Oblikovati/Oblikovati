//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// The feature-edit flow in the head: double-clicking a feature in the model browser (or its
// "Edit" context-menu entry) opens this dialog. It shows one field per editable scalar
// parameter (distance, radius, angle, diameter …) AND a row per geometric reference slot
// (the fillet's edges, a hole's placement face …) with Select (re-pick in the viewport) and
// Clear buttons. Editing is an interactive tool, so OK = s.OK() (commit), Cancel =
// s.CancelTool() (restore). Each field shows the document's unit for that quantity.

// featureEditUI holds the dialog's per-parameter field values across frames and whether it was
// open last frame (so it seeds the fields once when the edit opens).
var featureEditUI struct {
	values []float32
	open   bool
}

// drawFeatureEditDialog shows the parameter + reference editor while a feature edit is open.
func drawFeatureEditDialog(s *app.Session) {
	if !s.IsEditingFeature() {
		featureEditUI.open = false
		return
	}
	nParams := s.EditFeatureParamCount()
	nRefs := s.EditFeatureRefSlotCount()
	refreshFeatureEditUI(s, nParams)
	native.SetNextWindowSize(320, float32(108+nParams*30+nRefs*30))
	if native.Begin("Edit Feature") {
		native.Text(s.EditingFeatureName())
		drawEditParams(s, nParams)
		drawEditRefSlots(s, nRefs)
		drawFeatureEditButtons(s)
	}
	native.End()
}

func refreshFeatureEditUI(s *app.Session, nParams int) {
	if featureEditUI.open {
		return
	}
	featureEditUI.values = make([]float32, nParams)
	for i := 0; i < nParams; i++ {
		featureEditUI.values[i] = float32(s.EditFeatureParamValue(i))
	}
	featureEditUI.open = true
}

func drawFeatureEditButtons(s *app.Session) {
	if native.Button("OK") {
		if err := s.OK(); err == nil { // a sick result keeps the dialog open
			featureEditUI.open = false
		}
	}
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelTool()
		featureEditUI.open = false
	}
}

// drawEditParams renders one field per editable scalar (an integer field for whole-number
// inputs like a pattern count), syncing it to the feature.
func drawEditParams(s *app.Session, n int) {
	for i := 0; i < n && i < len(featureEditUI.values); i++ {
		label := s.EditFeatureParamLabel(i)
		if u := s.EditFeatureParamUnitName(i); u != "" {
			label += " (" + u + ")"
		}
		native.Text(label)
		if s.EditFeatureParamIsInteger(i) {
			iv := int32(featureEditUI.values[i] + 0.5)
			native.InputInt(fmt.Sprintf("##edit-feature-param-%d", i), &iv)
			featureEditUI.values[i] = float32(iv)
		} else {
			native.InputFloat(fmt.Sprintf("##edit-feature-param-%d", i), &featureEditUI.values[i])
		}
		s.SetEditFeatureParamValue(i, float64(featureEditUI.values[i])) // keep the feature in sync
	}
}

// drawEditRefSlots renders one row per reference slot: a count, a Select button that arms the
// slot for viewport picking (highlighted while armed), and a Clear button.
func drawEditRefSlots(s *app.Session, n int) {
	for i := 0; i < n; i++ {
		native.Text(fmt.Sprintf("%s: %d selected", s.EditFeatureRefSlotLabel(i), s.EditFeatureRefSlotRefCount(i)))
		native.SameLine()
		selectLabel := "Select"
		if s.EditFeatureRefSlotArmed(i) {
			selectLabel = "Selecting… (click geometry)"
		}
		if native.Button(fmt.Sprintf("%s##edit-ref-select-%d", selectLabel, i)) {
			s.EditFeatureArmRefSlot(i)
		}
		if s.EditFeatureRefSlotClearable(i) {
			native.SameLine()
			if native.Button(fmt.Sprintf("Clear##edit-ref-clear-%d", i)) {
				s.EditFeatureClearRefSlot(i)
			}
		}
	}
}
