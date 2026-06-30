// SPDX-License-Identifier: GPL-2.0-only

package gridlayout

import (
	"testing"

	"oblikovati.org/api/types"
)

func fixed(px float64) types.GridTrack { return types.GridTrack{Kind: types.GridTrackFixed, Value: px} }
func fr(w float64) types.GridTrack     { return types.GridTrack{Kind: types.GridTrackFraction, Value: w} }
func auto() types.GridTrack            { return types.GridTrack{Kind: types.GridTrackAuto} }

func eq(t *testing.T, got []float64, want ...float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if d := got[i] - want[i]; d > 0.01 || d < -0.01 {
			t.Errorf("col %d = %.2f, want %.2f", i, got[i], want[i])
		}
	}
}

func TestResolveFixed(t *testing.T) {
	eq(t, ResolveColumnWidths([]types.GridTrack{fixed(100), fixed(50)}, 300, 0, nil), 100, 50)
}

func TestResolveFixedPlusFraction(t *testing.T) {
	eq(t, ResolveColumnWidths([]types.GridTrack{fixed(100), fr(1)}, 300, 0, nil), 100, 200)
}

func TestResolveFractionsByWeight(t *testing.T) {
	eq(t, ResolveColumnWidths([]types.GridTrack{fr(1), fr(2)}, 300, 0, nil), 100, 200)
}

func TestResolveAutoUsesMeasuredWidth(t *testing.T) {
	eq(t, ResolveColumnWidths([]types.GridTrack{auto(), fr(1)}, 300, 0, []float64{80, 0}), 80, 220)
}

func TestResolveSubtractsGaps(t *testing.T) {
	eq(t, ResolveColumnWidths([]types.GridTrack{fr(1), fr(1)}, 300, 20, nil), 140, 140)
}

func TestResolveClampsMax(t *testing.T) {
	capped := fr(1)
	capped.MaxPx = 50
	eq(t, ResolveColumnWidths([]types.GridTrack{capped, fr(1)}, 300, 0, nil), 50, 150)
}

// cells builds a placement slice: -1 colSpan means "no cell" (auto-flow).
func cells(specs ...[2]int) []*types.GridCell {
	out := make([]*types.GridCell, len(specs))
	for i, s := range specs {
		if s[1] < 0 {
			continue // nil = auto-flow
		}
		out[i] = &types.GridCell{Col: s[0], ColSpan: s[1]}
	}
	return out
}

func TestFlowAutoWraps(t *testing.T) {
	rows := FlowRows(cells([2]int{0, -1}, [2]int{0, -1}, [2]int{0, -1}, [2]int{0, -1}), 2)
	if len(rows) != 2 || len(rows[0]) != 2 || rows[1][0].Child != 2 || rows[1][1].Col != 1 {
		t.Fatalf("auto-flow rows = %+v", rows)
	}
}

func TestFlowSpanWrapsToOwnRow(t *testing.T) {
	// a (span1), b (span2 — can't fit beside a in 2 cols), c (span1).
	rows := FlowRows(cells([2]int{0, -1}, [2]int{0, 2}, [2]int{0, -1}), 2)
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %+v", rows)
	}
	if rows[1][0].ColSpan != 2 || rows[1][0].Child != 1 {
		t.Errorf("span row = %+v", rows[1])
	}
}

func TestFlowExplicitColGetsOwnRow(t *testing.T) {
	rows := FlowRows(cells([2]int{0, -1}, [2]int{1, 1}), 2)
	if len(rows) != 2 || rows[1][0].Col != 1 || rows[1][0].Child != 1 {
		t.Fatalf("explicit-col rows = %+v", rows)
	}
}
