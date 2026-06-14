// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/model/bom"
)

// TestAssemblyBOMCountsComponents: two instances of one component group into a single BOM row with
// quantity 2 (the BOM counts identical components), proving the panel reads the live assembly.
func TestAssemblyBOMCountsComponents(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	placedWidget(t, s, asm, "widget:2")

	view, err := s.AssemblyBOM()
	if err != nil {
		t.Fatalf("AssemblyBOM: %v", err)
	}
	if len(view.Rows) != 1 {
		t.Fatalf("BOM rows = %d, want 1 (two instances of one component group)", len(view.Rows))
	}
	if got := view.Rows[0].Quantity; got != 2 {
		t.Errorf("row quantity = %d, want 2", got)
	}
}

// TestBOMViewKindSelectsView: the panel's view toggle drives which view AssemblyBOM builds.
func TestBOMViewKindSelectsView(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")

	s.SetBOMViewKind(bom.PartsOnlyView)
	view, err := s.AssemblyBOM()
	if err != nil {
		t.Fatalf("AssemblyBOM: %v", err)
	}
	if view.Kind != bom.PartsOnlyView {
		t.Errorf("view kind = %v, want PartsOnlyView", view.Kind)
	}
}

// TestExportBOMCSVWritesFile: Export writes a CSV with the standard header and a row per component,
// at the chosen path.
func TestExportBOMCSVWritesFile(t *testing.T) {
	s, asm := assemblyWithComponent(t)
	placedWidget(t, s, asm, "widget:1")
	placedWidget(t, s, asm, "widget:2")

	path := filepath.Join(t.TempDir(), "bom.csv")
	if err := s.ExportBOMCSV(path); err != nil {
		t.Fatalf("ExportBOMCSV: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	csv := string(data)
	if !strings.HasPrefix(csv, "Item,Part Number,Description,QTY,Structure") {
		t.Errorf("CSV header = %q, want the standard columns", strings.SplitN(csv, "\n", 2)[0])
	}
	// The single grouped row carries quantity 2 — the last data cell before Structure.
	if !strings.Contains(csv, ",2,") {
		t.Errorf("CSV does not record the quantity-2 row:\n%s", csv)
	}
}

// TestAssemblyBOMRequiresAssembly: building/exporting a BOM on a part errors rather than returning
// an empty view.
func TestAssemblyBOMRequiresAssembly(t *testing.T) {
	s, _ := emptyPartSession(t)
	if _, err := s.AssemblyBOM(); err == nil {
		t.Error("AssemblyBOM on a part should error")
	}
	if err := s.ExportBOMCSV(filepath.Join(t.TempDir(), "x.csv")); err == nil {
		t.Error("ExportBOMCSV on a part should error")
	}
}

// TestBOMCommandOpensPanel: the Assemble ▸ Bill of Materials command opens the panel and is gated
// to an active assembly.
func TestBOMCommandOpensPanel(t *testing.T) {
	s := assemblySession(t)
	if s.BOMPanelOpen() {
		t.Fatal("the BOM panel should start closed")
	}
	if err := s.Execute("Assembly.BOM"); err != nil {
		t.Fatalf("execute Assembly.BOM: %v", err)
	}
	if !s.BOMPanelOpen() {
		t.Error("executing Assembly.BOM should open the panel")
	}
	cmd, _ := s.Commands().ByID("Assembly.BOM")
	if cmd.IsEnabled(registeredSession(t)) {
		t.Error("Bill of Materials should be disabled on a part document")
	}
}
