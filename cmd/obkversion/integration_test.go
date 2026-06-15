// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestComputeAgainstRealRepo exercises the real git/file glue (compute, fixedFields,
// realGit, runGit) end to end in a throwaway repo — the part assemble's fake cannot
// reach. It mirrors the documented behavior: a feature commit on a fresh api line gives
// .1.0, then a fix on top of that tag gives .1.1.
func TestComputeAgainstRealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	writeFixture(t, dir, "version.yaml", "major: 0\n")
	writeFixture(t, dir, "go.mod", "module oblikovati.org\n\ngo 1.22\n\nrequire oblikovati.org/api v0.2.0\n")
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.email", "t@example.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "feat: initial cut")

	if got, err := compute("stable", dir); err != nil || got != "0.000200.1.0" {
		t.Fatalf("compute(stable) = %q, %v; want 0.000200.1.0", got, err)
	}

	// Tag that release, add a fix, and the next stable is a PATCH bump off the tag.
	git(t, dir, "tag", "v0.000200.1.0")
	writeFixture(t, dir, "f.txt", "x")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "fix: a bug")

	if got, err := compute("stable", dir); err != nil || got != "0.000200.1.1" {
		t.Fatalf("compute(stable) after fix = %q, %v; want 0.000200.1.1", got, err)
	}

	// Nightly carries the same core plus the prerelease stamp.
	got, err := compute("nightly", dir)
	if err != nil || got[:len("0.000200.1.1-nightly.")] != "0.000200.1.1-nightly." {
		t.Fatalf("compute(nightly) = %q, %v; want 0.000200.1.1-nightly.<ts>", got, err)
	}
}

func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
