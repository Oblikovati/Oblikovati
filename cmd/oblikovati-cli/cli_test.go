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
	path := filepath.Join(t.TempDir(), "bracket.obk")

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

func TestNewAppendsPackageExtension(t *testing.T) {
	stem := filepath.Join(t.TempDir(), "noext")
	if _, err := runCLI(t, "new", "assembly", stem); err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := os.Stat(stem + ".obk"); err != nil {
		t.Fatalf("expected %s.obk on disk: %v", stem, err)
	}
}

func TestInfoJSONIsParseable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plate.obk")
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
	if report.Type != "part" || report.Name != "plate" || report.SchemaVersion != 3 {
		t.Errorf("report = %+v, want {3 part plate}", report)
	}
}

func TestSaveAsCopiesPackage(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.obk")
	dst := filepath.Join(dir, "dst.obk")
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
	path := filepath.Join(t.TempDir(), "p.obk")
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
		{"unknown type", []string{"new", "bogus", "/tmp/x.obk"}},
		{"new missing path", []string{"new", "part"}},
		{"info missing path", []string{"info"}},
		{"save-as one arg", []string{"save-as", "/tmp/a.obk"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := runCLI(t, tc.args...); err == nil {
				t.Errorf("run(%v) = nil error, want a usage error", tc.args)
			}
		})
	}
}
