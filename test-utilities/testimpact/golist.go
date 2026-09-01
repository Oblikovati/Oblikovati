// SPDX-License-Identifier: GPL-2.0-only

package testimpact

import (
	"fmt"
	"os/exec"
	"strings"
)

// graphFormat asks go list for one line per package: path, directory, transitive deps.
// The separators are characters no import path or directory may contain.
const graphFormat = `{{.ImportPath}}|{{.Dir}}|{{join .Deps " "}}`

// GoListLoader loads the package graph by running `go list -test` in a module root.
// -test adds the test variants, so a package that only reaches a change THROUGH a
// _test.go import is still selected.
type GoListLoader struct {
	root    string
	pattern string
}

// NewGoListLoader returns a loader for every package under root.
func NewGoListLoader(root string) *GoListLoader {
	return &GoListLoader{root: root, pattern: "./..."}
}

// LoadGraph runs go list and parses its output into the package graph.
func (l *GoListLoader) LoadGraph() ([]Package, error) {
	cmd := exec.Command(goBinary(), "list", "-test", "-e", "-f", graphFormat, l.pattern)
	cmd.Dir = l.root
	cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list -test %s in %s: %w", l.pattern, l.root, err)
	}
	return parseGraph(string(out)), nil
}

// parseGraph turns go list's line-per-package output into Packages, dropping any
// line that carries no directory (a synthetic node go list could not resolve).
func parseGraph(out string) []Package {
	var pkgs []Package
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|", 3)
		if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
			continue
		}
		pkg := Package{ImportPath: parts[0], Dir: parts[1]}
		if deps := strings.Fields(parts[2]); len(deps) > 0 {
			pkg.Deps = deps // leave nil, not empty, so a dep-free package compares equal to its zero value
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}
