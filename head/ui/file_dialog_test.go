// SPDX-License-Identifier: GPL-2.0-only

package ui

import "testing"

// typePath simulates the user typing into the modal's in-place buffer — what
// native.InputText writes each frame — so these tests exercise the state machine
// without the cgo draw path.
func (d *fileDialog) typePath(s string) {
	d.path = [pathBufferLen]byte{}
	copy(d.path[:], s)
}

func TestFileDialogConfirmReturnsActionForMode(t *testing.T) {
	var d fileDialog
	d.openFor(dialogSaveAs)
	if !d.isOpen() || d.title() != "Save As" {
		t.Fatalf("openFor(SaveAs): isOpen=%v title=%q", d.isOpen(), d.title())
	}
	d.typePath("/models/part.obk")
	act := d.confirm()
	if act.Kind != dialogSaveAs || act.Path != "/models/part.obk" {
		t.Errorf("confirm() = %+v, want SaveAs /models/part.obk", act)
	}
	if d.isOpen() {
		t.Error("dialog should be closed after confirm")
	}
}

func TestFileDialogCancelYieldsNoAction(t *testing.T) {
	var d fileDialog
	d.openFor(dialogOpen)
	d.typePath("/x.obk")
	d.cancel()
	if d.isOpen() {
		t.Error("cancel should close the dialog")
	}
	if d.text() != "" {
		t.Errorf("cancel should clear the path, got %q", d.text())
	}
}

func TestFileDialogConfirmEmptyPathIsNoAction(t *testing.T) {
	var d fileDialog
	d.openFor(dialogOpen)
	if act := d.confirm(); act.Kind != dialogClosed {
		t.Errorf("confirm with empty path = %+v, want dialogClosed", act)
	}
}

func TestFileDialogOpenForReArmsAndClears(t *testing.T) {
	var d fileDialog
	d.openFor(dialogOpen)
	d.typePath("stale")
	d.openFor(dialogSaveAs)
	if d.title() != "Save As" {
		t.Errorf("title after re-arm = %q, want Save As", d.title())
	}
	if d.text() != "" {
		t.Errorf("re-arm should clear the prior path, got %q", d.text())
	}
}

func TestFileDialogTitleDefaultsToOpen(t *testing.T) {
	var d fileDialog
	d.openFor(dialogOpen)
	if d.title() != "Open" {
		t.Errorf("title() = %q, want Open", d.title())
	}
}

// TestFileDialogImportExportModes covers the Import/Export modes: their titles and that a
// confirmed Export carries the selected mesh resolution.
func TestFileDialogImportExportModes(t *testing.T) {
	var d fileDialog
	d.openFor(dialogImport)
	if d.title() != "Import (.stl/.obj/.3mf/.step)" {
		t.Errorf("import title = %q", d.title())
	}
	d.openFor(dialogExport)
	if d.resolution != 1 { // defaults to medium
		t.Errorf("export default resolution = %d, want 1 (medium)", d.resolution)
	}
	d.resolution = 2 // high
	copy(d.path[:], "out.stl")
	act := d.confirm()
	if act.Kind != dialogExport || act.Path != "out.stl" || act.Resolution != "high" {
		t.Errorf("export confirm = %+v, want {dialogExport, out.stl, high}", act)
	}
}
