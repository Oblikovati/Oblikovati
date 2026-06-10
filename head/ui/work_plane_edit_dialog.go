//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Editing / redefining a placed work plane (browser double-click): an offset plane edits its
// distance, a line-plane-angle plane edits its angle, and a reference-built plane (three
// points, two planes, a tangent face, …) re-picks its references. The dialog mirrors the
// feature-edit dialog — one field per editable scalar, one row per reference slot with a
// Select button that arms it for viewport/browser picking. While it is open the model is
// rolled back to the plane (edit-scope, issue #132). Scalars are in the document's unit.

// workPlaneEditUI holds the dialog's per-scalar field values across frames, keyed by the
// plane being edited — so switching directly from one plane's edit to another's reseeds the
// fields (a bare "was open" bool would carry the first plane's stale values over).
var workPlaneEditUI struct {
	values  []float32
	editing string // EditPlaneName of the plane the fields were seeded from ("" = none)
}

// drawWorkPlaneEditDialog shows the scalar + reference editor while a work-plane edit is open.
func drawWorkPlaneEditDialog(s *app.Session) {
	if !s.IsEditingWorkPlane() {
		workPlaneEditUI.editing = ""
		return
	}
	nScalars := s.EditPlaneScalarCount()
	nRefs := s.EditPlaneRefSlotCount()
	refreshWorkPlaneEditUI(s, nScalars)
	native.SetNextWindowSize(320, float32(104+nScalars*30+nRefs*30))
	if native.Begin("Edit Work Plane") {
		native.Text(s.EditPlaneName())
		drawWorkPlaneScalars(s, nScalars)
		drawWorkPlaneRefSlots(s, nRefs)
		drawWorkPlaneEditButtons(s)
	}
	native.End()
}

func refreshWorkPlaneEditUI(s *app.Session, nScalars int) {
	if workPlaneEditUI.editing == s.EditPlaneName() {
		return
	}
	workPlaneEditUI.values = make([]float32, nScalars)
	for i := 0; i < nScalars; i++ {
		workPlaneEditUI.values[i] = float32(s.EditPlaneScalarValue(i))
	}
	workPlaneEditUI.editing = s.EditPlaneName()
}

// drawWorkPlaneScalars renders one field per editable scalar (offset/angle), syncing it to the
// plane each frame.
func drawWorkPlaneScalars(s *app.Session, n int) {
	for i := 0; i < n && i < len(workPlaneEditUI.values); i++ {
		label := s.EditPlaneScalarLabel(i)
		if u := s.EditPlaneScalarUnitName(i); u != "" {
			label += " (" + u + ")"
		}
		native.Text(label)
		native.InputFloat(fmt.Sprintf("##edit-plane-scalar-%d", i), &workPlaneEditUI.values[i])
		s.SetEditPlaneScalarValue(i, float64(workPlaneEditUI.values[i])) // keep the plane in sync
	}
}

// drawWorkPlaneRefSlots renders one row per reference slot: a label and a Select button that
// arms the slot for viewport/browser picking (highlighted while armed).
func drawWorkPlaneRefSlots(s *app.Session, n int) {
	for i := 0; i < n; i++ {
		native.Text(s.EditPlaneRefSlotLabel(i))
		native.SameLine()
		selectLabel := "Select"
		if s.EditPlaneRefSlotArmed(i) {
			selectLabel = "Selecting… (click geometry)"
		}
		if native.Button(fmt.Sprintf("%s##edit-plane-ref-%d", selectLabel, i)) {
			s.EditPlaneArmRefSlot(i)
		}
	}
}

func drawWorkPlaneEditButtons(s *app.Session) {
	if native.Button("OK") {
		if err := s.OK(); err == nil { // a sick result keeps the dialog open
			workPlaneEditUI.editing = ""
		}
	}
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelTool()
		workPlaneEditUI.editing = ""
	}
}
