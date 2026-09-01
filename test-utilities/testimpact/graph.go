// SPDX-License-Identifier: GPL-2.0-only

package testimpact

import (
	"path/filepath"
	"sort"
	"strings"
)

// NormalizeImportPath maps a go list TEST variant back to the package it belongs to.
// go list -test reports three extra nodes per tested package: "X.test" (the binary),
// "X [X.test]" (X recompiled with its internal tests) and "X_test [X.test]" (the
// external test package). All three mean "run the tests of X".
func NormalizeImportPath(path string) string {
	if i := strings.Index(path, " ["); i >= 0 {
		path = path[:i]
	}
	path = strings.TrimSuffix(path, ".test")
	return strings.TrimSuffix(path, "_test")
}

// touchesGlobalInput reports whether any changed path is a build input shared by
// every package, in which case no selection is safe.
func touchesGlobalInput(changed []string) bool {
	for _, p := range changed {
		for _, g := range globalPaths {
			if filepath.ToSlash(p) == g {
				return true
			}
		}
	}
	return false
}

// allImportPaths returns every real (non-test-variant) import path in the graph.
func allImportPaths(pkgs []Package) []string {
	set := map[string]bool{}
	for _, p := range pkgs {
		set[NormalizeImportPath(p.ImportPath)] = true
	}
	return sortedKeys(set)
}

// seedPackages returns the import paths that OWN a changed file. A file under a
// testdata/ tree (or any directory with no package of its own) is attributed to the
// nearest enclosing package directory.
func (s *Selector) seedPackages(pkgs []Package, changed []string) map[string]bool {
	byDir := map[string]string{}
	for _, p := range pkgs {
		byDir[filepath.Clean(p.Dir)] = NormalizeImportPath(p.ImportPath)
	}
	seeds := map[string]bool{}
	for _, rel := range changed {
		if path := s.owningPackage(byDir, rel); path != "" {
			seeds[path] = true
		}
	}
	return seeds
}

// owningPackage walks up from a changed file's directory to the first directory that
// is a package, stopping at the module root. It returns "" when nothing owns the file.
func (s *Selector) owningPackage(byDir map[string]string, rel string) string {
	dir := filepath.Dir(filepath.Join(s.root, filepath.FromSlash(rel)))
	root := filepath.Clean(s.root)
	for {
		if path, ok := byDir[filepath.Clean(dir)]; ok {
			return path
		}
		if filepath.Clean(dir) == root {
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// widen grows the seed set to every package whose TEST binary transitively imports a
// seed. The seeds themselves are always included.
func widen(pkgs []Package, seeds map[string]bool) []string {
	out := map[string]bool{}
	for seed := range seeds {
		out[seed] = true
	}
	for _, p := range pkgs {
		if dependsOnSeed(p, seeds) {
			out[NormalizeImportPath(p.ImportPath)] = true
		}
	}
	return sortedKeys(out)
}

// dependsOnSeed reports whether p transitively imports any seed package.
func dependsOnSeed(p Package, seeds map[string]bool) bool {
	for _, dep := range p.Deps {
		if seeds[NormalizeImportPath(dep)] {
			return true
		}
	}
	return false
}

// sortedKeys returns the set's keys in a stable order, so the selection is
// reproducible run to run (the suite's determinism rule).
func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
