// SPDX-License-Identifier: GPL-2.0-only

package testimpact

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestNormalizeImportPathMapsEveryTestVariantBack(t *testing.T) {
	cases := map[string]string{
		"oblikovati.org/kernel/ops":                                       "oblikovati.org/kernel/ops",
		"oblikovati.org/kernel/ops.test":                                  "oblikovati.org/kernel/ops",
		"oblikovati.org/kernel/ops [oblikovati.org/kernel/ops.test]":      "oblikovati.org/kernel/ops",
		"oblikovati.org/kernel/ops_test [oblikovati.org/kernel/ops.test]": "oblikovati.org/kernel/ops",
	}
	for in, want := range cases {
		if got := NormalizeImportPath(in); got != want {
			t.Errorf("NormalizeImportPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseGraphReadsPathDirAndDeps(t *testing.T) {
	got := parseGraph("m/ops|/src/ops|m/geom m/topo\nm/geom|/src/geom|\n")
	want := []Package{
		{ImportPath: "m/ops", Dir: "/src/ops", Deps: []string{"m/geom", "m/topo"}},
		{ImportPath: "m/geom", Dir: "/src/geom"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseGraph() = %+v, want %+v", got, want)
	}
}

func TestParseGraphDropsLinesWithoutADirectory(t *testing.T) {
	if got := parseGraph("m/broken||\n\ngarbage\n"); len(got) != 0 {
		t.Errorf("parseGraph() = %+v, want none", got)
	}
}

func TestSelectionFollowsATestOnlyImport(t *testing.T) {
	// feature's TEST binary imports ops even though the package itself does not:
	// go list -test reports that edge on the "m/feature.test" node.
	j := func(p string) string { return filepath.Join("/m", p) }
	pkgs := []Package{
		{ImportPath: "m/ops", Dir: j("ops")},
		{ImportPath: "m/feature", Dir: j("feature")},
		{ImportPath: "m/feature.test", Dir: j("feature"), Deps: []string{"m/ops"}},
	}
	sel := NewSelector("/m", &FakeGraphLoader{Packages: pkgs},
		&FakeChangeLister{Paths: []string{"ops/boolean.go"}})
	got, _ := sel.Impacted()
	want := []string{"m/feature", "m/ops"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Impacted() = %v, want %v", got, want)
	}
}

func TestTouchesGlobalInputIgnoresANestedFileOfTheSameName(t *testing.T) {
	if touchesGlobalInput([]string{"head/go.mod"}) {
		t.Error("a nested go.mod must not force a full run of the root module")
	}
	if !touchesGlobalInput([]string{"go.mod"}) {
		t.Error("the root go.mod must force a full run")
	}
}
