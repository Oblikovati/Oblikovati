// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"errors"
	"testing"

	"oblikovati.org/kernel/exchange"
)

// TestDecodeWithProgressCancels checks the DWG decoder honours the shared cancel seam (#1647): a
// ProgressFunc that cancels on the first object tick aborts the import promptly (before decoding the
// whole object map) with an ErrCancelled-wrapping error. Skips when the DWG corpus is absent.
func TestDecodeWithProgressCancels(t *testing.T) {
	data := loadTestFile(t, "testfile-2.dwg")
	cancel := func(string, int, int) bool { return true }
	_, _, err := DecodeWithProgress(data, exchange.TranslationOptions{Progress: cancel})
	if !errors.Is(err, exchange.ErrCancelled) {
		t.Fatalf("cancelled DWG import error = %v, want it to wrap exchange.ErrCancelled", err)
	}
}
