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

// assertBranchCountMatchesTheRoots re-reads the certificate from OUTSIDE the reduction: at every station
// the curves must carry exactly as many azimuths as the quartic has certified real roots, and each
// curve's own azimuth must BE one of them.
//
// The second half is not redundant with the production certificate, which reads each curve's LANE
// (torusLaneRoot) rather than evaluating the curve. This row asks TorusSectionArc.azimuthAt — the
// evaluator a consumer actually calls — so a curve whose evaluator drifted off its own lane fails here
// and nowhere else. Review round 1 planted exactly that (+1e-9 inside azimuthAt) and this is what caught
// it.
func assertBranchCountMatchesTheRoots(t *testing.T, ring Torus, q Quadric, curves []Curve3) {
	t.Helper()
	seen := map[int]int{}
	for i := range 97 {
		v := twoPi * float64(i) / 97
		h := torusSecondHarmonicAt(ring, q, v)
		roots := h.azimuths()
		seen[len(roots)]++
		if got := len(torusAzimuthsCarriedAt(h, h.extrema(), roots, curves, v)); got != len(roots) {
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

// TestFourArcsOnOneLaneAreRefused is the plant review round 1 found the certificate blind to (finding 1).
// The section's azimuth census used to be a TALLY: two per covering loop, one per arc, compared with the
// number of roots. Four arcs all anchored on the SAME lane carry the right number of branches, three of
// them duplicates of the first and three certified roots carried by nothing — and a tally passes it. The
// certificate now compares POSITIONS, so it does not.
//
// The row builds the real section first and then re-anchors every arc onto one lane, so what changes is
// only the thing under test: the same four curves, on the same torus, taking one azimuth between them.
func TestFourArcsOnOneLaneAreRefused(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	fat, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 2)
	q := fat.QuadricForm()
	curves, _, ok := torusSkewSection(ring, q, ResolutionForSize(12))
	if !ok || len(curves) != 4 {
		t.Fatalf("the fat rod gave ok=%v with %d curves; this row needs the four-arc section", ok, len(curves))
	}
	if why := torusCurvesAccountForEveryAzimuth(ring, q, curves); why != DeclineNone {
		t.Fatalf("the correct section is reported as %v", why)
	}
	if why := torusCurvesAccountForEveryAzimuth(ring, q, arcsReanchoredOntoOneLane(t, curves)); why != DeclineTorusLaneUnaccounted {
		t.Errorf("four arcs on one lane are reported as %v, want the unaccounted refusal", why)
	}
}

// arcsReanchoredOntoOneLane copies the section with every arc moved onto the first one's lane, so all of
// them trace the same branch.
func arcsReanchoredOntoOneLane(t *testing.T, curves []Curve3) []Curve3 {
	t.Helper()
	first, isArc := curves[0].(TorusSectionArc)
	if !isArc {
		t.Fatalf("curve 0 is %T; this row re-anchors arcs", curves[0])
	}
	out := make([]Curve3, 0, len(curves))
	for _, cv := range curves {
		a := cv.(TorusSectionArc)
		a.UA = first.UA
		out = append(out, a)
	}
	return out
}

// TestAGrazingStationIsNamedATangency splits what review round 1 found sharing one name (finding 3). A
// rod whose wall lies TANGENT to the top of the ring's tube touches it along the ring's own top circle
// and crosses it nowhere. That is a statement about the INPUT — no pairing of branches can carry a double
// root — and it must not reach a user as "the kernel's curves are not the azimuths the stations certify",
// which is a statement about this reduction.
func TestAGrazingStationIsNamedATangency(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	grazing, _ := NewCylinder(math.P3(0, 0, 3.5), math.V3(1, 0, 0), 2)
	_, why, ok := IntersectSurfacesAnalyticDeclining(ring, grazing, ResolutionForSize(12))
	if ok {
		t.Fatal("the grazing rod built; the fixture no longer drives the tangency refusal")
	}
	if why != DeclineTorusTangentStation || !why.IsConditioning() {
		t.Errorf("the grazing rod refused with %v (conditioning=%v), want the tangency", why, why.IsConditioning())
	}
}
