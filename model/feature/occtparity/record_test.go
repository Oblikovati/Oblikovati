// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

// A constant-radius record straight from the oracle (simple/A1: box 100³, one edge, r=10).
func TestParseOracleRecordGolden(t *testing.T) {
	t.Parallel()
	r, err := parseRecord([]byte(`{"grid":"simple","case":"A1","verb":"blend","expectedArea":59527.9,"deps":0.01,"todo":"","inputStep":"A1.step","picks":[{"radius":10,"locator":{"midpoint":[100,0,50],"direction":[0,0,1]},"law":null}]}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r.Grid != "simple" || r.Case != "A1" || r.Verb != "blend" {
		t.Fatalf("grid/case/verb: %+v", r)
	}
	if r.ExpectedArea != 59527.9 || r.Deps != 0.01 || r.InputStep != "A1.step" {
		t.Fatalf("area/deps/input: %+v", r)
	}
	if len(r.Picks) != 1 || r.Picks[0].Radius != 10 || r.Picks[0].Law != nil {
		t.Fatalf("picks: %+v", r.Picks)
	}
	if r.Picks[0].Locator.Midpoint != [3]float64{100, 0, 50} {
		t.Fatalf("midpoint: %v", r.Picks[0].Locator.Midpoint)
	}
	if r.Picks[0].Locator.Direction != [3]float64{0, 0, 1} {
		t.Fatalf("direction: %v", r.Picks[0].Locator.Direction)
	}
}

// A variable-radius record (buildevol/A1: law parses into (parameter, radius) pairs).
func TestParseOracleRecordBuildevolLaw(t *testing.T) {
	t.Parallel()
	r, err := parseRecord([]byte(`{"grid":"buildevol","case":"A1","verb":"buildevol","expectedArea":23985.2,"deps":0.01,"todo":"","inputStep":"buildevol_A1.step","picks":[{"radius":0,"locator":{"midpoint":[50,0,10],"direction":[1,0,0]},"law":[[0,2],[1,4],[2,2]]}]}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r.Verb != "buildevol" || len(r.Picks) != 1 {
		t.Fatalf("verb/picks: %+v", r)
	}
	want := [][2]float64{{0, 2}, {1, 4}, {2, 2}}
	got := r.Picks[0].Law
	if len(got) != len(want) {
		t.Fatalf("law length %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("law[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// A per-case exact-deviation record parses into a non-nil Deviation with the exact/OCCT areas and
// receipt; a record without the key defaults to nil (every genuine-parity case). areaTarget then
// picks OUR exact area for a deviation case and OCCT's ExpectedArea otherwise.
func TestParseOracleRecordDeviation(t *testing.T) {
	t.Parallel()
	r, err := parseRecord([]byte(`{"grid":"simple","case":"C8","verb":"blend","expectedArea":9640.68,"deps":0.01,"todo":"","inputStep":"simple/C8.step","deviation":{"exactArea":9781.45,"occtArea":9640.68,"reason":"OCCT sag"},"picks":[]}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r.Deviation == nil {
		t.Fatal("deviation record parsed with nil Deviation")
	}
	if r.Deviation.ExactArea != 9781.45 || r.Deviation.OCCTArea != 9640.68 || r.Deviation.Reason != "OCCT sag" {
		t.Fatalf("deviation fields: %+v", *r.Deviation)
	}
	if r.areaTarget() != 9781.45 {
		t.Fatalf("deviation areaTarget = %v, want the exact area 9781.45", r.areaTarget())
	}
	plain, _ := parseRecord([]byte(`{"grid":"simple","case":"A1","expectedArea":100,"deps":0.01,"picks":[]}`))
	if plain.Deviation != nil {
		t.Fatalf("no-deviation record got non-nil Deviation: %+v", plain.Deviation)
	}
	if plain.areaTarget() != 100 {
		t.Fatalf("plain areaTarget = %v, want ExpectedArea 100", plain.areaTarget())
	}
}

// A malformed record surfaces the offending input in the error (CLAUDE.md exception rule).
func TestParseOracleRecordRejectsGarbage(t *testing.T) {
	t.Parallel()
	_, err := parseRecord([]byte(`{"grid":`))
	if err == nil {
		t.Fatal("expected error on truncated JSON")
	}
}
