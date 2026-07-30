// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// singleHostObstacleCase is one corpus body whose filleted edge carries exactly ONE mid-span obstacle,
// located by the very pick (edge midpoint + radius) the parity grid scores the case with (corpus.json).
type singleHostObstacleCase struct {
	name   string
	step   string
	mid    math.Point3
	radius float64
}

// singleHostObstacleCorpus is EVERY corpus case that reaches the surf-rst canal tier: the five
// single-host obstacle bodies. Only U3 carries a fingerprint pin, so before
// TestObstacleCanalEndStationsWeldToTheWingsBitForBit the other four had nothing gating their end-station
// weld at all — a no-op pinObstacleCanalEnds moved U3's hash and NOTHING else.
var singleHostObstacleCorpus = []singleHostObstacleCase{
	{"R9", "simple/R9.step", math.P3(0, -10, 10), 3},
	{"S3", "simple/S3.step", math.P3(0, -15, 0), 8},
	{"T6", "simple/T6.step", math.P3(0, -13, 0), 6},
	{"U3", "simple/U3.step", math.P3(0, 0, 15), 5},
	{"X3", "simple/X3.step", math.P3(50, 0, 100), 25},
}

// TestObstacleCanalEndStationsWeldToTheWingsBitForBit is the ops-level gate on pinObstacleCanalEnds, on
// ALL FIVE canal cases rather than on U3's fingerprint hash alone: the patch's v=0/v=1 boundary must weld
// to the wing faces BY VALUE, so each end station's centre and wall foot must be the wing section's OWN
// value to the last bit — not merely inside the model weld, which is what the closed form lands at on its
// own. It drives the REAL pipeline (resolveFilletPicks → computeCorners → computeFillets →
// detectObstacle → buildObstacleFeature) on the real imported corpus bodies, so it also pins that
// buildObstacleCanal still calls the pin at all.
func TestObstacleCanalEndStationsWeldToTheWingsBitForBit(t *testing.T) {
	for _, c := range singleHostObstacleCorpus {
		t.Run(c.name, func(t *testing.T) {
			body := importCorpusBody(t, c.step)
			_, of, og, _ := obstacleFeatureFor(t, body, c.name, c.mid, c.radius)
			if of.Canal == nil {
				t.Fatalf("%s: no surf-rst payload — this case must reach the canal tier", c.name)
			}
			assertCanalEndsPinnedToWings(t, of, og)
		})
	}
}

// assertCanalEndsPinnedToWings checks the six end-station values against the wing sections' own, with ==
// (bit-for-bit), and reports the metric gap when one differs so a near-miss reads as the rounding-level
// drift it is rather than as a gross error.
//
// The two RIM ends are CONSTRUCTION invariants, not evidence: dipRimSamples already writes the exact
// nodes into RimArcPts' endpoints, so those two can only fail if that changes. The four CENTRE and WALL
// ends are the pin's real work — the closed form lands there only to rounding.
func assertCanalEndsPinnedToWings(t *testing.T, of *ObstacleFeature, og obstacleGeom) {
	t.Helper()
	c, last := of.Canal, len(of.Canal.Centres)-1
	for _, w := range []struct {
		what      string
		got, want math.Point3
	}{
		{"centre[0]", c.Centres[0], og.startCen},
		{"centre[last]", c.Centres[last], og.endCen},
		{"wall foot[0]", c.FeetWall[0], og.wallA},
		{"wall foot[last]", c.FeetWall[last], og.wallD},
		{"rim foot[0] (construction invariant)", c.FeetRim[0], of.Nodes[0]},
		{"rim foot[last] (construction invariant)", c.FeetRim[last], of.Nodes[1]},
	} {
		if w.got != w.want {
			t.Errorf("%s = %v, want the wing section's own %v BIT-FOR-BIT (off by %.3e) — the patch's end boundary no longer welds to the wing by value",
				w.what, w.got, w.want, w.got.DistanceTo(w.want))
		}
	}
}
