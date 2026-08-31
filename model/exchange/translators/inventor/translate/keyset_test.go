// SPDX-License-Identifier: GPL-2.0-only

package translate

import "testing"

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
