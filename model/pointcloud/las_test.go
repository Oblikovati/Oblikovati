// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import "testing"

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
	_, err := ReadScan("survey.las", []byte("this is not a LAS file"))
	if err == nil {
		t.Fatal("want a decode error for non-LAS bytes routed to the las reader")
	}
}
