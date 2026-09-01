// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"strings"
	"testing"

	"oblikovati.org/math"
)

// TestMatrixFromRecipe pins the three branches of the occurrence-transform decoder, including the
// corrupt-recipe guard whose error must name the offending cell count and the expected 16 (#785).
func TestMatrixFromRecipe(t *testing.T) {
	t.Parallel()
	if m, err := matrixFromRecipe(occurrenceRecipe{Name: "box:1"}); err != nil || m != math.Identity4() {
		t.Errorf("empty transform should decode to identity, got %v err=%v", m, err)
	}

	cells := math.Translation4(math.V3(1, 2, 3)).Cells()
	if m, err := matrixFromRecipe(occurrenceRecipe{Name: "box:1", Transform: cells[:]}); err != nil || m != math.Translation4(math.V3(1, 2, 3)) {
		t.Errorf("16-cell transform should round-trip, got %v err=%v", m, err)
	}

	_, err := matrixFromRecipe(occurrenceRecipe{Name: "box:1", Transform: []float64{1, 2, 3}})
	if err == nil {
		t.Fatal("a non-16-cell transform should be rejected")
	}
	if !strings.Contains(err.Error(), "3 cells") || !strings.Contains(err.Error(), "box:1") {
		t.Errorf("error should name the bad count and occurrence, got %q", err)
	}
}
