// SPDX-License-Identifier: GPL-2.0-only

// Package testtiming reads `go test -json` and ranks the suite by test time.
//
// The suite's cost is not spread over its 10,000 tests: 152 of them hold 94% of it
// (architecture/testing/03-test-tiers-and-selection.md). Ranking is how that stays
// visible, and OverBudget is how tier 1 keeps its promise to run in seconds.
//
//	runs, err := testtiming.Parse(os.Stdin)
//	for _, r := range testtiming.Slowest(runs, 25) { fmt.Println(r.Elapsed, r.Name) }
package testtiming

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// TestRun is one finished top-level test.
type TestRun struct {
	Package string
	Name    string
	Elapsed float64 // seconds
}

// PackageRun is one finished package: its WALL time, which is what a tier budget can
// hold. Per-test elapsed is not a budget under t.Parallel() — 32 tests sharing 32
// cores each report a stretched time while the package still finishes in seconds.
type PackageRun struct {
	Package string
	Elapsed float64 // seconds of wall clock
}

// String renders a run as one aligned report line.
func (r TestRun) String() string {
	return fmt.Sprintf("%8.2fs  %s :: %s", r.Elapsed, r.Package, r.Name)
}

// event is the subset of go's test2json record this package reads.
type event struct {
	Action  string
	Package string
	Test    string
	Elapsed float64
}

// Parse reads a `go test -json` stream and returns every top-level test that
// finished. Subtests are excluded: their time is already inside the parent, so
// counting both would double the total.
func Parse(r io.Reader) ([]TestRun, error) {
	runs, _, err := ParseAll(r)
	return runs, err
}

// ParseAll reads the stream once and returns both the test results and the package
// results, so a caller that needs both does not have to buffer the stream twice.
func ParseAll(r io.Reader) ([]TestRun, []PackageRun, error) {
	var runs []TestRun
	var pkgs []PackageRun
	scan := bufio.NewScanner(r)
	scan.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scan.Scan() {
		var e event
		if err := json.Unmarshal(scan.Bytes(), &e); err != nil {
			continue // go test interleaves non-JSON build output; skip it
		}
		switch {
		case isTopLevelResult(e):
			runs = append(runs, TestRun{Package: e.Package, Name: e.Test, Elapsed: e.Elapsed})
		case isPackageResult(e):
			pkgs = append(pkgs, PackageRun{Package: e.Package, Elapsed: e.Elapsed})
		}
	}
	if err := scan.Err(); err != nil {
		return nil, nil, fmt.Errorf("read go test -json stream: %w", err)
	}
	return runs, pkgs, nil
}

// isPackageResult reports whether the event is a finished package rather than a test.
func isPackageResult(e event) bool {
	return e.Test == "" && (e.Action == "pass" || e.Action == "fail")
}

// PackagesOverBudget returns the packages slower than budget seconds, longest first.
// This is the gate a tier can actually hold: package wall time is unaffected by how
// the tests inside were scheduled.
func PackagesOverBudget(pkgs []PackageRun, budget float64) []PackageRun {
	var out []PackageRun
	for _, p := range pkgs {
		if p.Elapsed > budget {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Elapsed != out[j].Elapsed {
			return out[i].Elapsed > out[j].Elapsed
		}
		return out[i].Package < out[j].Package
	})
	return out
}

// String renders a package result as one aligned report line.
func (p PackageRun) String() string {
	return fmt.Sprintf("%8.2fs  %s", p.Elapsed, p.Package)
}

// isTopLevelResult reports whether the event is a finished test that is not a subtest.
func isTopLevelResult(e event) bool {
	if e.Test == "" || strings.Contains(e.Test, "/") {
		return false
	}
	return e.Action == "pass" || e.Action == "fail"
}

// Slowest returns the n slowest runs, longest first. n <= 0 returns them all.
func Slowest(runs []TestRun, n int) []TestRun {
	out := append([]TestRun(nil), runs...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Elapsed != out[j].Elapsed {
			return out[i].Elapsed > out[j].Elapsed
		}
		return out[i].Package+out[i].Name < out[j].Package+out[j].Name
	})
	if n > 0 && n < len(out) {
		out = out[:n]
	}
	return out
}

// OverBudget returns the runs slower than budget seconds, longest first. A tier-1
// run with a non-empty result has an unguarded corpus test in it.
func OverBudget(runs []TestRun, budget float64) []TestRun {
	var out []TestRun
	for _, r := range runs {
		if r.Elapsed > budget {
			out = append(out, r)
		}
	}
	return Slowest(out, 0)
}

// Total returns the summed test time, the suite's cost independent of how many
// cores ran it.
func Total(runs []TestRun) float64 {
	var sum float64
	for _, r := range runs {
		sum += r.Elapsed
	}
	return sum
}

// GuardLookup answers whether a test excludes itself from the fast tier. It is an
// interface so this package does not depend on how that is decided (a static scan,
// today, in test-utilities/testguard).
type GuardLookup interface {
	Guards(pkgDir, test string) bool
}

// UnguardedOverBudget returns the tests slower than budget seconds that do NOT guard
// themselves, longest first — a corpus test that forgot its testing.Short().
//
// It reads a TIER 2 run, which is why the budget must be generous: elapsed time there
// is stretched by core contention, so the honest tests and the corpus tests are only
// separated by a wide margin, not a tight one.
func UnguardedOverBudget(runs []TestRun, modulePath string, guards GuardLookup, budget float64) []TestRun {
	var out []TestRun
	for _, r := range runs {
		if r.Elapsed > budget && !guards.Guards(packageDir(r.Package, modulePath), r.Name) {
			out = append(out, r)
		}
	}
	return Slowest(out, 0)
}

// packageDir turns an import path back into its directory relative to the module root.
func packageDir(importPath, modulePath string) string {
	if importPath == modulePath {
		return "."
	}
	return strings.TrimPrefix(importPath, modulePath+"/")
}

// SlowestUnguarded returns the slowest test that does NOT exclude itself from the fast
// tier, or false when every test is guarded. It is what makes the guard budget's HEADROOM
// visible: a gate that only speaks when it fails cannot tell you it is about to.
func SlowestUnguarded(runs []TestRun, modulePath string, guards GuardLookup) (TestRun, bool) {
	var worst TestRun
	found := false
	for _, r := range runs {
		if guards.Guards(packageDir(r.Package, modulePath), r.Name) {
			continue
		}
		if !found || r.Elapsed > worst.Elapsed {
			worst, found = r, true
		}
	}
	return worst, found
}
