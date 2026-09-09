// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// The WRAPPING half of the torus section (Oblikovati/Oblikovati#3515), and the classification that tells
// it from a folded one.
//
// A track — one of a lane's two azimuths — that never merges with a neighbour has no fold to bound a
// window with, so it is an independent full-period branch. The one-harmonic family has always carried
// that shape as two [TorusSectionArc]s; the general second-harmonic family declined it by name, which
// made an ordinary part — a rod through a ring's cross-section, thicker than the ring's tube — a
// refusal. These rows drive the shape on the general side and pin the classification the two halves
// share.

// TestTheFullTurnTopologyIsCarriedAsFourBranches: a rod ACROSS the ring whose radius EXCEEDS the tube's
// swallows the tube's own flank at every tube angle, so the branch pair never folds — four independent
// full-period branches rather than a folded pair (Oblikovati/Oblikovati#3515). Nothing about the count
// is written down: the station's quartic certifies four real roots, the extremum tracks name four
// lanes, and each lane contributes the one arc its upper track traces. This row reads the count back off
// the RESULT and then checks it against the roots, station by station, on the curves themselves.
func TestTheFullTurnTopologyIsCarriedAsFourBranches(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	fat, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 2)
	curves, why, ok := IntersectSurfacesAnalyticDeclining(ring, fat, ResolutionForSize(12))
	if !ok || why != DeclineNone {
		t.Fatalf("ok=%v why=%v, want the four full-period branches", ok, why)
	}
	assertEveryCurveIsAFullTurnArc(t, curves)
	assertBranchCountMatchesTheRoots(t, ring, fat.QuadricForm(), curves)
}

// assertEveryCurveIsAFullTurnArc requires the section to be full-period arcs and nothing else: a lane
// that never folds has no window, so a folded loop here would be a fold the sampler invented.
func assertEveryCurveIsAFullTurnArc(t *testing.T, curves []Curve3) {
	t.Helper()
	for i, cv := range curves {
		a, isArc := cv.(TorusSectionArc)
		if !isArc {
			t.Fatalf("curve %d is %T; a lane that never folds carries a full-period arc", i, cv)
		}
		if a.V0 != 0 || a.V1 != twoPi {
			t.Errorf("curve %d spans [%g, %g]; a wrap runs the tube's whole turn", i, a.V0, a.V1)
		}
	}
}

// assertBranchCountMatchesTheRoots is the certificate itself, re-read from outside the reduction: at
// every station the curves must carry exactly as many azimuths as the quartic has certified real roots,
// and each curve's own azimuth must BE one of them. A count that agreed while the arcs sat on the wrong
// azimuths would pass the first half and fail the second.
func assertBranchCountMatchesTheRoots(t *testing.T, ring Torus, q Quadric, curves []Curve3) {
	t.Helper()
	seen := map[int]int{}
	for i := range 97 {
		v := twoPi * float64(i) / 97
		roots := torusSecondHarmonicAt(ring, q, v).azimuths()
		seen[len(roots)]++
		if got := azimuthsCarriedAt(curves, v); got != len(roots) {
			t.Fatalf("v=%g: the curves carry %d azimuths, the station certifies %d", v, got, len(roots))
		}
		for j, cv := range curves {
			assertAzimuthIsACertifiedRoot(t, v, j, cv.(TorusSectionArc).azimuthAt(v), roots)
		}
	}
	if seen[4] != 97 {
		t.Fatalf("root counts over the turn: %v; the fat rod carries four at every station", seen)
	}
}

// assertAzimuthIsACertifiedRoot requires one arc's azimuth to be one of the station's certified roots.
func assertAzimuthIsACertifiedRoot(t *testing.T, v float64, arc int, u float64, roots []float64) {
	t.Helper()
	for _, r := range roots {
		if stdmath.Abs(shortestTurnDelta(u, r)) <= 1e-12 { // tol:angular — the arc and the root solver on one root
			return
		}
	}
	t.Fatalf("v=%g: arc %d takes azimuth %.17g, which is none of the station's roots %v", v, arc, u, roots)
}

// TestFoldsAlternateOnEverySignPattern is the proof behind a DELETED branch. periodicRootWindows used
// to refuse a discriminant whose rises and falls came back in unequal numbers, calling it numerical
// noise, and that refusal made "not a window" mean two different things at every call site — which is
// what hid the wrap the fat-rod section needs (Oblikovati/Oblikovati#3515).
//
// It was unreachable. The sign pattern is taken from a FIXED sample array walked as a cycle, so the
// up-transitions and the down-transitions bound the same arcs and are equal in number whatever the
// samples are. This row drives every one of the 4096 twelve-probe patterns and requires the windows to
// come back well formed: one per rise, each one a non-empty span, and none of them the whole period.
func TestFoldsAlternateOnEverySignPattern(t *testing.T) {
	t.Parallel()
	const probes = 12
	for pattern := range 1 << probes {
		spans, folded := periodicRootWindows(func(u float64) float64 {
			return float64(pattern>>probeIndexAt(u, probes)&1)*2 - 1
		}, probes)
		assertWindowsAreWellFormed(t, pattern, spans, folded)
	}
}

// probeIndexAt is which of the probes' equal arcs an angle falls in, so a bit pattern reads as a
// piecewise-constant discriminant the fold bisection can be driven over.
func probeIndexAt(u float64, probes int) int {
	return int(wrapAngle(u) * float64(probes) / twoPi)
}

// assertWindowsAreWellFormed requires one sign pattern's windows to match the shape the classification
// promises: a wrap carries no spans, and every span of a folded section is a proper sub-arc.
func assertWindowsAreWellFormed(t *testing.T, pattern int, spans [][2]float64, folded bool) {
	t.Helper()
	if !folded && len(spans) != 0 {
		t.Fatalf("pattern %012b: a wrap came back with %d spans", pattern, len(spans))
	}
	for _, w := range spans {
		if w[1] <= w[0] || w[1]-w[0] >= twoPi {
			t.Fatalf("pattern %012b: span [%g, %g] is not a proper sub-arc of the period", pattern, w[0], w[1])
		}
	}
}

// TestAWrapIsNotAnEmptySection pins the distinction the folded flag carries: positive at every station
// is the WRAP, and non-positive at every station is the empty section. Both have no fold, and reading
// them as one thing is what made a fat rod's four branches look like a refusal.
func TestAWrapIsNotAnEmptySection(t *testing.T) {
	t.Parallel()
	if _, folded := periodicRootWindows(func(float64) float64 { return 1 }, 8); folded {
		t.Error("an everywhere-positive discriminant was classified as folded, want the wrap")
	}
	spans, folded := periodicRootWindows(func(float64) float64 { return -1 }, 8)
	if !folded || len(spans) != 0 {
		t.Errorf("an everywhere-negative discriminant gave folded=%v spans=%d, want the empty section", folded, len(spans))
	}
}
