// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"
)

// The lane anchors' own TRACKING test (ADR-0066, Oblikovati#3515).
//
// torusLaneAnchors seeds one azimuth per extremum track at v = 0 and then requires every later station
// to present the same tracks — the certificate that a branch pair stays on the lane it was named by.
// The predicate that says so is anglesTrackSeeds, and its three refusals are the only thing between a
// labelled lane and a guessed one: the extremum COUNT changed, a track DRIFTED further than half the
// seeds' own spacing, or two seeds CLAIMED the same extremum. Every one of them was reached only
// through a whole surface pair, where a refusal reads as "this pair demoted" and says nothing about
// which of the three fired (#3527).

// TestAnglesTrackSeedsAcceptsTheTracksItWasSeededOn is the positive row: the same angles, and angles
// each within reach of their own seed, track.
func TestAnglesTrackSeedsAcceptsTheTracksItWasSeededOn(t *testing.T) {
	t.Parallel()
	seed := []float64{0.2, 1.7, 3.4, 5.0}
	if !anglesTrackSeeds(seed, seed, 0.1) {
		t.Error("a station presenting its own seeds does not track them")
	}
	drifted := []float64{0.24, 1.66, 3.44, 4.96}
	if !anglesTrackSeeds(seed, drifted, 0.05) {
		t.Error("extrema within reach of their own seeds do not track")
	}
}

// TestAnglesTrackSeedsRefusesACountChange: an extremum born or lost between two stations leaves the
// seeds with nothing one-to-one to claim, and the anchor set is then a label for a topology that is no
// longer there. It is refused on the count alone, before any distance is measured.
func TestAnglesTrackSeedsRefusesACountChange(t *testing.T) {
	t.Parallel()
	seed := []float64{0.2, 1.7, 3.4, 5.0}
	if anglesTrackSeeds(seed, []float64{0.2, 1.7, 3.4, 5.0, 5.9}, 1) {
		t.Error("a station carrying a FIFTH extremum tracked four seeds")
	}
	if anglesTrackSeeds(seed, []float64{0.2, 1.7, 3.4}, 1) {
		t.Error("a station that LOST an extremum tracked four seeds")
	}
}

// TestAnglesTrackSeedsRefusesADriftBeyondReach: a track that moved further than reach is not the same
// track, however well the count matches. reach is half the seeds' own minimum spacing, so a drift past
// it is a drift into another lane's half of the gap.
func TestAnglesTrackSeedsRefusesADriftBeyondReach(t *testing.T) {
	t.Parallel()
	seed := []float64{0.2, 1.7, 3.4, 5.0}
	moved := []float64{0.2, 1.7, 3.4, 5.3}
	if !anglesTrackSeeds(seed, moved, 0.35) {
		t.Fatal("the 0.3 rad drift is inside a reach of 0.35 and was refused; the row would not test the bound")
	}
	if anglesTrackSeeds(seed, moved, 0.2) {
		t.Error("a 0.3 rad drift tracked a seed at a reach of 0.2")
	}
}

// TestAnglesTrackSeedsRefusesADoubleClaim is the one-to-one half, and the only refusal a count test and
// a distance test together cannot make: two seeds whose nearest extremum is the SAME one. That is a
// station where two lanes have merged, and pairing branches by it would put both on one arc.
//
// The station here carries two extrema at the count the seeds have, and both of the seeds at 3.4 and
// 5.0 are nearest to 4.2 — so the count matches, each is within reach, and the pairing is still
// impossible.
func TestAnglesTrackSeedsRefusesADoubleClaim(t *testing.T) {
	t.Parallel()
	seed := []float64{3.4, 5.0}
	ex := []float64{4.2, 0.1}
	if n := len(ex); n != len(seed) {
		t.Fatalf("the fixture presents %d extrema against %d seeds; the count branch would refuse first", n, len(seed))
	}
	for _, a := range seed {
		if i := nearestAngleIndex(ex, a); i != 0 {
			t.Fatalf("seed %g claims extremum %d, not the shared one; the row would not test the double claim", a, i)
		}
		if d := stdmath.Abs(shortestTurnDelta(a, ex[0])); d > 1 {
			t.Fatalf("seed %g is %g from the shared extremum, outside the reach the row passes; the drift branch would refuse first", a, d)
		}
	}
	if anglesTrackSeeds(seed, ex, 1) {
		t.Error("two seeds claiming ONE extremum tracked it; the lanes have merged and the pairing is a guess")
	}
}
