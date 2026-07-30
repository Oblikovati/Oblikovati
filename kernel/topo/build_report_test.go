// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"testing"

	"oblikovati.org/kernel/diag"
)

// TestBuildReportTravelsWithTheBody pins the build-report channel: what an assembler could not do
// ideally must reach the caller ON the body it produced, in emission order, and the accessor must
// hand back a COPY so a reader cannot mutate a finished body's report. A body whose assembler
// recorded nothing reports nothing — the overwhelmingly common case, which must stay allocation-free
// of surprises.
func TestBuildReportTravelsWithTheBody(t *testing.T) {
	bld := NewBuilder(true, NewLineage(Tok("t", "body", 0)))
	if got := bld.Build().BuildDiagnostics(); len(got) != 0 {
		t.Fatalf("a clean assembly reported %v, want nothing", got)
	}

	bld = NewBuilder(true, NewLineage(Tok("t", "body", 1)))
	bld.Diagnose(diag.Diagnostic{Code: "t.first", Severity: diag.Warning, Detail: "one"})
	bld.Diagnose(diag.Diagnostic{Code: "t.second", Severity: diag.Defect, Detail: "two"})
	body := bld.Build()

	got := body.BuildDiagnostics()
	if len(got) != 2 || got[0].Code != "t.first" || got[1].Code != "t.second" {
		t.Fatalf("build report = %v, want t.first then t.second in emission order", got)
	}
	got[0].Code = "mutated"
	if again := body.BuildDiagnostics(); again[0].Code != "t.first" {
		t.Errorf("mutating the returned slice changed the body's own report to %q — BuildDiagnostics must copy", again[0].Code)
	}
}
