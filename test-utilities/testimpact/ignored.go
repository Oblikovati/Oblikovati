// SPDX-License-Identifier: GPL-2.0-only

package testimpact

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// DropIgnored removes the import paths whose directory git ignores, so a gate measures what git
// tracks. `go list ./...` walks experiments/, which .gitignore excludes and which holds unfinished
// code by design, so a half-written experiment on one developer's disk failed every local run of a
// gate about code being committed while CI — which never receives those files — passed (#3557).
//
// It asks git once, through `git check-ignore --stdin`, rather than matching .gitignore here, so the
// answer is whatever git itself would say. module is the import prefix an import path loses to become
// a directory under root.
//
//	kept, err := testimpact.DropIgnored(root, "oblikovati.org", paths)
func DropIgnored(root, module string, paths []string) ([]string, error) {
	dirs := make([]string, len(paths))
	for i, p := range paths {
		dirs[i] = "." + strings.TrimPrefix(p, module)
	}
	ignored, err := ignoredDirs(root, dirs)
	if err != nil {
		return nil, err
	}
	kept := make([]string, 0, len(paths))
	for i, p := range paths {
		if !ignored[strings.TrimSuffix(dirs[i], "/")] {
			kept = append(kept, p)
		}
	}
	return kept, nil
}

// ignoredDirs is the subset of dirs git ignores under root. `git check-ignore` exits 1 when none is,
// which is the ordinary case on a clean checkout, so that status is an answer; any other failure is
// an error, never "keep everything" — a gate that silently stops filtering is the defect this exists
// to remove, moved one step along.
func ignoredDirs(root string, dirs []string) (map[string]bool, error) {
	// NOSONAR go:S4036 — gitBinary() has already resolved this through exec.LookPath. This is
	// developer tooling invoked from make in the developer's own shell; a PATH that can substitute
	// `git` owns that shell already.
	cmd := exec.Command(gitBinary(), "check-ignore", "--stdin") // NOSONAR
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(strings.Join(dirs, "\n"))
	out, err := cmd.Output()
	if err != nil && !nothingIgnored(err) {
		return nil, fmt.Errorf("git check-ignore in %s: %w", root, err)
	}
	ignored := make(map[string]bool)
	for _, d := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if d != "" {
			ignored[strings.TrimSuffix(d, "/")] = true
		}
	}
	return ignored, nil
}

// nothingIgnored reports whether a `git check-ignore` failure is its exit 1, "no path matched" — an
// answer, unlike 128, which is git failing.
func nothingIgnored(err error) bool {
	var exit *exec.ExitError
	return errors.As(err, &exit) && exit.ExitCode() == 1
}
