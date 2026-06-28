//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"strconv"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The generic feature editor: features whose creation panel can round-trip their
// definition re-open that panel instead (app.BeginEditFeature dispatch); this dialog
// serves the rest (patterns, mirror, rib, emboss). It follows the same property-panel
// schema — an Input Geometry section of armable reference-slot chips and a Behavior
// section of scalar rows — so editing looks the same everywhere. OK = s.OK() (commit),
// Cancel = s.CancelTool() (restore). Each field shows the document's unit.

// featureEditUI holds the dialog's per-parameter field values across frames, keyed by the
// feature being edited — so switching directly from one feature's edit to another's reseeds
// the fields (a bare "was open" bool would carry the first feature's stale values over).
var featureEditUI struct {
	values  []float32 // integer fields (pattern counts) stay numeric
	texts   [][]byte  // scalar fields edit as text so they accept parameter expressions
	editing string    // EditingFeatureName of the feature the fields were seeded from ("" = none)
}

// editFieldBufLen bounds a parameter-expression field ("d0 + 12 mm" and the like).
const editFieldBufLen = 64

// drawFeatureEditDialog shows the parameter + reference editor while a generic feature
// edit is open.
func drawFeatureEditDialog(s *app.Session) {
	if !s.IsEditingFeature() {
		featureEditUI.editing = ""
		return
	}
	nParams := s.EditFeatureParamCount()
	nRefs := s.EditFeatureRefSlotCount()
	refreshFeatureEditUI(s, nParams)
	native.SetNextWindowSizeOnce(340, float32(150+nParams*28+nRefs*28))
	if native.Begin("Edit Feature") {
		drawFeatureBreadcrumb(s.EditingFeatureName(), "")
		drawEditRefSlots(s, nRefs)
		drawEditParams(s, nParams)
		native.Separator()
		drawFeatureEditButtons(s)
	}
	native.End()
}

func refreshFeatureEditUI(s *app.Session, nParams int) {
	if featureEditUI.editing == s.EditingFeatureName() {
		return
	}
	featureEditUI.values = make([]float32, nParams)
	featureEditUI.texts = make([][]byte, nParams)
	for i := 0; i < nParams; i++ {
		featureEditUI.values[i] = float32(s.EditFeatureParamValue(i))
		featureEditUI.texts[i] = make([]byte, editFieldBufLen)
		setBuf(featureEditUI.texts[i], paramSeedText(s.EditFeatureParamValue(i), s.EditFeatureParamUnitName(i)))
	}
	featureEditUI.editing = s.EditingFeatureName()
}

func drawFeatureEditButtons(s *app.Session) {
	if native.Button("OK") {
		if err := s.OK(); err == nil { // a sick result keeps the dialog open
			featureEditUI.editing = ""
		}
	}
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelTool()
		featureEditUI.editing = ""
	}
}

// drawEditRefSlots is the Input Geometry section: one armable chip per reference slot.
// Clicking a chip arms its slot for viewport picking; × clears a clearable slot.
func drawEditRefSlots(s *app.Session, n int) {
	if n == 0 || !propertySection("Input Geometry") {
		return
	}
	for i := 0; i < n; i++ {
		label := s.EditFeatureRefSlotLabel(i)
		propertyRow(label)
		text := editSlotChipText(s.EditFeatureRefSlotRefCount(i), label)
		arm, clear := propertyArmableSlotChip(fmt.Sprintf("edit-ref-%d", i), text,
			s.EditFeatureRefSlotRefCount(i) > 0, s.EditFeatureRefSlotArmed(i), s.EditFeatureRefSlotClearable(i))
		if arm {
			s.EditFeatureArmRefSlot(i)
		}
		if clear {
			s.EditFeatureClearRefSlot(i)
		}
	}
}

// editSlotChipText is the slot chip caption: the reference panel's generic "N Selected"
// once filled, the slot's own prompt while empty.
func editSlotChipText(count int, label string) string {
	if count == 0 {
		return "Select " + label
	}
	return strconv.Itoa(count) + " Selected"
}

// drawEditParams is the Behavior section: one row per editable scalar (an integer field
// for whole-number inputs like a pattern count), synced to the feature each frame.
func drawEditParams(s *app.Session, n int) {
	if n == 0 || !propertySection("Behavior") {
		return
	}
	for i := 0; i < n && i < len(featureEditUI.values); i++ {
		drawEditParamRow(s, i)
	}
}

// drawEditParamRow renders one scalar row: label, value field, unit suffix. Integer
// fields stay a numeric spinner; scalar fields edit as text and commit through the
// parameter evaluator (SetEditFeatureParamText), so a field accepts a parameter name
// or formula ("bore_r + 2 mm") as well as a literal (Oblikovati.API#187, UI side). An
// expression that does not yet parse (mid-typing) is simply not committed.
func drawEditParamRow(s *app.Session, i int) {
	propertyRow(s.EditFeatureParamLabel(i))
	native.SetNextItemWidth(propertyFieldWidth)
	if s.EditFeatureParamIsInteger(i) {
		iv := int32(featureEditUI.values[i] + 0.5)
		if native.InputInt(fmt.Sprintf("##edit-feature-param-%d", i), &iv) {
			featureEditUI.values[i] = float32(iv)
			s.SetEditFeatureParamValue(i, float64(iv))
		}
	} else if native.InputText(fmt.Sprintf("##edit-feature-param-%d", i), featureEditUI.texts[i]) {
		_ = s.SetEditFeatureParamText(i, bufString(featureEditUI.texts[i]))
	}
	// The unit is seeded INTO the field's text ("10 mm", paramSeedText) rather than painted beside
	// it as a label (#1519): an expression field carries its own dimension, so there is no separate
	// unit label here. An integer field (a count) has no unit and stays a plain spinner.
}

// paramSeedText is the initial text for an editable scalar's expression field: the value in the
// document unit with the unit appended ("10 mm"), so the dimension is part of the editable token. A
// unitless field (an integer count, or a ratio) is seeded as the bare number.
func paramSeedText(value float64, unit string) string {
	t := strconv.FormatFloat(value, 'g', -1, 64)
	if unit != "" {
		t += " " + unit
	}
	return t
}
