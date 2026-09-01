// SPDX-License-Identifier: GPL-2.0-only

// Package testimpact selects the Go packages whose tests a change set can affect.
//
// The full suite is dominated by a handful of packages (architecture/testing/03), so
// running `go test ./...` after touching one subsystem wastes minutes on packages the
// change cannot reach. This package maps changed files to their packages, then widens
// that seed set over the TEST-inclusive import graph: a package is impacted when its
// test binary transitively imports a seed.
//
//	sel := testimpact.NewSelector(root, testimpact.NewGoListLoader(root), testimpact.NewGitChanges(root, "origin/develop"))
//	pkgs, err := sel.Impacted() // ["oblikovati.org/kernel/ops", ...]
package testimpact

import "fmt"

// Package is one node of the module's package graph.
type Package struct {
	// ImportPath is the go list import path. Test variants keep their synthetic
	// form ("X.test", "X [X.test]"); NormalizeImportPath maps them back to X.
	ImportPath string
	// Dir is the absolute directory holding the package's files.
	Dir string
	// Deps is the TRANSITIVE import list go list reports for ImportPath.
	Deps []string
}

// GraphLoader supplies the module's package graph, test variants included.
type GraphLoader interface {
	LoadGraph() ([]Package, error)
}

// ChangeLister supplies the repository-relative paths a change set touches.
type ChangeLister interface {
	ChangedPaths() ([]string, error)
}

// globalPaths force a full run: they can change how every package builds or runs.
var globalPaths = []string{
	"go.mod", "go.sum", "go.work", "go.work.sum", "Makefile", ".golangci.yml", "tools.go",
}

// Selector turns a change set into the list of packages to test.
type Selector struct {
	root    string
	graph   GraphLoader
	changes ChangeLister
}

// NewSelector wires a selector to a module root and its two data sources.
func NewSelector(root string, graph GraphLoader, changes ChangeLister) *Selector {
	return &Selector{root: root, graph: graph, changes: changes}
}

// Impacted returns the sorted, de-duplicated import paths whose tests the change set
// can affect. It returns every package when the change set is empty or touches a file
// that has no package of its own (a global build input).
func (s *Selector) Impacted() ([]string, error) {
	pkgs, err := s.graph.LoadGraph()
	if err != nil {
		return nil, fmt.Errorf("load package graph: %w", err)
	}
	changed, err := s.changes.ChangedPaths()
	if err != nil {
		return nil, fmt.Errorf("list changed paths: %w", err)
	}
	if len(changed) == 0 || touchesGlobalInput(changed) {
		return allImportPaths(pkgs), nil
	}
	seeds := s.seedPackages(pkgs, changed)
	if len(seeds) == 0 {
		return nil, nil
	}
	return widen(pkgs, seeds), nil
}
