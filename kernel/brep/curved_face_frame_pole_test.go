// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// anchorPoleEnds states the chart's rule at a singular point: the azimuth of a pole sample is not the
// one continuity would give it, it is the one its NEIGHBOUR has, so the polyline reaches the pole on
// the neighbour's own meridian and stops there (ADR-0061).
func TestAnchorPoleEndsTakesTheNeighboursMeridian(t *testing.T) {
	t.Parallel()
	const poleV = -1.5707963267948966
	for _, row := range []struct {
		name         string
		a, b         math.Point2
		aPole, bPole bool
		wantA, wantB math.Point2
	}{
		{
			name: "an end at the pole takes the other end's u",
			a:    math.P2(1.5, -1.55), b: math.P2(4.5, poleV), bPole: true,
			wantA: math.P2(1.5, -1.55), wantB: math.P2(1.5, poleV),
		},
		{
			name: "a start at the pole takes the other end's u",
			a:    math.P2(4.5, poleV), b: math.P2(1.5, -1.55), aPole: true,
			wantA: math.P2(1.5, poleV), wantB: math.P2(1.5, -1.55),
		},
		{
			name: "a regular segment is untouched",
			a:    math.P2(1.5, -1.55), b: math.P2(1.6, -1.4),
			wantA: math.P2(1.5, -1.55), wantB: math.P2(1.6, -1.4),
		},
		{
			name: "both ends at the pole keep what they have: neither can anchor the other",
			a:    math.P2(1.5, poleV), b: math.P2(4.5, poleV), aPole: true, bPole: true,
			wantA: math.P2(1.5, poleV), wantB: math.P2(4.5, poleV),
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			gotA, gotB := anchorPoleEnds(row.a, row.b, row.aPole, row.bPole)
			if gotA != row.wantA || gotB != row.wantB {
				t.Errorf("anchorPoleEnds = %v,%v; want %v,%v", gotA, gotB, row.wantA, row.wantB)
			}
		})
	}
}
