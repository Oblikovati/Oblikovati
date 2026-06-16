// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"slices"
	"strings"
	"testing"
)

// TestImportDialogAcceptsDWG checks the Import dialog lists and accepts .dwg (so the
// browser shows DWG files and the user can pick one), while Export does not (no DWG
// writer yet).
func TestImportDialogAcceptsDWG(t *testing.T) {
	var d fileDialog
	d.openFor(dialogImport)
	if !slices.Contains(d.allowedExts(), ".dwg") {
		t.Errorf("Import allowedExts %v missing .dwg", d.allowedExts())
	}
	if !d.allowsFile("floor.dwg") {
		t.Error("Import dialog rejected floor.dwg")
	}
	if !strings.Contains(d.title(), ".dwg") {
		t.Errorf("Import title %q does not mention .dwg", d.title())
	}

	var e fileDialog
	e.openFor(dialogExport)
	if slices.Contains(e.allowedExts(), ".dwg") {
		t.Error("Export should not offer .dwg yet (no writer)")
	}
}

// TestImportConfirmCarriesPlaneIndex checks the chosen work-plane index rides along
// in the action, so the DWG import lands on the plane the user picked.
func TestImportConfirmCarriesPlaneIndex(t *testing.T) {
	var d fileDialog
	d.openFor(dialogImport)
	d.typePath("/tmp/floor.dwg")
	d.planeIndex = 2
	act := d.confirm()
	if act.Kind != dialogImport || act.PlaneIndex != 2 {
		t.Fatalf("confirm() = {Kind:%v PlaneIndex:%d}, want {dialogImport, 2}", act.Kind, act.PlaneIndex)
	}
}
