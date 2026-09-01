// SPDX-License-Identifier: GPL-2.0-only

// Package testguard reports which tests exclude themselves from the fast tier.
//
// A tier-2 test guards itself with testing.Short(), directly or through a harness it
// calls; a whole corpus package gates itself in TestMain
// (architecture/testing/03-test-tiers-and-selection.md). Knowing which is which lets
// the budget gate run off the tier-2 timings CI already produces, instead of paying
// for a second `-short` run of the same tests.
//
//	pkgs, err := testguard.Scan(".")
//	if !pkgs["kernel/ops"].Guards("TestSlowCorpusCase") { ... }
package testguard

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// skipDirs are trees with no first-party tests to classify.
var skipDirs = map[string]bool{
	".git": true, "third_party": true, "testdata": true, "node_modules": true,
	"dist": true, "_api": true,
}

// Set is the guard state of every scanned package, keyed by slash-separated path
// relative to the scan root ("kernel/ops").
type Set map[string]*Package

// Guards reports whether the named test in the named package directory is excluded
// from the fast tier. It satisfies testtiming.GuardLookup.
func (s Set) Guards(pkgDir, test string) bool {
	return s[pkgDir].Guards(test)
}

// Package is one directory's guard state.
type Package struct {
	// WholePackage is true when TestMain skips the package under -short, which
	// puts every test in it in tier 2.
	WholePackage bool
	guarded      map[string]bool
}

// Guards reports whether the named top-level test is excluded from the fast tier.
func (p *Package) Guards(test string) bool {
	if p == nil {
		return false
	}
	return p.WholePackage || p.guarded[test]
}

// Scan walks root and classifies every package directory that holds _test.go files.
// Keys are slash-separated paths relative to root ("kernel/ops").
func Scan(root string) (Set, error) {
	out := Set{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		if skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
			return fs.SkipDir
		}
		pkg, scanErr := scanDir(path)
		if scanErr != nil {
			return scanErr
		}
		if pkg != nil {
			out[relKey(root, path)] = pkg
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan %s for test guards: %w", root, err)
	}
	return out, nil
}

// ModulePath reads the module path from root's go.mod, so a package import path can
// be turned back into the directory Scan keyed it by.
func ModulePath(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("read go.mod in %s: %w", root, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if path, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(path), nil
		}
	}
	return "", fmt.Errorf("no module directive in %s/go.mod", root)
}

// relKey renders dir as a slash-separated path relative to root.
func relKey(root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return filepath.ToSlash(dir)
	}
	return filepath.ToSlash(rel)
}
