//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Editing / redefining a placed work plane (browser double-click): an offset plane edits
// its distance, a line-plane-angle plane edits its angle, and a reference-built plane
// (three points, two planes, a tangent face, …) re-picks its references. The dialog
// follows the property-panel schema like the feature editors — an Input Geometry section
// of armable reference-slot chips and a Behavior section of scalar rows. While it is
// open the model is rolled back to the plane (edit-scope, issue #132). Scalars are in
// the document's unit.

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
	native.SetNextWindowSizeOnce(340, float32(150+nScalars*28+nRefs*28))
	if native.Begin("Edit Work Plane") {
		drawFeatureBreadcrumb(s.EditPlaneName(), "")
		drawWorkPlaneRefSlots(s, nRefs)
		drawWorkPlaneScalars(s, nScalars)
		native.Separator()
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

// drawWorkPlaneScalars is the Behavior section: one row per editable scalar
// (offset/angle), synced to the plane each frame.
func drawWorkPlaneScalars(s *app.Session, n int) {
	if n == 0 || !propertySection("Behavior") {
		return
	}
	for i := 0; i < n && i < len(workPlaneEditUI.values); i++ {
		parameterFloatRow(s, s.EditPlaneScalarLabel(i), fmt.Sprintf("edit-plane-scalar-%d", i),
			planeScalarKind(s, i), "", &workPlaneEditUI.values[i])
		s.SetEditPlaneScalarValue(i, float64(workPlaneEditUI.values[i])) // keep the plane in sync
	}
}

// planeScalarKind maps an edit-plane scalar's unit (offset is a length, angle an angle) to the
// ParameterInput kind, so the field formats and evaluates it in the right dimension.
func planeScalarKind(s *app.Session, i int) paramFieldKind {
	switch s.EditPlaneScalarUnitName(i) {
	case s.AngleUnitName():
		return paramAngle
	case s.LengthUnitName():
		return paramLength
	default:
		return paramUnitless
	}
}

// drawWorkPlaneRefSlots is the Input Geometry section: one armable chip per reference
// slot — clicking it arms the slot for viewport/browser picking (plane slots have no
// clear: a plane always needs its references).
func drawWorkPlaneRefSlots(s *app.Session, n int) {
	if n == 0 || !propertySection("Input Geometry") {
		return
	}
	for i := 0; i < n; i++ {
		label := s.EditPlaneRefSlotLabel(i)
		propertyRow(label)
		arm, _ := propertyArmableSlotChip(fmt.Sprintf("edit-plane-ref-%d", i), label,
			true, s.EditPlaneRefSlotArmed(i), false)
		if arm {
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
