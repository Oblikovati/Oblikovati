// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunImportsBoxToOPD(t *testing.T) {
	out := filepath.Join(t.TempDir(), "box.opd")
	err := run([]string{filepath.Join("..", "..", "testdata", "10_box.ipt"), "-o", out}, os.Stdout)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("expected %s to exist: %v", out, err)
	}
}

func TestRunRejectsMissingSource(t *testing.T) {
	if err := run([]string{"-o", "x.opd"}, os.Stdout); err == nil {
		t.Errorf("expected an error when no .ipt argument is given")
	}
}
