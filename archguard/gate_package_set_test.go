// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"errors"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// This file is the #3557 guard, and it is the other half of the claim gate_module_coverage_test.go
// makes. That file proves the gate cannot cover LESS than the repo holds; this one proves it cannot
// be failed by MORE — by code the repo does not hold at all.
//
// `./...` walks the working tree, and the working tree is not the checkout: `experiments/` is
// git-ignored scratch. A half-finished experiment on a developer's disk therefore failed `make gate`
// — 35 minutes, on a test whose fixture path was a /tmp directory from another session — about code
// that was being committed, while CI, which never sees those files, went green. A gate that fails on
// what it does not gate teaches people to skip it.
//
// Probes used to prove BOTH clauses bite (arm it, watch it fail, restore):
//   - `TEST_PKGS = $(shell go list ./...)` in the Makefile — the unfiltered expansion, which is what
//     `./...` meant before #3557 — fails the ignored-package clause naming every `experiments/`
//     package on disk.
//   - `TEST_PKGS = $(PKG)` — one literal `./...` token rather than a list — fails the coverage
//     clause instead, naming every tracked package.
//
// The first probe also caught this guard's own first draft deriving the package directory by
// trimming a module prefix taken from `go list -m`, which in a go.work workspace prints EVERY
// workspace module. The prefix never matched, and the paths it built only looked ignored because
// `experiments/` matches at any depth. The directories now come from `go list` itself.

// TestGatePackageSetIsWhatGitTracks asserts the set `make gate` runs holds no package git ignores,
// and that it drops nothing else: every package `go list ./...` finds is either in the set or
// ignored. The two halves matter separately — the first is the defect, the second is the guard
// against "fixing" it by narrowing the gate.
func TestGatePackageSetIsWhatGitTracks(t *testing.T) {
	gated := strings.Fields(runInRepoRoot(t, "make", "--no-print-directory", "print-test-packages"))
	if len(gated) == 0 {
		t.Fatal("make print-test-packages printed nothing: `make gate` would test no package at all " +
			"and still report green (#3557)")
	}
	dirOf := packageDirs(t)
	if len(dirOf) == 0 {
		t.Fatal("go list reported no package; the comparison below would be vacuous")
	}

	for _, pkg := range gated {
		dir, ok := dirOf[pkg]
		if !ok {
			t.Errorf("`make gate` would test %s, which `go list ./...` does not report at all", pkg)
			continue
		}
		if gitIgnores(t, dir) {
			t.Errorf("`make gate` would test %s (%s), which git ignores: a gate about committed code "+
				"must not be failed by code that is not committed (#3557)", pkg, dir)
		}
	}
	for pkg, dir := range dirOf {
		if slices.Contains(gated, pkg) || gitIgnores(t, dir) {
			continue
		}
		t.Errorf("go list ./... finds %s (%s), git tracks it, and `make gate` does not test it — the "+
			"gate covers less than the repo holds (#3526)", pkg, dir)
	}
}

// packageDirs maps every root-module import path to its directory relative to the repo root, in one
// `go list` call. Asking go for the directory is what keeps this guard from doing module-path
// arithmetic of its own: `go list -m` prints every module in the go.work workspace, so a prefix
// trimmed from it silently matches nothing.
func packageDirs(t *testing.T) map[string]string {
	t.Helper()
	root := strings.TrimSpace(runInRepoRoot(t, "git", "rev-parse", "--show-toplevel"))
	out := runInRepoRoot(t, "go", "list", "-f", "{{.ImportPath}} {{.Dir}}", "./...")
	dirs := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		pkg, dir, found := strings.Cut(strings.TrimSpace(line), " ")
		if !found {
			continue
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			t.Fatalf("package %s sits at %q, which is not under the repo root %q: %v", pkg, dir, root, err)
		}
		dirs[pkg] = rel
	}
	return dirs
}

// gitIgnores asks git, rather than matching .gitignore patterns here: a second pattern matcher is a
// second answer, and the whole point is that one authority decides what the checkout contains.
func gitIgnores(t *testing.T, dir string) bool {
	t.Helper()
	cmd := exec.Command("git", "check-ignore", "-q", dir)
	cmd.Dir = ".."
	err := cmd.Run()
	if err == nil {
		return true
	}
	// git check-ignore exits 1 for "not ignored" and 128 for a real failure; only 1 is an answer.
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false
	}
	t.Fatalf("git check-ignore %q failed: %v", dir, err)
	return false
}
