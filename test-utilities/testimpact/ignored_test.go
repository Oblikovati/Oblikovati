// SPDX-License-Identifier: GPL-2.0-only

package testimpact

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// TestDropIgnoredKeepsOnlyWhatGitTracks builds a throwaway repository with one tracked package and one
// in an ignored directory, and asks git — not a pattern matcher — which to keep (#3557).
func TestDropIgnoredKeepsOnlyWhatGitTracks(t *testing.T) {
	root := newIgnoreFixtureRepo(t)
	paths := []string{"example.org", "example.org/lib", "example.org/scratch/probe"}
	kept, err := DropIgnored(root, "example.org", paths)
	if err != nil {
		t.Fatalf("DropIgnored: %v", err)
	}
	if want := []string{"example.org", "example.org/lib"}; !slices.Equal(kept, want) {
		t.Errorf("kept %v, want %v: scratch/ is ignored, the root and lib/ are not", kept, want)
	}
}

// TestDropIgnoredKeepsEverythingWhenNothingIsIgnored: git's exit 1 ("nothing matched") is an answer,
// not a failure, so every path survives.
func TestDropIgnoredKeepsEverythingWhenNothingIsIgnored(t *testing.T) {
	root := newIgnoreFixtureRepo(t)
	paths := []string{"example.org/lib"}
	kept, err := DropIgnored(root, "example.org", paths)
	if err != nil || !slices.Equal(kept, paths) {
		t.Errorf("DropIgnored = %v, %v; want %v, nil", kept, err, paths)
	}
}

// TestDropIgnoredReportsAGitFailure: outside a repository git exits 128. The old code answered that by
// keeping every package — a filter that silently stops filtering — and this pins that it is an error.
func TestDropIgnoredReportsAGitFailure(t *testing.T) {
	if _, err := DropIgnored(t.TempDir(), "example.org", []string{"example.org/lib"}); err == nil {
		t.Error("DropIgnored outside a git repository returned no error")
	}
}

// newIgnoreFixtureRepo is a git repository whose .gitignore ignores scratch/.
func newIgnoreFixtureRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	cmd := exec.Command(gitBinary(), "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, d := range []string{"lib", "scratch/probe"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("scratch/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}
