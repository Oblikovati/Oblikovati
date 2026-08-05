//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The model GD&T flow in the head: while the Feature Control Frame or Datum Feature tool runs, a
// modeless property panel shows the annotated-geometry chip and the mode's inputs, then
// OK/Cancel. These annotate MODEL geometry — the Drawing tab's frames annotate a view (#2049).

// modelToleranceUI holds the panel's fields across frames. The two text buffers are fixed-size
// because native.InputText writes into a caller-owned byte slice.
var modelToleranceUI = struct {
	value  float32
	datums [64]byte
	label  [8]byte
	seeded *app.ModelToleranceTool
}{value: 0.1}

// drawModelToleranceDialog shows the GD&T property panel while either tool is active.
func drawModelToleranceDialog(s *app.Session) {
	mt := s.ActiveModelTolerance()
	if mt == nil {
		modelToleranceUI.seeded = nil
		return
	}
	if modelToleranceUI.seeded != mt {
		seedModelToleranceUI(mt)
	}
	dialogSizeOnce(360, 250)
	if native.Begin(mt.Name()) {
		drawFeatureBreadcrumb(mt.Name(), "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Geometry", "tolerance-geometry",
				pickChipText(mt.GeometryPicked(), "1 Reference", "Select Face or Edge"),
				mt.GeometryPicked(), "Click the model face or edge to annotate", mt.ClearGeometry)
		}
		if propertySection("Tolerance") && !mt.DatumMode() {
			drawFrameCharacteristicRow(mt)
			lengthCmRow(s, "Tolerance", "tolerance-value", &modelToleranceUI.value)
			mt.SetValue(float64(modelToleranceUI.value))
			drawFrameDatumsRow(mt)
		} else if mt.DatumMode() {
			drawDatumLabelRow(mt)
		}
		native.Separator()
		drawCommitCancelButtons(s, mt.CanCommit())
	}
	native.End()
}

// seedModelToleranceUI loads the panel buffers from the tool the first frame it appears.
func seedModelToleranceUI(mt *app.ModelToleranceTool) {
	modelToleranceUI.value = float32(mt.Value())
	setBuf(modelToleranceUI.datums[:], mt.Datums())
	setBuf(modelToleranceUI.label[:], mt.Label())
	modelToleranceUI.seeded = mt
}

// drawDatumLabelRow renders the datum-feature letter.
func drawDatumLabelRow(mt *app.ModelToleranceTool) {
	propertyRow("Label")
	native.SetNextItemWidth(propertyFieldWidth)
	native.InputText("##datum-label", modelToleranceUI.label[:])
	mt.SetLabel(bufString(modelToleranceUI.label[:]))
}

// drawFrameCharacteristicRow renders the geometric-characteristic combo.
func drawFrameCharacteristicRow(mt *app.ModelToleranceTool) {
	if i := propertyComboRow("Characteristic", "tolerance-characteristic",
		app.GeometricCharacteristicOptions(), mt.CharacteristicIndex()); i >= 0 {
		mt.SetCharacteristicIndex(i)
	}
}

// drawFrameDatumsRow renders the frame's datum references, typed as "A,B".
func drawFrameDatumsRow(mt *app.ModelToleranceTool) {
	propertyRow("Datums")
	native.SetNextItemWidth(propertyFieldWidth)
	native.InputText("##tolerance-datums", modelToleranceUI.datums[:])
	mt.SetDatums(bufString(modelToleranceUI.datums[:]))
}
