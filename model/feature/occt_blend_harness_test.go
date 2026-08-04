// SPDX-License-Identifier: GPL-2.0-only

package feature_test

import (
	"testing"

	"oblikovati.org/model/feature/occtparity"
)

// runCorpusGrids runs every corpus case in the named grids through the real fillet feature and
// asserts OCCT's area. TODO / import-divergent / variable-radius cases skip themselves inside
// RunCase; the remaining reds are the ADR-0050 greening backlog and must never be loosened —
// this suite is the parity gate, so it stays red until the engine closes those gaps.
func runCorpusGrids(t *testing.T, grids ...string) {
	want := map[string]bool{}
	for _, g := range grids {
		want[g] = true
	}
	ran := 0
	for _, r := range occtparity.Corpus() {
		if !want[r.Grid] {
			continue
		}
		ran++
		t.Run(r.Case, func(t *testing.T) {
			occtparity.RunCase(t, r, occtparity.CorpusFixtureDir())
		})
	}
	if ran == 0 {
		t.Fatalf("no corpus cases for grids %v — corpus not generated?", grids)
	}
}
