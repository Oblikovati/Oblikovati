// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"bytes"
	"path/filepath"
	"testing"
)

func fixture() string { return filepath.Join("..", "..", "testdata", "box10_fmtb.sldprt") }

func TestRunListsStreams(t *testing.T) {
	var buf bytes.Buffer
	if err := run([]string{fixture()}, &buf); err != nil {
		t.Fatalf("run: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected a stream listing, got no output")
	}
}

func TestRunDumpsSketchesAndFeatures(t *testing.T) {
	for _, mode := range []string{"-sketches", "-features"} {
		var buf bytes.Buffer
		if err := run([]string{fixture(), mode}, &buf); err != nil {
			t.Errorf("run %s: %v", mode, err)
		}
	}
}

func TestRunRejectsNoArgs(t *testing.T) {
	if err := run(nil, &bytes.Buffer{}); err == nil {
		t.Error("expected an error when no file argument is given")
	}
}

func TestRunReportsUnreadableFile(t *testing.T) {
	if err := run([]string{filepath.Join(t.TempDir(), "nope.sldprt")}, &bytes.Buffer{}); err == nil {
		t.Error("expected an error for a missing file")
	}
}
