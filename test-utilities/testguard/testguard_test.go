// SPDX-License-Identifier: GPL-2.0-only

package testguard

import (
	"os"
	"path/filepath"
	"testing"
)

// writeModule lays out a throwaway module whose files exercise one guard shape each,
// so the scan is tested against source rather than against the repository it ships in.
func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files["go.mod"] = "module example.org/sample\n\ngo 1.27.0\n"
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return root
}

const directGuard = `package ops

import "testing"

func TestGuarded(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier")
	}
}

func TestPlain(t *testing.T) {}
`

const harnessGuard = `package feature

import "testing"

func runCorpus(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier")
	}
}

func TestViaHarness(t *testing.T) { runCorpus(t) }
`

const wholePackage = `package corpus

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if testing.Short() {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestAnything(t *testing.T) {}
`

func scanSample(t *testing.T) Set {
	t.Helper()
	root := writeModule(t, map[string]string{
		"ops/ops_test.go":         directGuard,
		"feature/feature_test.go": harnessGuard,
		"corpus/corpus_test.go":   wholePackage,
		"plain/plain.go":          "package plain\n",
	})
	set, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return set
}

func TestScanFindsADirectShortGuard(t *testing.T) {
	set := scanSample(t)
	if !set.Guards("ops", "TestGuarded") {
		t.Error("a test whose own body calls testing.Short() must read as guarded")
	}
}

func TestScanLeavesAnUnguardedTestInTheFastTier(t *testing.T) {
	if scanSample(t).Guards("ops", "TestPlain") {
		t.Error("a test with no guard must read as unguarded")
	}
}

func TestScanFollowsAGuardInsideAHarness(t *testing.T) {
	// The OCCT blend tests guard themselves through runCorpusGrids, one call deep;
	// a direct-body check alone would report them as unguarded.
	if !scanSample(t).Guards("feature", "TestViaHarness") {
		t.Error("a test that reaches testing.Short() through a same-package helper must read as guarded")
	}
}

func TestScanTreatsAShortSkippingTestMainAsWholePackage(t *testing.T) {
	set := scanSample(t)
	if !set["corpus"].WholePackage {
		t.Fatal("a TestMain that exits under -short must gate the whole package")
	}
	if !set.Guards("corpus", "TestAnythingAtAll") {
		t.Error("every test in a whole-package-gated directory must read as guarded")
	}
}

func TestScanSkipsDirectoriesWithNoTests(t *testing.T) {
	if _, ok := scanSample(t)["plain"]; ok {
		t.Error("a directory with no _test.go must not appear in the set")
	}
}

func TestGuardsOfAnUnknownPackageIsFalse(t *testing.T) {
	if scanSample(t).Guards("no/such/package", "TestWhatever") {
		t.Error("an unscanned package must not read as guarded")
	}
}

func TestScanReportsAMalformedTestFile(t *testing.T) {
	root := writeModule(t, map[string]string{"bad/bad_test.go": "package bad\n\nfunc ((("})
	if _, err := Scan(root); err == nil {
		t.Fatal("Scan() succeeded, want the parse error")
	}
}

func TestModulePathReadsTheModuleDirective(t *testing.T) {
	root := writeModule(t, map[string]string{})
	got, err := ModulePath(root)
	if err != nil || got != "example.org/sample" {
		t.Errorf("ModulePath() = %q, %v; want example.org/sample", got, err)
	}
}

func TestModulePathReportsAMissingGoMod(t *testing.T) {
	if _, err := ModulePath(t.TempDir()); err == nil {
		t.Fatal("ModulePath() succeeded, want an error with no go.mod")
	}
}
