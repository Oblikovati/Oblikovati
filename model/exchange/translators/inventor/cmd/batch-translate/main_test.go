// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunClassifiesModuleTestdata runs the batch over the module's own testdata and checks the
// parametric-state classifier: 16_revolve rebuilds a SOLID, and a sketch-only part lands as
// SKETCH (extraction decoupled from any feature). This exercises run → translateOne → classify
// end-to-end without a network drive.
func TestRunClassifiesModuleTestdata(t *testing.T) {
	inDir := filepath.Join("..", "..", "testdata")
	rows, tally := run(inDir, t.TempDir())
	if len(rows) == 0 {
		t.Fatal("no .ipt files classified from testdata")
	}
	got := map[string]string{}
	for _, r := range rows {
		got[filepath.Base(r.rel)] = r.outcome
	}
	if got["16_revolve.ipt"] != "SOLID" {
		t.Errorf("16_revolve.ipt classified %q, want SOLID", got["16_revolve.ipt"])
	}
	if got["sketch_line.ipt"] != "SKETCH" {
		t.Errorf("sketch_line.ipt classified %q, want SKETCH (sketch emitted, no feature)", got["sketch_line.ipt"])
	}
	if tally["SOLID"] == 0 {
		t.Errorf("expected at least one SOLID part in the tally, got %v", tally)
	}
}

// TestFeatureTagsMarksRevolve confirms the coverage hint surfaces a part's feature types even
// when they wouldn't build — 16_revolve carries a revolve.
func TestFeatureTagsMarksRevolve(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "16_revolve.ipt"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if tags := featureTags(data); !strings.Contains(tags, "rev") {
		t.Errorf("featureTags = %q, want it to contain \"rev\"", tags)
	}
}

// TestWriteReportHasHeader checks the TSV writer emits the column header and one line per row.
func TestWriteReportHasHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.tsv")
	if err := writeReport(path, []row{{outcome: "SOLID", volumeMm3: 8000, sketches: 1, features: 1, dof: 3, eqs: 5, featTags: "ext", rel: "a.ipt"}}); err != nil {
		t.Fatalf("writeReport: %v", err)
	}
	b, _ := os.ReadFile(path)
	text := string(b)
	if !strings.HasPrefix(text, "outcome\tvolume_mm3\tsketches\tfeatures\tdof\teqs\tfeat_tags\tpart\n") {
		t.Errorf("report missing header, got:\n%s", text)
	}
	if !strings.Contains(text, "SOLID\t8000\t1\t1\t3\t5\text\ta.ipt") {
		t.Errorf("report missing data row, got:\n%s", text)
	}
}

// TestPct covers the percentage helper, including the divide-by-zero guard.
func TestPct(t *testing.T) {
	if got := pct(1, 4); got != 25 {
		t.Errorf("pct(1,4) = %v, want 25", got)
	}
	if got := pct(3, 0); got != 0 {
		t.Errorf("pct(3,0) = %v, want 0 (no divide by zero)", got)
	}
}

// TestPrintSummariesDoNotPanic exercises the stdout summary helpers on a small synthetic library —
// they format aggregates and must handle both populated and empty inputs without panicking.
func TestPrintSummariesDoNotPanic(t *testing.T) {
	rows := []row{
		{sketches: 2, constrained: 1, eqs: 5, dof: 3},
		{sketches: 0, constrained: 0, eqs: 0, dof: 0},
	}
	printConstraintStats(rows)
	printConstraintStats(nil) // empty library: the pct guards must hold
	printTally("out.tsv", 3, map[string]int{"SOLID": 2, "SKETCH": 1})
}
