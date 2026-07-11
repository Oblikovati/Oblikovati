// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"
)

func TestImportInputBoxArea(t *testing.T) {
	b, err := importInput(filepath.Join("testdata", "A1.step"))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	got := inputArea(b)
	if got < 59000 || got > 61000 {
		t.Fatalf("box surface area = %g, want ~60000", got)
	}
}
