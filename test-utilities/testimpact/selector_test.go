// SPDX-License-Identifier: GPL-2.0-only

package testimpact

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

// FakeGraphLoader serves a fixed package graph, so a selection test needs no module
// on disk and no `go list`.
type FakeGraphLoader struct {
	Packages []Package
	Err      error
}

func (f *FakeGraphLoader) LoadGraph() ([]Package, error) { return f.Packages, f.Err }

// FakeChangeLister serves a fixed change set in place of git.
type FakeChangeLister struct {
	Paths []string
	Err   error
}

func (f *FakeChangeLister) ChangedPaths() ([]string, error) { return f.Paths, f.Err }

// testGraph is a three-package module: geom <- ops <- feature, plus a leaf nobody imports.
func testGraph(root string) []Package {
	j := func(p string) string { return filepath.Join(root, filepath.FromSlash(p)) }
	return []Package{
		{ImportPath: "m/geom", Dir: j("geom")},
		{ImportPath: "m/ops", Dir: j("ops"), Deps: []string{"m/geom"}},
		{ImportPath: "m/feature", Dir: j("feature"), Deps: []string{"m/geom", "m/ops"}},
		{ImportPath: "m/theme", Dir: j("theme")},
	}
}

func selectorFor(root string, changed []string) *Selector {
	return NewSelector(root,
		&FakeGraphLoader{Packages: testGraph(root)},
		&FakeChangeLister{Paths: changed})
}

func TestImpactedIncludesTheChangedPackageAndItsDependents(t *testing.T) {
	got, err := selectorFor("/m", []string{"geom/plane.go"}).Impacted()
	if err != nil {
		t.Fatalf("Impacted: %v", err)
	}
	want := []string{"m/feature", "m/geom", "m/ops"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Impacted() = %v, want %v", got, want)
	}
}

func TestImpactedOfALeafPackageIsOnlyThatPackage(t *testing.T) {
	got, _ := selectorFor("/m", []string{"theme/dark.go"}).Impacted()
	want := []string{"m/theme"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Impacted() = %v, want %v", got, want)
	}
}

func TestImpactedAttributesATestdataFileToItsPackage(t *testing.T) {
	got, _ := selectorFor("/m", []string{"ops/testdata/corpus/case.json"}).Impacted()
	want := []string{"m/feature", "m/ops"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Impacted() = %v, want %v", got, want)
	}
}

func TestImpactedIsEverythingWhenAGlobalBuildInputChanges(t *testing.T) {
	got, _ := selectorFor("/m", []string{"go.mod"}).Impacted()
	want := []string{"m/feature", "m/geom", "m/ops", "m/theme"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Impacted() = %v, want %v", got, want)
	}
}

func TestImpactedIsEverythingWhenNothingChanged(t *testing.T) {
	got, _ := selectorFor("/m", nil).Impacted()
	if len(got) != 4 {
		t.Errorf("Impacted() = %v, want all four packages", got)
	}
}

func TestImpactedIsEmptyWhenNoPackageOwnsTheChange(t *testing.T) {
	got, err := selectorFor("/m", []string{"architecture/README.md"}).Impacted()
	if err != nil {
		t.Fatalf("Impacted: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Impacted() = %v, want no packages", got)
	}
}

func TestImpactedReportsAGraphLoadFailure(t *testing.T) {
	sel := NewSelector("/m",
		&FakeGraphLoader{Err: errors.New("go list exploded")},
		&FakeChangeLister{Paths: []string{"geom/plane.go"}})
	if _, err := sel.Impacted(); err == nil {
		t.Fatal("Impacted() succeeded, want the loader error")
	}
}

func TestImpactedReportsAChangeListFailure(t *testing.T) {
	sel := NewSelector("/m",
		&FakeGraphLoader{Packages: testGraph("/m")},
		&FakeChangeLister{Err: errors.New("not a git repository")})
	if _, err := sel.Impacted(); err == nil {
		t.Fatal("Impacted() succeeded, want the change-lister error")
	}
}
