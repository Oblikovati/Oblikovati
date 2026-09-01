// SPDX-License-Identifier: GPL-2.0-only
package blend

import "testing"

// fakeVerdict scripts pass/fail per candidate composition, standing in for
// obstacleImprovedSolid (which needs a real assembled *topo.Body). Named fake.
type fakeVerdict struct{ pass map[rebuildChoice]bool }

func (f fakeVerdict) improved(c rebuildChoice) bool { return f.pass[c] }

func TestChooseRebuild_RescuesObstacleWhenRunoutOpensShell(t *testing.T) {
	t.Parallel()
	v := fakeVerdict{pass: map[rebuildChoice]bool{
		chooseBoth: false, chooseObstacle: true, chooseRunout: true, chooseBaseline: true}}
	if got := chooseRebuild(v.improved); got != chooseObstacle {
		t.Fatalf("chooseRebuild = %v, want chooseObstacle (obstacle survives a failing {both})", got)
	}
}
func TestChooseRebuild_PrefersBothWhenClean(t *testing.T) {
	t.Parallel()
	v := fakeVerdict{pass: map[rebuildChoice]bool{chooseBoth: true}}
	if got := chooseRebuild(v.improved); got != chooseBoth {
		t.Fatalf("chooseRebuild = %v, want chooseBoth", got)
	}
}
func TestChooseRebuild_FallsToBaseline(t *testing.T) {
	t.Parallel()
	if got := chooseRebuild(fakeVerdict{pass: map[rebuildChoice]bool{}}.improved); got != chooseBaseline {
		t.Fatalf("chooseRebuild = %v, want chooseBaseline", got)
	}
}
