// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunTranslatesToOPD(t *testing.T) {
	out := filepath.Join(t.TempDir(), "box.opd")
	err := run([]string{filepath.Join("..", "..", "testdata", "box10_fmtb.sldprt"), "-o", out}, os.Stdout)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("expected %s to exist: %v", out, err)
	}
}

func TestRunRejectsMissingSource(t *testing.T) {
	if err := run([]string{"-o", "x.opd"}, os.Stdout); err == nil {
		t.Error("expected an error when no .SLDPRT argument is given")
	}
}

// TestRunDefaultsOutputBesideSource covers the branch that derives the .opd path from the source
// when -o is omitted. The fixture is copied into a temp dir so the output lands there, not in the repo.
func TestRunDefaultsOutputBesideSource(t *testing.T) {
	src := filepath.Join("..", "..", "testdata", "box10_fmtb.sldprt")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	dir := t.TempDir()
	tmp := filepath.Join(dir, "box10_fmtb.sldprt")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		t.Fatalf("write temp fixture: %v", err)
	}
	if err := run([]string{tmp}, os.Stdout); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "box10_fmtb.opd")); err != nil {
		t.Errorf("expected default .opd beside the source: %v", err)
	}
}
