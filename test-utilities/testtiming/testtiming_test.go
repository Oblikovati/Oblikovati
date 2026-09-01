// SPDX-License-Identifier: GPL-2.0-only

package testtiming

import (
	"reflect"
	"strings"
	"testing"
)

// stream is a small `go test -json` transcript: two packages, one subtest, one
// build line that is not JSON at all.
const stream = `{"Action":"run","Package":"m/ops","Test":"TestFast"}
{"Action":"pass","Package":"m/ops","Test":"TestFast","Elapsed":0.01}
{"Action":"pass","Package":"m/ops","Test":"TestSlow/case_a","Elapsed":3}
{"Action":"pass","Package":"m/ops","Test":"TestSlow","Elapsed":7.5}
{"Action":"fail","Package":"m/geom","Test":"TestBroken","Elapsed":2.25}
{"Action":"pass","Package":"m/geom","Elapsed":9.9}
# m/geom compile output that is not JSON
`

func parsed(t *testing.T) []TestRun {
	t.Helper()
	runs, err := Parse(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return runs
}

func TestParseKeepsTopLevelResultsOnly(t *testing.T) {
	got := parsed(t)
	want := []TestRun{
		{Package: "m/ops", Name: "TestFast", Elapsed: 0.01},
		{Package: "m/ops", Name: "TestSlow", Elapsed: 7.5},
		{Package: "m/geom", Name: "TestBroken", Elapsed: 2.25},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse() = %+v, want %+v", got, want)
	}
}

func TestParseSurvivesNonJSONBuildOutput(t *testing.T) {
	if len(parsed(t)) != 3 {
		t.Error("a non-JSON build line must not stop the parse")
	}
}

func TestSlowestRanksLongestFirst(t *testing.T) {
	got := Slowest(parsed(t), 2)
	if len(got) != 2 || got[0].Name != "TestSlow" || got[1].Name != "TestBroken" {
		t.Errorf("Slowest() = %+v, want TestSlow then TestBroken", got)
	}
}

func TestSlowestBreaksTiesByName(t *testing.T) {
	runs := []TestRun{
		{Package: "m/b", Name: "TestX", Elapsed: 1},
		{Package: "m/a", Name: "TestX", Elapsed: 1},
	}
	if got := Slowest(runs, 0); got[0].Package != "m/a" {
		t.Errorf("Slowest() = %+v, want m/a first — the order must be reproducible", got)
	}
}

func TestOverBudgetNamesOnlyTheTestsPastTheLimit(t *testing.T) {
	got := OverBudget(parsed(t), 2)
	if len(got) != 2 || got[0].Name != "TestSlow" || got[1].Name != "TestBroken" {
		t.Errorf("OverBudget(2s) = %+v, want TestSlow and TestBroken", got)
	}
}

func TestOverBudgetIsEmptyWhenEveryTestIsFast(t *testing.T) {
	if got := OverBudget(parsed(t), 60); len(got) != 0 {
		t.Errorf("OverBudget(60s) = %+v, want none", got)
	}
}

func TestTotalSumsTopLevelTestTime(t *testing.T) {
	if got := Total(parsed(t)); got != 9.76 {
		t.Errorf("Total() = %v, want 9.76", got)
	}
}

func TestStringRendersAnAlignedReportLine(t *testing.T) {
	line := TestRun{Package: "m/ops", Name: "TestSlow", Elapsed: 7.5}.String()
	if !strings.Contains(line, "7.50s") || !strings.HasSuffix(line, "m/ops :: TestSlow") {
		t.Errorf("String() = %q", line)
	}
}

func TestParseAllReadsPackageResultsToo(t *testing.T) {
	_, pkgs, err := ParseAll(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("ParseAll: %v", err)
	}
	want := []PackageRun{{Package: "m/geom", Elapsed: 9.9}}
	if !reflect.DeepEqual(pkgs, want) {
		t.Errorf("ParseAll packages = %+v, want %+v", pkgs, want)
	}
}

func TestPackagesOverBudgetNamesTheSlowPackages(t *testing.T) {
	pkgs := []PackageRun{
		{Package: "m/fast", Elapsed: 1},
		{Package: "m/slow", Elapsed: 90},
		{Package: "m/slower", Elapsed: 120},
	}
	got := PackagesOverBudget(pkgs, 60)
	if len(got) != 2 || got[0].Package != "m/slower" || got[1].Package != "m/slow" {
		t.Errorf("PackagesOverBudget(60s) = %+v, want m/slower then m/slow", got)
	}
}

func TestPackagesOverBudgetIsEmptyWhenEveryPackageIsFast(t *testing.T) {
	pkgs := []PackageRun{{Package: "m/fast", Elapsed: 1}}
	if got := PackagesOverBudget(pkgs, 60); len(got) != 0 {
		t.Errorf("PackagesOverBudget(60s) = %+v, want none", got)
	}
}

func TestPackageRunStringRendersAnAlignedLine(t *testing.T) {
	if got := (PackageRun{Package: "m/ops", Elapsed: 12.5}).String(); !strings.Contains(got, "12.50s") {
		t.Errorf("PackageRun.String() = %q", got)
	}
}
