// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// TestValidateRunoutFansRejectsSelfIntersecting is the honest-reject gate — the n-valent analogue
// of the #1800 over-radius reject. A far edge that never reaches the fillet tube (radius grossly
// oversized relative to the geometry) fails solveRunoutSpread's crossing certificate; without
// Task 7's pre-pass, buildSpreadMaps would silently skip the fan and the rebuild would ship an open
// shell instead of erroring. This proves the certificate still fires (Task 4/5 wired it), and the
// next test proves validateRunoutFans surfaces it as a hard reject.
func TestValidateRunoutFansRejectsSelfIntersecting(t *testing.T) {
	fan := endCornerFan{
		radius: 100, center: math.P3(0, 0, 0), axis: math.V3(1, 0, 0), apex: math.P3(0, 0, 0),
		ta: math.P3(0, 2, 0), tb: math.P3(0, -2, 0),
		fan:      []fanFace{{face: 1, normal: math.V3(0, 1, 0), exitEdge: 9}, {face: 2, normal: math.V3(0, -1, 0), entryEdge: 9}},
		farEdges: []fanEdge{{edge: 9, from: math.P3(0, 0, 0), to: math.P3(0, 1, 0), leftFace: 1, rightFace: 2}},
	}
	if _, err := solveRunoutSpread(fan); err == nil {
		t.Fatal("expected honest-reject on an over-radius runout")
	}
}

// TestValidateRunoutFansAcceptsV3 is the no-blanket-reject guard: V3's real valence-5 fan is
// genuinely valid (it closes to a solid, TestV3FilletClosesToSolid), so validateRunoutFans must let
// it through — proving the pre-pass rejects only genuinely invalid fans, not every fan.
func TestValidateRunoutFansAcceptsV3(t *testing.T) {
	b := importCorpusSolid(t, "simple/V3")
	fils := solvedFilsForCase(t, b, "simple/V3")
	if err := validateRunoutFans(fils); err != nil {
		t.Fatalf("validateRunoutFans rejected V3's valid runout: %v", err)
	}
}
