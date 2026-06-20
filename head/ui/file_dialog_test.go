// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// typePath simulates the user typing into the modal's in-place buffer — what
// native.InputText writes each frame — so these tests exercise the state machine
// without the cgo draw path.
func (d *fileDialog) typePath(s string) {
	d.path = [pathBufferLen]byte{}
	copy(d.path[:], s)
}

func (d *fileDialog) typeSearch(s string) {
	d.search = [fileSearchBufferLen]byte{}
	copy(d.search[:], s)
}

func TestFileDialogConfirmReturnsActionForMode(t *testing.T) {
	var d fileDialog
	d.openFor(dialogSaveAs)
	if !d.isOpen() || d.title() != "Save As" {
		t.Fatalf("openFor(SaveAs): isOpen=%v title=%q", d.isOpen(), d.title())
	}
	path := filepath.Join(t.TempDir(), "part.opd")
	d.typePath(path)
	act := d.confirm()
	if act.Kind != dialogSaveAs || act.Path != path {
		t.Errorf("confirm() = %+v, want SaveAs %q", act, path)
	}
	if d.isOpen() {
		t.Error("dialog should be closed after confirm")
	}
}

// TestFileDialogPlaceComponentMode checks the Place Component picker (#763): its title, that it
// filters to the same .obk document family as Open (a component is a part or sub-assembly), and
// that confirming yields a dialogPlaceComponent action — the one the chrome routes to
// SetPlaceComponentDocument.
func TestFileDialogPlaceComponentMode(t *testing.T) {
	var d fileDialog
	d.openFor(dialogPlaceComponent)
	if !d.isOpen() || d.title() != "Place Component" {
		t.Fatalf("openFor(PlaceComponent): isOpen=%v title=%q", d.isOpen(), d.title())
	}
	var open fileDialog
	open.openFor(dialogOpen)
	if !slices.Equal(d.allowedExts(), open.allowedExts()) {
		t.Errorf("place-component exts = %v, want the document family %v", d.allowedExts(), open.allowedExts())
	}
	path := filepath.Join(t.TempDir(), "widget.obk")
	d.typePath(path)
	if act := d.confirm(); act.Kind != dialogPlaceComponent || act.Path != path {
		t.Errorf("confirm() = %+v, want PlaceComponent %q", act, path)
	}
}

func TestFileDialogCancelYieldsNoAction(t *testing.T) {
	var d fileDialog
	d.openFor(dialogOpen)
	d.typePath("/x.opd")
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
	if d.title() != "Import (.stl/.obj/.3mf/.step/.dwg/.dxf · scans .ply/.xyz/.pts)" {
		t.Errorf("import title = %q", d.title())
	}
	d.openFor(dialogExport)
	if d.title() != "Export (.stl/.obj/.3mf/.step/.dxf)" {
		t.Errorf("export title = %q", d.title())
	}
	if d.resolution != 1 { // defaults to medium
		t.Errorf("export default resolution = %d, want 1 (medium)", d.resolution)
	}
	d.resolution = 2 // high
	copy(d.path[:], "out.stl")
	act := d.confirm()
	want := filepath.Join(initialExplorerDir(), "out.stl")
	if act.Kind != dialogExport || act.Path != want || act.Resolution != "high" {
		t.Errorf("export confirm = %+v, want {dialogExport, %q, high}", act, want)
	}
}

// TestFileDialogExportDXFVersion covers a confirmed Export carrying the selected DXF version.
func TestFileDialogExportDXFVersion(t *testing.T) {
	var d fileDialog
	d.openFor(dialogExport)
	d.dxfVersion = 1 // r2018
	copy(d.path[:], "out.dxf")
	act := d.confirm()
	if act.DXFVersion != "r2018" {
		t.Errorf("export confirm DXFVersion = %q, want r2018", act.DXFVersion)
	}
}

func TestFileDialogSearchFiltersAllowedFiles(t *testing.T) {
	dir := t.TempDir()
	writeDialogFile(t, filepath.Join(dir, "alpha.opd"))
	writeDialogFile(t, filepath.Join(dir, "beta.stl"))
	var d fileDialog
	d.openFor(dialogOpen)
	d.openDir(dir)
	d.typeSearch("alpha")
	got := d.visibleEntries()
	if len(got) != 1 || got[0].Name != "alpha.opd" {
		t.Fatalf("visibleEntries = %+v, want only alpha.opd", got)
	}
}

func TestFileDialogChooseDirectoryAndFile(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	file := filepath.Join(nested, "part.opd")
	writeDialogFile(t, file)
	var d fileDialog
	d.openFor(dialogOpen)
	d.openDir(dir)
	d.chooseEntry(findDialogEntry(t, d.visibleEntries(), "nested"))
	d.chooseEntry(findDialogEntry(t, d.visibleEntries(), "part.opd"))
	if act := d.confirm(); act.Path != file || act.Kind != dialogOpen {
		t.Fatalf("confirm after choose = %+v, want open %q", act, file)
	}
}

func TestFileDialogSaveJoinsDirectoryAndDefaultExt(t *testing.T) {
	dir := t.TempDir()
	var d fileDialog
	d.openFor(dialogSaveAs)
	d.openDir(dir)
	d.typePath("bracket")
	act := d.confirm()
	want := filepath.Join(dir, "bracket.opd")
	if act.Kind != dialogSaveAs || act.Path != want {
		t.Fatalf("confirm save = %+v, want save %q", act, want)
	}
}

func TestFileDialogRefreshReportsDirectoryErrors(t *testing.T) {
	var d fileDialog
	d.openFor(dialogOpen)
	d.openDir(filepath.Join(t.TempDir(), "missing"))
	if d.errorText == "" || len(d.visibleEntries()) != 0 {
		t.Fatalf("missing directory error=%q entries=%d", d.errorText, len(d.visibleEntries()))
	}
}

func writeDialogFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func findDialogEntry(t *testing.T, entries []fileEntry, name string) fileEntry {
	t.Helper()
	for _, entry := range entries {
		if entry.Name == name {
			return entry
		}
	}
	t.Fatalf("entry %q not found in %+v", name, entries)
	return fileEntry{}
}
