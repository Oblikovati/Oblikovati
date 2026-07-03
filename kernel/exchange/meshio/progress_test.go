// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"errors"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
)

// meshProgressSink records the progress ticks the mesh importer reports (a named fake per the house
// rules), so a test can assert the shared seam fired.
type meshProgressSink struct{ ticks int }

func (m *meshProgressSink) fn(string, int, int) bool { m.ticks++; return false }

// TestImportBodyReportsProgress checks ImportBody threads the shared progress seam (#1647): it fires
// a triangle-count tick for a decoded mesh.
func TestImportBodyReportsProgress(t *testing.T) {
	var sink meshProgressSink
	_, _, err := ImportBody(types.FormatSTL, []byte(asciiTetraSTL), "import:stl#0", DefaultWeldTolerance,
		exchange.TranslationOptions{Progress: sink.fn})
	if err != nil {
		t.Fatalf("ImportBody: %v", err)
	}
	if sink.ticks == 0 {
		t.Fatal("the mesh importer reported no progress; the seam is not wired")
	}
}

// TestImportBodyCancels checks a cancelling ProgressFunc aborts the mesh import (before the weld)
// with an ErrCancelled-wrapping error.
func TestImportBodyCancels(t *testing.T) {
	cancel := func(string, int, int) bool { return true }
	_, _, err := ImportBody(types.FormatSTL, []byte(asciiTetraSTL), "import:stl#0", DefaultWeldTolerance,
		exchange.TranslationOptions{Progress: cancel})
	if !errors.Is(err, exchange.ErrCancelled) {
		t.Fatalf("cancelled mesh import error = %v, want it to wrap exchange.ErrCancelled", err)
	}
}
