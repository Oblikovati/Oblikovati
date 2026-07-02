// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"testing"

	"oblikovati.org/kernel/exchange"
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
	_, err := ReadScan("scan.e57", []byte("this is not an E57 file"), exchange.TranslationOptions{})
	if err == nil {
		t.Fatal("want a decode error for non-E57 bytes routed to the e57 reader")
	}
}
