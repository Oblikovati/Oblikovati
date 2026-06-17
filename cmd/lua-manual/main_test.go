// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// apiDir resolves the oblikovati.org/api module root the same way the wiki publish script does.
func apiDir(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "oblikovati.org/api").Output()
	if err != nil {
		t.Fatalf("go list api: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func TestRunWritesToStdout(t *testing.T) {
	var buf bytes.Buffer
	if err := run([]string{apiDir(t)}, &buf); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(buf.String(), "# Lua Scripting") || !strings.Contains(buf.String(), "## API reference") {
		t.Error("stdout output is not the Lua manual")
	}
}

func TestRunWritesToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Lua-Scripting.md")
	if err := run([]string{apiDir(t), path}, nil); err != nil {
		t.Fatalf("run: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(b), "oblikovati.documents.create") {
		t.Error("file output missing the reference")
	}
}

func TestRunRejectsMissingAPIDir(t *testing.T) {
	if err := run(nil, nil); err == nil {
		t.Error("run with no api dir should error")
	}
}

func TestLoadExamplesHaveDescriptions(t *testing.T) {
	exs, err := loadExamples()
	if err != nil {
		t.Fatalf("loadExamples: %v", err)
	}
	if len(exs) == 0 {
		t.Fatal("no examples loaded")
	}
	for _, ex := range exs {
		if ex.Source == "" || ex.Description == "" {
			t.Errorf("%s: missing source or description", ex.Name)
		}
	}
}

func TestLeadingComment(t *testing.T) {
	src := "-- first line\n-- second line\noblikovati.documents.list{}\n"
	if got := leadingComment(src); got != "first line second line" {
		t.Errorf("leadingComment = %q, want %q", got, "first line second line")
	}
	if got := leadingComment("oblikovati.x()\n"); got != "" {
		t.Errorf("leadingComment with no comment = %q, want empty", got)
	}
}
