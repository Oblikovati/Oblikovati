// SPDX-License-Identifier: GPL-2.0-only

package ops

import "testing"

// TestExtractObstacleIsClosedValence4 proves extractObstacle turns a real T6 ObstacleFeature into a
// 4-sided RailLoop whose sides chain end-to-start (RailLoop.Closed) — the precondition coons4Provider
// requires before it will even attempt a fill.
func TestExtractObstacleIsClosedValence4(t *testing.T) {
	of := newT6Obstacle(t)
	loop, ok := extractObstacle(of)
	if !ok {
		t.Fatal("extractObstacle declined T6")
	}
	if loop.Valence() != 4 {
		t.Fatalf("valence = %d, want 4", loop.Valence())
	}
	if !loop.Closed(blendScale().Weld()) {
		t.Fatal("loop not closed")
	}
}

// TestExtractObstacleResolvesToCoons4 proves the extracted loop fills via the general coons4 tier and
// passes the F2 probe (the corrected, non-folding sign).
func TestExtractObstacleResolvesToCoons4(t *testing.T) {
	of := newT6Obstacle(t)
	loop, _ := extractObstacle(of)
	fill, rails, sides, ok := coons4Fill(loop)
	if !ok {
		t.Fatal("coons4Fill declined the extracted obstacle loop")
	}
	if !ribbonSeamNonFolding(fill, rails, sides, blendScale()) {
		t.Fatal("extracted obstacle loop folds under coons4")
	}
}
