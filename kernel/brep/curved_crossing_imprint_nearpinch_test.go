// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// Near-pinch crossings through the general pipeline (#1781/#1818, ADR-0061 stage 4). Two cylinders whose
// radii differ by a hair meet in two lens loops separated by a neck narrow relative to the imprint chord.
// This file used to test the GATE that classified such a pair — typicalLoopChord, interLoopMinDistance,
// loopGapChordRatio, nearPinchLoops, unequalRadiusCrossing — because the crossing cut and join declined the
// band to a faceted fallback while the intersect took a per-loop wall trim. There is no fallback and no
// second path to route to, so the gate and its tests are deleted and what remains is the corpus: the band
// must simply build, at both ends of it.

// TestCrossingCylinderIntersectRecoveredBand: a crossing just ABOVE the retired gate (Δr = 2e-4) builds the
// exact three-face intersect — the rod band between the two imprint loops plus the fat wall's two lens caps.
func TestCrossingCylinderIntersectRecoveredBand(t *testing.T) {
	t.Parallel()
	const r = 3.0
	thin, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), r, 12)
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), r+2e-4, 12)
	body, err := Boolean(Intersection, thin, fat)
	if err != nil {
		t.Fatalf("recovered-band crossing intersect: %v", err)
	}
	assertWatertight(t, body)
	if n := len(body.Faces()); n != 3 {
		t.Errorf("recovered-band intersect has %d faces, want 3 (rod band + two lens caps)", n)
	}
}

// TestCrossingCylinderIntersectNearPinchPerLoop: a crossing DEEP in the band (Δr = 5e-5, below the retired
// gate, where the neck is a fraction of the chord) builds the same exact three-face intersect. The fat wall
// is trimmed one lens loop at a time, so the narrow neck never has to be resolved as a single trim (#1818).
func TestCrossingCylinderIntersectNearPinchPerLoop(t *testing.T) {
	t.Parallel()
	const r = 3.0
	thin, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), r, 12)
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), r+5e-5, 12)
	body, err := Boolean(Intersection, thin, fat)
	if err != nil {
		t.Fatalf("near-pinch crossing intersect: %v", err)
	}
	assertWatertight(t, body)
	if n := len(body.Faces()); n != 3 {
		t.Errorf("near-pinch intersect has %d faces, want 3 (rod band + two lens caps)", n)
	}
}
