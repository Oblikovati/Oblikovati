// SPDX-License-Identifier: GPL-2.0-only

package translate

import "testing"

// TestLessKey covers the 3-level lexicographic vertex-key ordering, including each tie-break level.
func TestLessKey(t *testing.T) {
	cases := []struct {
		a, b [3]int64
		want bool
	}{
		{[3]int64{1, 0, 0}, [3]int64{2, 0, 0}, true},  // first component decides
		{[3]int64{2, 0, 0}, [3]int64{1, 9, 9}, false}, // first component decides the other way
		{[3]int64{1, 1, 0}, [3]int64{1, 2, 0}, true},  // tie on first, second decides
		{[3]int64{1, 1, 1}, [3]int64{1, 1, 2}, true},  // tie on first two, third decides
		{[3]int64{1, 1, 1}, [3]int64{1, 1, 1}, false}, // equal keys are not less
	}
	for _, c := range cases {
		if got := lessKey(c.a, c.b); got != c.want {
			t.Errorf("lessKey(%v,%v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// TestSameCurveSet covers equal, differing, length-mismatched, and empty curve-key lists.
func TestSameCurveSet(t *testing.T) {
	if !sameCurveSet([]string{"a", "b"}, []string{"a", "b"}) {
		t.Error("identical non-empty sets should be equal")
	}
	if sameCurveSet([]string{"a", "b"}, []string{"a", "c"}) {
		t.Error("sets differing in an element are not equal")
	}
	if sameCurveSet([]string{"a"}, []string{"a", "b"}) {
		t.Error("sets of different length are not equal")
	}
	if sameCurveSet(nil, nil) {
		t.Error("empty sets are treated as not-equal by design (no curves to match)")
	}
}
