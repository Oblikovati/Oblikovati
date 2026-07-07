// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/math"
)

// The E57 decode itself is covered in kernel/exchange/e57fmt; here we cover the model-layer
// reader's wiring: its extension, that ReadScan dispatches .e57 to it, and that malformed bytes
// surface an error rather than panicking (#645).

func TestE57ReaderExtensions(t *testing.T) {
	exts := NewE57Reader().Extensions()
	if len(exts) != 1 || exts[0] != ".e57" {
		t.Fatalf("E57 extensions = %v, want [.e57]", exts)
	}
}

func TestReadScanDispatchesE57Error(t *testing.T) {
	// A registered .e57 reader must be reached (not the "no reader" error) and reject non-E57 bytes.
	_, _, err := ReadScan("scan.e57", []byte("this is not an E57 file"), exchange.TranslationOptions{})
	if err == nil {
		t.Fatal("want a decode error for non-E57 bytes routed to the e57 reader")
	}
}

// TestE57UnitMM maps the cartesian-resolution fact to the file's length unit: an integer-resolution
// scan is treated as millimetres (#1789), a conformant scan keeps the ASTM E2807 metre.
func TestE57UnitMM(t *testing.T) {
	if got := e57UnitMM(true); got != 1 {
		t.Errorf("e57UnitMM(integerResolution) = %v mm, want 1 (millimetres)", got)
	}
	if got := e57UnitMM(false); got != 1000 {
		t.Errorf("e57UnitMM(conformant) = %v mm, want 1000 (metres)", got)
	}
}

// TestE57ReaderImplementsPerFileUnit: the E57 reader must expose the per-file unit seam, and an
// undecodable header must decline (ok=false) so the static FileUnitMM stays in force (#1789).
func TestE57ReaderImplementsPerFileUnit(t *testing.T) {
	r, ok := NewE57Reader().(perFileUnitReader)
	if !ok {
		t.Fatal("E57 reader must implement perFileUnitReader for the #1789 mm override")
	}
	if _, ok := r.fileUnitMM([]byte("not an E57 file")); ok {
		t.Error("fileUnitMM should report ok=false for undecodable bytes, leaving FileUnitMM in force")
	}
}

// fakeUnitReader is a named fake PointReader (per the test guidelines) whose per-file unit differs
// from its static unit, so readScaled's preference for the per-file override is observable (#1789).
type fakeUnitReader struct {
	staticMM  float64
	perFileMM float64
	perFileOK bool
}

func (fakeUnitReader) Extensions() []string                         { return []string{".fake"} }
func (f fakeUnitReader) FileUnitMM() float64                        { return f.staticMM }
func (fakeUnitReader) Read([]byte) ([]math.Point3, []string, error) { return nil, nil, nil }
func (fakeUnitReader) ReadSamples([]byte) ([]PointSample, []string, error) {
	return []PointSample{{Point: math.P3(5, 0, 0)}}, nil, nil
}
func (f fakeUnitReader) fileUnitMM([]byte) (float64, bool) { return f.perFileMM, f.perFileOK }

// TestReadScaledPrefersPerFileUnit: with a metre static unit but a millimetre per-file override, a
// point at x=5 scales as millimetres (×1 into the mm database unit → 5), not metres (which would
// give 5000) — the exact 1000× the override exists to prevent (#1789).
func TestReadScaledPrefersPerFileUnit(t *testing.T) {
	r := fakeUnitReader{staticMM: 1000, perFileMM: 1, perFileOK: true}
	samples, _, err := readScaled(r, nil, exchange.TranslationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := samples[0].Point.X; got != 5 {
		t.Errorf("per-file mm override: x = %v, want 5 (metre path would give 5000)", got)
	}
}

// TestReadScaledFallsBackToStaticUnit: when the per-file override declines (ok=false), the static
// metre unit applies, so x=5 scales to 5000.
func TestReadScaledFallsBackToStaticUnit(t *testing.T) {
	r := fakeUnitReader{staticMM: 1000, perFileMM: 1, perFileOK: false}
	samples, _, err := readScaled(r, nil, exchange.TranslationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := samples[0].Point.X; got != 5000 {
		t.Errorf("static metre fallback: x = %v, want 5000", got)
	}
}
