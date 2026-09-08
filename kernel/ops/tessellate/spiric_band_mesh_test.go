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

// tubeBandColumns scales the loft column count with the band's widest travel and the angle tolerance.
func TestTubeBandColumns(t *testing.T) {
	t.Parallel()
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	pl, _ := geom.NewPlane(math.P3(0, 2, 0), math.V3(0, 1, 0))
	phi, m, k, c := geom.TorusSectionCoeffs(tor, pl)
	plus := geom.SpiricArc{Torus: tor, Phi: phi, M: m, K: k, C: c, Branch: +1, V0: 0, V1: 2 * stdmath.Pi}
	minus := geom.SpiricArc{Torus: tor, Phi: phi, M: m, K: k, C: c, Branch: -1, V0: 0, V1: 2 * stdmath.Pi}
	loU, hiU := plus.UAt, minus.UAt
	vs := make([]float64, 32)
	for i := range vs {
		vs[i] = 2 * stdmath.Pi * float64(i) / 32
	}
	if n := tubeBandColumns(loU, hiU, vs, 1, DefaultQuality()); n < 3 {
		t.Errorf("tubeBandColumns = %d, want ≥3 for a wide band at default angle tolerance", n)
	}
	// A coarse angle tolerance needs fewer columns than a fine one.
	coarse := tubeBandColumns(loU, hiU, vs, 1, Quality{AngleTolerance: 1})
	fine := tubeBandColumns(loU, hiU, vs, 1, Quality{AngleTolerance: 0.05})
	if coarse >= fine {
		t.Errorf("tubeBandColumns coarse=%d should be < fine=%d", coarse, fine)
	}
}

// TestBandWidthAtIsOnePeriodEitherWay: the two directions across a band partition the tube's period, at
// every station. That is the invariant the whole side-choice rests on, and it is what a raw azimuth
// difference does NOT give — a boundary's samples carry an arbitrary whole turn, so an unfolded
// difference can be a period out at one station and not the next, and the loft then covers the tube
// more than once.
func TestBandWidthAtIsOnePeriodEitherWay(t *testing.T) {
	t.Parallel()
	// Two boundaries whose raw azimuths are deliberately on different branches.
	loU := func(v float64) float64 { return 0.4*stdmath.Sin(v) + 12*stdmath.Pi }
	hiU := func(v float64) float64 { return 2.2 + 0.3*stdmath.Cos(v) - 8*stdmath.Pi }
	for _, v := range []float64{0, 1, 2, 3, 4, 5, 6} {
		fwd := stdmath.Abs(bandWidthAt(loU, hiU, v, 1))
		back := stdmath.Abs(bandWidthAt(loU, hiU, v, -1))
		if stdmath.Abs(fwd+back-2*stdmath.Pi) > 1e-9 { // tol:numeric — the fold's own rounding
			t.Errorf("at v=%g the two directions span %.6f + %.6f, want one period", v, fwd, back)
		}
		if fwd < 0 || fwd > 2*stdmath.Pi {
			t.Errorf("at v=%g the forward travel is %.6f, outside one period", v, fwd)
		}
	}
}

// TestEdgeWrapsTheTube is the gate that replaced the spiric-pair test: a band's boundary goes the WHOLE
// way round the tube, and an arc fillet's quarter-tube run-out section does not. The old guard
// (oppositeRootsOfOneSection) asked whether two spiric arcs were the two roots of one plane's section,
// which answered the same question for spiric edges only; this one answers it for any curve, which is
// what lets the loft take a torus∩quadric section too (ADR-0061 stage 5).
func TestEdgeWrapsTheTube(t *testing.T) {
	t.Parallel()
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	whole := make([]math.Point3, 0, 65)
	quarter := make([]math.Point3, 0, 17)
	for i := 0; i <= 64; i++ {
		v := 2 * stdmath.Pi * float64(i) / 64
		whole = append(whole, tor.PointAt(0.3, v))
		if i <= 16 {
			quarter = append(quarter, tor.PointAt(0.3, v))
		}
	}
	if !edgeWrapsTheTube(tor, whole) {
		t.Error("a boundary going the whole way round the tube was not recognised as a band's")
	}
	if edgeWrapsTheTube(tor, quarter) {
		t.Error("a quarter-tube section was taken for a band boundary; lofting between two of those sweeps the whole tube")
	}
	// A chain that runs out and BACK covers the same stations twice and turns by nothing.
	outAndBack := append(append([]math.Point3(nil), quarter...), quarter[len(quarter)-2]) //nolint:gocritic // deliberate
	for i := len(quarter) - 2; i >= 0; i-- {
		outAndBack = append(outAndBack, quarter[i])
	}
	if edgeWrapsTheTube(tor, outAndBack) {
		t.Error("a chain that runs out and back was taken for a band boundary; its NET turn is zero")
	}
}

// TestAzimuthInterpolatorFollowsTheSeam: a boundary's azimuth is read from its own samples, and those
// samples cross the azimuth seam like any curve. The interpolator must follow it as one curve rather
// than average across the jump, or the loft's interior rows land on the far side of the torus.
func TestAzimuthInterpolatorFollowsTheSeam(t *testing.T) {
	t.Parallel()
	// A boundary at a nearly constant azimuth just below the seam, straddling it.
	vs := []float64{0, 1, 2, 3, 4, 5}
	us := []float64{6.2, 6.28, 0.02, 0.08, 6.27, 6.2}
	at := azimuthInterpolator(vs, us)
	for _, v := range []float64{0.5, 1.5, 2.5, 3.5, 4.5} {
		u := wrapToPeriod(at(v))
		if u > 0.3 && u < 2*stdmath.Pi-0.3 {
			t.Errorf("at v=%g the interpolated azimuth is %.4f, far from the boundary's own band near the seam", v, u)
		}
	}
}
