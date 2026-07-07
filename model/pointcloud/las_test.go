// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"testing"

	"oblikovati.org/kernel/exchange"
)

// The LAS decode itself is covered in kernel/exchange/lasfmt; here we cover the model-layer
// reader's wiring: its extension, that ReadScan dispatches .las to it, and that malformed bytes
// surface an error rather than panicking (#645).

func TestLASReaderExtensions(t *testing.T) {
	exts := NewLASReader().Extensions()
	if len(exts) != 1 || exts[0] != ".las" {
		t.Fatalf("LAS extensions = %v, want [.las]", exts)
	}
}

func TestReadScanDispatchesLASError(t *testing.T) {
	_, _, err := ReadScan("survey.las", []byte("this is not a LAS file"), exchange.TranslationOptions{})
	if err == nil {
		t.Fatal("want a decode error for non-LAS bytes routed to the las reader")
	}
}

// TestLASMillimetreScan covers the #1789 signal: a LAS quantised to a whole metre or coarser on
// every axis is read as millimetres, while any sub-metre axis (a real survey) keeps the metre unit.
// A zero/unset scale (malformed header) is not flagged, so the metre default stands.
func TestLASMillimetreScan(t *testing.T) {
	cases := []struct {
		name  string
		scale [3]float64
		want  bool
	}{
		{"unit metre steps", [3]float64{1, 1, 1}, true},
		{"coarser than metre", [3]float64{2, 3, 1.5}, true},
		{"sub-metre survey", [3]float64{0.001, 0.001, 0.01}, false},
		{"one sub-metre axis", [3]float64{1, 0.5, 1}, false},
		{"unset scale", [3]float64{0, 0, 0}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := lasMillimetreScan(c.scale); got != c.want {
				t.Errorf("lasMillimetreScan(%v) = %v, want %v", c.scale, got, c.want)
			}
		})
	}
}

// TestLASReaderImplementsPerFileUnit: the LAS reader must expose the per-file unit seam, and an
// undecodable header must decline (ok=false) so the static FileUnitMM stays in force (#1789).
func TestLASReaderImplementsPerFileUnit(t *testing.T) {
	r, ok := NewLASReader().(perFileUnitReader)
	if !ok {
		t.Fatal("LAS reader must implement perFileUnitReader for the #1789 mm override")
	}
	if _, ok := r.fileUnitMM([]byte("not a LAS file")); ok {
		t.Error("fileUnitMM should report ok=false for undecodable bytes, leaving FileUnitMM in force")
	}
}
