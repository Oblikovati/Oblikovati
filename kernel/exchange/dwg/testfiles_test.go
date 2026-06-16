// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"os"
	"path/filepath"
	"testing"
)

// testFilesEnv overrides the default search path for the large real .dwg files
// used by integration tests. They live in the git-ignored experiments tree (each
// is 8–26 MB), so they are never committed; tests skip cleanly when absent so the
// unit suite stays self-contained.
const testFilesEnv = "DWG_TESTFILES_DIR"

func testFilesDir() string {
	if d := os.Getenv(testFilesEnv); d != "" {
		return d
	}
	// Package dir is .../Oblikovati/kernel/exchange/dwg; the corpus sits four
	// levels up under the workspace's experiments tree.
	return filepath.Join("..", "..", "..", "..", "experiments", "dwg-reverse-engineering")
}

// loadTestFile returns the bytes of a corpus file, skipping the test if the
// corpus is not present in this checkout.
func loadTestFile(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join(testFilesDir(), name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("corpus file %s unavailable (%v); set %s to run", name, err, testFilesEnv)
	}
	return data
}
