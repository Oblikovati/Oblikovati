// SPDX-License-Identifier: GPL-2.0-only

package pdf

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange/drawing"
)

// corpusFile loads a large real PDF from PDF_TESTFILES_DIR (the workspace root holds the
// pdf-vector*.pdf samples), skipping the test when the corpus is unavailable — the
// multi-megabyte files are not committed, mirroring the DWG corpus convention.
func corpusFile(t *testing.T, name string) []byte {
	t.Helper()
	dir := os.Getenv("PDF_TESTFILES_DIR")
	if dir == "" {
		dir = filepath.Join("..", "..", "..", "..")
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Skipf("corpus unavailable: %v", err)
	}
	return data
}

// TestDecodeRealAutoCADPlot decodes a real AutoCAD plot-to-PDF and checks it yields a
// drawing-sized body of curve geometry on a single page, in millimetres.
func TestDecodeRealAutoCADPlot(t *testing.T) {
	pages, _, err := DecodePages(corpusFile(t, "pdf-vector2.pdf"))
	if err != nil {
		t.Fatalf("DecodePages: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(pages))
	}
	dr := pages[0]
	if dr.Units != drawing.INSMillimetres {
		t.Errorf("Units = %d, want mm", dr.Units)
	}
	if len(dr.Entities) < 100 {
		t.Errorf("decoded only %d entities; expected a dense drawing", len(dr.Entities))
	}
}
