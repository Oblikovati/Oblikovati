// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"os"
	"path/filepath"
	"testing"
)

// wantCorpusSize is the OCCT tests/blend case count across all 7 grids, verified by walking
// ../OCCT/tests/blend/<grid>/<case> and matching case files ^[A-Z][0-9]+$ (the plan's
// documented "477" is stale — see docs/superpowers/plans/2026-07-11-occt-blend-parity-corpus.md
// Task 9 note in the generator invocation).
const wantCorpusSize = 475

// TestCorpusIsComplete asserts the generated corpus.go carries every OCCT blend case (no case
// silently dropped), that each record identifies itself, that non-TODO records carry a real
// reference area and at least one pick, and that any committed STEP fixture a record points at
// actually exists on disk.
func TestCorpusIsComplete(t *testing.T) {
	t.Parallel()
	c := Corpus()
	if len(c) != wantCorpusSize {
		t.Fatalf("corpus has %d cases, want %d", len(c), wantCorpusSize)
	}
	for _, r := range c {
		assertRecordIdentified(t, r)
		assertNonTODORecordComplete(t, r)
		assertFixturePresent(t, r)
	}
}

// assertRecordIdentified fails if a record can't be traced back to its OCCT case.
func assertRecordIdentified(t *testing.T, r Record) {
	t.Helper()
	if r.Grid == "" || r.Case == "" {
		t.Fatalf("case missing grid/case: %+v", r)
	}
}

// assertNonTODORecordComplete fails if a case that OCCT itself did not mark TODO is missing
// the data RunCase needs (reference area, picks) — that would silently no-op the case.
func assertNonTODORecordComplete(t *testing.T, r Record) {
	t.Helper()
	if r.TODO != "" {
		return
	}
	if r.ExpectedArea <= 0 {
		t.Fatalf("%s/%s: non-TODO but expectedArea<=0", r.Grid, r.Case)
	}
	if len(r.Picks) == 0 {
		t.Fatalf("%s/%s: non-TODO but no picks", r.Grid, r.Case)
	}
}

// assertFixturePresent fails if a record names a STEP fixture that was not actually committed.
func assertFixturePresent(t *testing.T, r Record) {
	t.Helper()
	if r.InputStep == "" {
		return
	}
	path := filepath.Join(CorpusFixtureDir(), r.InputStep)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("%s/%s: fixture %s missing: %v", r.Grid, r.Case, path, err)
	}
}
