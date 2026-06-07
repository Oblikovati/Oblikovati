// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeScript writes src to a temp .lua file and returns its path.
func writeScript(t *testing.T, src string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "script.lua")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

func TestScriptRunBuildsAndSavesDocument(t *testing.T) {
	out := filepath.Join(t.TempDir(), "built.obk")
	lua := writeScript(t, `oblikovati.call("documents.create", { type = "part", name = "built" })
	                       oblikovati.call("parameters.add", { name = "width", expression = "4 cm" })
	                       print("done")`)
	stdout, err := runCLI(t, "script", "run", lua, "--save", out)
	if err != nil {
		t.Fatalf("script run: %v", err)
	}
	if !strings.Contains(stdout, "done") {
		t.Errorf("script print() missing from output: %q", stdout)
	}
	if _, statErr := os.Stat(out); statErr != nil {
		t.Fatalf("expected saved document %s: %v", out, statErr)
	}
	// The saved document round-trips: info reports the part.
	info, err := runCLI(t, "info", out)
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	if !strings.Contains(info, "type: part") {
		t.Errorf("saved doc info = %q, want a part", info)
	}
}

func TestScriptRunFailingScriptExitsNonZero(t *testing.T) {
	lua := writeScript(t, `error("intentional failure value")`)
	out, err := runCLI(t, "script", "run", lua)
	if err == nil {
		t.Fatal("a failing script must return a non-nil error (non-zero exit)")
	}
	if !strings.Contains(err.Error(), "intentional failure value") {
		t.Errorf("error should carry the offending value, got %q", err.Error())
	}
	_ = out
}

func TestScriptRunSandboxEscapeFails(t *testing.T) {
	lua := writeScript(t, `local f = io.open("/etc/passwd", "r")`)
	if _, err := runCLI(t, "script", "run", lua); err == nil {
		t.Fatal("io must be denied: the script should fail, not read the filesystem")
	}
}

func TestScriptRunMissingFileErrors(t *testing.T) {
	if _, err := runCLI(t, "script", "run", "/no/such/file.lua"); err == nil {
		t.Fatal("a missing script file must error")
	}
}

func TestScriptRunBadUsageErrors(t *testing.T) {
	if _, err := runCLI(t, "script"); err == nil {
		t.Fatal("`script` with no subcommand must error")
	}
	if _, err := runCLI(t, "script", "frobnicate"); err == nil {
		t.Fatal("an unknown script subcommand must error")
	}
}
