// SPDX-License-Identifier: GPL-2.0-only

package geom

// The GENERAL half of the torus∩quadric bucket (ADR-0061 stage 5, third slice): a quadric whose
// quadratic form is NOT invariant about the torus axis — a rod driven across a ring, a tilted drill, an
// off-axis cone. Its azimuth dependence is the second harmonic torus_quadric_harmonic2.go derives, and
// its branch pairing the lanes torus_quadric_lane.go names.
//
// The TOPOLOGY question is the one the ruled and one-harmonic buckets already ask, and periodicRootWindows
// answers it here too — once per lane. Each lane's discriminant is positive over the tube angles where
// that pair of azimuths exists and crosses zero at its folds, so a lane's windows are folded loops of
// exactly the same shape the one-harmonic form builds, on the same [TorusQuadricLoop].
//
// Two configurations decline by name rather than being guessed at. A lane whose discriminant never
// falls — the pair exists at EVERY tube angle — is the full-turn topology, which for four azimuths per
// station is four separate branches rather than a folded pair, and this reduction does not carry it.
// And extremum tracks that cross, merge or change in number over the turn make "which lane" a guess;
// both demote to the general marcher (IntersectSurfaceSurface), which is the pipeline's own fallback
// for a pair no closed form names.

// torusSkewSection returns the exact intersection of a torus with a quadric that is not invariant about
// the torus axis, on the torus's own chart. An empty result with ok=true is the honest "they do not
// meet"; ok=false is the named decline the file comment describes.
//
//	curves, ok := torusSkewSection(ring, rod.QuadricForm(), geom.ResolutionForBox(box))
func torusSkewSection(t Torus, q Quadric, res Resolution) ([]Curve3, bool) {
	anchors, ok := torusLaneAnchors(t, q)
	if !ok {
		return nil, false
	}
	var out []Curve3
	for _, anchor := range anchors {
		loops, ok := torusLaneLoops(t, q, anchor, res)
		if !ok {
			return nil, false
		}
		out = append(out, loops...)
	}
	return out, true
}

// torusLaneLoops returns one folded loop per tube-angle window of ONE lane. A window whose branch pair
// merges at a FLANKING extremum instead of the lane's own is skipped: it is the complementary arc of a
// neighbouring lane, which carries that pair itself, and taking both would double the section.
func torusLaneLoops(t Torus, q Quadric, anchor float64, res Resolution) ([]Curve3, bool) {
	spans, ok := periodicRootWindows(func(v float64) float64 {
		return torusLaneAt(torusSecondHarmonicAt(t, q, v), anchor).discriminant()
	}, torusStationProbes)
	if !ok {
		return nil, false // the pair exists at every station: the full-turn topology, not this form's
	}
	var out []Curve3
	for _, w := range spans {
		loop := TorusQuadricLoop{Torus: t, Quad: q, V0: w[0], V1: w[1], UA: anchor}
		if !torusLaneOwnsWindow(loop) {
			continue
		}
		if !torusWindowConditioning(loop, res) {
			return nil, false
		}
		out = append(out, loop)
	}
	return out, true
}

// torusLaneOwnsWindow reports that the window's branch pair merges at the lane's OWN extremum at both
// folds — the certificate that this lane, and not a neighbour, carries the pair. It is read just inside
// each end, because periodicRootWindows returns the fold on the non-positive side where the pair has
// already merged and every candidate reads the same azimuth.
func torusLaneOwnsWindow(l TorusQuadricLoop) bool {
	step := (l.V1 - l.V0) / torusWindowProbes
	return torusLaneOwnsStation(l, l.V0+step) && torusLaneOwnsStation(l, l.V1-step)
}

// torusLaneOwnsStation reads the lane at one tube angle and asks whether its own extremum is the one
// closest to vanishing there.
func torusLaneOwnsStation(l TorusQuadricLoop, v float64) bool {
	return torusLaneAt(torusSecondHarmonicAt(l.Torus, l.Quad, v), l.UA).mergesAtItsCenter()
}

// torusAzimuthAt is the ONE azimuth reader both torus section forms evaluate through. It classifies the
// station and takes exactly one reduction: the one-harmonic arccos where the quadric's form is
// invariant about the torus axis, and the anchor's lane where it is not.
func torusAzimuthAt(t Torus, q Quadric, v, anchor float64, upper bool) float64 {
	st := torusStationAt(t, q, v)
	if st.invariant {
		return st.harmonic().root(upper)
	}
	return torusLaneAt(st.secondHarmonic(), anchor).root(upper)
}
