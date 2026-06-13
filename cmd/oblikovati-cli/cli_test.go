// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runCLI invokes the dispatcher with args, capturing user-facing output. It is the
// seam the table tests drive — no process, no os.Exit.
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := run(args, &out)
	return out.String(), err
}

func TestNewWritesPackageThatInfoReports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bracket.opd")

	out, err := runCLI(t, "new", "part", path)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if !strings.Contains(out, "created part") {
		t.Errorf("new output = %q, want it to mention 'created part'", out)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("expected %s on disk: %v", path, statErr)
	}

	info, err := runCLI(t, "info", path)
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	if !strings.Contains(info, "type: part") || !strings.Contains(info, "name: bracket") {
		t.Errorf("info output = %q, want type: part and name: bracket", info)
	}
}

// TestNewCreatesEachKindWithItsExtension is the basic-file-handling guard for the
// four document formats (ADR-0034, M11-F00): creating each kind stamps its own
// extension and writes a package that `info` can read back as that kind.
func TestNewCreatesEachKindWithItsExtension(t *testing.T) {
	for _, c := range []struct{ kind, ext string }{
		{"part", ".opd"}, {"assembly", ".oad"}, {"drawing", ".odd"}, {"presentation", ".ord"},
	} {
		t.Run(c.kind, func(t *testing.T) {
			stem := filepath.Join(t.TempDir(), "doc")
			if _, err := runCLI(t, "new", c.kind, stem); err != nil {
				t.Fatalf("new %s: %v", c.kind, err)
			}
			if _, err := os.Stat(stem + c.ext); err != nil {
				t.Fatalf("expected %s%s on disk: %v", stem, c.ext, err)
			}
			out, err := runCLI(t, "info", stem+c.ext)
			if err != nil {
				t.Fatalf("info %s: %v", c.kind, err)
			}
			if !strings.Contains(out, "type: "+c.kind) {
				t.Errorf("info = %q, want it to report type: %s", out, c.kind)
			}
		})
	}
}

func TestNewAppendsPackageExtension(t *testing.T) {
	stem := filepath.Join(t.TempDir(), "noext")
	if _, err := runCLI(t, "new", "assembly", stem); err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := os.Stat(stem + ".oad"); err != nil {
		t.Fatalf("expected %s.oad on disk: %v", stem, err)
	}
}

func TestInfoJSONIsParseable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plate.opd")
	if _, err := runCLI(t, "new", "part", path); err != nil {
		t.Fatalf("new: %v", err)
	}
	// --json must work in either position relative to the path.
	out, err := runCLI(t, "info", path, "--json")
	if err != nil {
		t.Fatalf("info --json: %v", err)
	}
	var report manifestReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("info --json emitted invalid JSON %q: %v", out, err)
	}
	if report.Type != "part" || report.Name != "plate" || report.SchemaVersion != 2 {
		t.Errorf("report = %+v, want {2 part plate}", report)
	}
}

func TestSaveAsCopiesPackage(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.odd")
	dst := filepath.Join(dir, "dst.odd")
	if _, err := runCLI(t, "new", "drawing", src); err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := runCLI(t, "save-as", src, dst); err != nil {
		t.Fatalf("save-as: %v", err)
	}
	out, err := runCLI(t, "info", dst)
	if err != nil {
		t.Fatalf("info dst: %v", err)
	}
	if !strings.Contains(out, "type: drawing") || !strings.Contains(out, "name: dst") {
		t.Errorf("copied package info = %q, want type: drawing and name: dst", out)
	}
}

func TestOpenReportsIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.opd")
	if _, err := runCLI(t, "new", "part", path); err != nil {
		t.Fatalf("new: %v", err)
	}
	out, err := runCLI(t, "open", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if !strings.Contains(out, "opened part") {
		t.Errorf("open output = %q, want 'opened part'", out)
	}
}

func TestUsageErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"no command", nil},
		{"unknown command", []string{"frobnicate"}},
		{"unknown type", []string{"new", "bogus", "/tmp/x.opd"}},
		{"new missing path", []string{"new", "part"}},
		{"info missing path", []string{"info"}},
		{"save-as one arg", []string{"save-as", "/tmp/a.opd"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := runCLI(t, tc.args...); err == nil {
				t.Errorf("run(%v) = nil error, want a usage error", tc.args)
			}
		})
	}
}
