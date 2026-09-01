// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// finerRowVs returns whichever row's tube stations are denser (the interior loft reuses them).
func TestFinerRowVs(t *testing.T) {
	t.Parallel()
	a := bandRow{ang: []float64{0, 1, 2}}
	b := bandRow{ang: []float64{0, 1}}
	if got := finerRowVs(a, b); len(got) != 3 {
		t.Errorf("finerRowVs picked the coarser row: len %d, want 3", len(got))
	}
	if got := finerRowVs(b, a); len(got) != 3 {
		t.Errorf("finerRowVs(b,a) picked the coarser row: len %d, want 3", len(got))
	}
}

// spiricBandColumns scales the loft column count with the band's widest u-span and the angle tolerance.
func TestSpiricBandColumns(t *testing.T) {
	t.Parallel()
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	pl, _ := geom.NewPlane(math.P3(0, 2, 0), math.V3(0, 1, 0))
	phi, m, k, c := geom.TorusSectionCoeffs(tor, pl)
	plus := geom.SpiricArc{Torus: tor, Phi: phi, M: m, K: k, C: c, Branch: +1, V0: 0, V1: 2 * stdmath.Pi}
	minus := geom.SpiricArc{Torus: tor, Phi: phi, M: m, K: k, C: c, Branch: -1, V0: 0, V1: 2 * stdmath.Pi}
	if n := spiricBandColumns(plus, minus, DefaultQuality()); n < 3 {
		t.Errorf("spiricBandColumns = %d, want ≥3 for a wide band at default angle tolerance", n)
	}
	// A coarse angle tolerance needs fewer columns than a fine one.
	coarse := spiricBandColumns(plus, minus, Quality{AngleTolerance: 1})
	fine := spiricBandColumns(plus, minus, Quality{AngleTolerance: 0.05})
	if coarse >= fine {
		t.Errorf("spiricBandColumns coarse=%d should be < fine=%d", coarse, fine)
	}
}

// oppositeRootsOfOneSection must accept the two arccos roots of one plane's section and reject two
// sections of DIFFERENT planes — the arc band's two run-out terminations, which are not an oval pair.
func TestOppositeRootsOfOneSection(t *testing.T) {
	t.Parallel()
	tor, _ := geom.NewTorusWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 5, 2)
	plus := geom.SpiricArc{Torus: tor, Phi: 1, M: 1, K: -3, C: 0, Branch: +1, V0: 0, V1: 2 * stdmath.Pi}
	minus := plus
	minus.Branch = -1
	if !oppositeRootsOfOneSection(plus, minus) {
		t.Errorf("oppositeRootsOfOneSection rejected the two roots of one section")
	}
	if oppositeRootsOfOneSection(plus, plus) {
		t.Errorf("oppositeRootsOfOneSection accepted two copies of the SAME root")
	}
	other := minus
	other.K = 1 // a different plane's section
	if oppositeRootsOfOneSection(plus, other) {
		t.Errorf("oppositeRootsOfOneSection accepted sections of two different planes")
	}
}
