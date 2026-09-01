// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/math"
)

// TestFilletIntoExistingRoundRejectedHonestly_1797 pins the kernel guard for Discord #1797: round a
// box's top rim (leaving cylinder faces), then fillet the four verticals — each is plane∩plane but
// its TOP vertex runs into the top-rim cylinders, a fillet-meets-fillet corner the planar blend can't
// close. It must fail with an actionable, rounded-cause message — never the misleading "not a valid
// solid" that shipped a facet-cage octagon.
func TestFilletIntoExistingRoundRejectedHonestly_1797(t *testing.T) {
	t.Parallel()
	box := csgBox(math.P3(0, 0, 0), 4, 4, 4)
	rounded, err := blend.FilletEdges(box, topPerimeterKeys(t, box), 0.5)
	if err != nil {
		t.Fatalf("top-rim fillet setup: %v", err)
	}

	_, err = blend.FilletEdges(rounded, verticalEdgeKeys(t, rounded), 0.5)
	if err == nil {
		t.Fatal("filleting an edge that runs into an existing round must be rejected, got nil error")
	}
	if strings.Contains(err.Error(), "not a valid solid") {
		t.Errorf("misleading message — should name the rounded cause, got: %v", err)
	}
	if !strings.Contains(err.Error(), "rounded") {
		t.Errorf("message should name the rounded cause (so the user knows to fillet these first), got: %v", err)
	}
}

// TestFilletAdjacentEdgesTogetherNotRejected guards against over-rejection: filleting two adjacent
// edges in ONE op (their shared corner) must NOT trip the endpoint guard — the adjacent round does
// not exist yet, so it is this op's own corner, solved normally.
func TestFilletAdjacentEdgesTogetherNotRejected(t *testing.T) {
	t.Parallel()
	box := csgBox(math.P3(0, 0, 0), 4, 4, 4)
	top := topPerimeterKeys(t, box)
	if len(top) < 2 {
		t.Fatalf("plain box top edges = %d, want ≥2", len(top))
	}
	if _, err := blend.FilletEdges(box, top[:2], 0.5); err != nil {
		t.Fatalf("filleting two adjacent edges together must succeed, got: %v", err)
	}
}
