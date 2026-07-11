// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"sort"
	"testing"
)

// Scoreboard tallies a fake evaluator's verdicts.
func TestScoreboardTally(t *testing.T) {
	recs := []Record{{Grid: "simple"}, {Grid: "simple"}, {Grid: "simple", TODO: "x"}}
	got := Scoreboard(recs, func(r Record) Outcome {
		if r.TODO != "" {
			return SkipTODO
		}
		return Pass
	})
	if got[Pass] != 2 || got[SkipTODO] != 1 {
		t.Fatalf("tally: %+v", got)
	}
}

// TestOCCTBlendScoreboard prints a per-grid PASS/FAIL/SKIP table over the real corpus. It is
// NON-GATING (the per-grid table tests in package feature_test are the gate); it just reports
// the parity landscape so the greening backlog is visible at a glance.
func TestOCCTBlendScoreboard(t *testing.T) {
	dir := CorpusFixtureDir()
	byGrid := map[string]map[Outcome]int{}
	for _, r := range Corpus() {
		if byGrid[r.Grid] == nil {
			byGrid[r.Grid] = map[Outcome]int{}
		}
		byGrid[r.Grid][ScoreCase(r, dir)]++
	}

	grids := make([]string, 0, len(byGrid))
	for g := range byGrid {
		grids = append(grids, g)
	}
	sort.Strings(grids)

	order := []Outcome{Pass, FailArea, FailFaulty, SkipTODO, SkipImportDivergence, Incomplete}
	total := map[Outcome]int{}
	t.Logf("OCCT blend parity scoreboard (non-gating):")
	for _, g := range grids {
		line := g
		for _, o := range order {
			n := byGrid[g][o]
			total[o] += n
			line += "  " + o.String() + "=" + itoa(n)
		}
		t.Log(line)
	}
	sum := 0
	tline := "TOTAL"
	for _, o := range order {
		tline += "  " + o.String() + "=" + itoa(total[o])
		sum += total[o]
	}
	t.Log(tline)
	if sum != len(Corpus()) {
		t.Fatalf("scoreboard counted %d cases, want %d", sum, len(Corpus()))
	}
}

// itoa avoids importing strconv for one call.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
