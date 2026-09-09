// SPDX-License-Identifier: GPL-2.0-only

package geom

import stdmath "math"

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
// Every refusal here is NAMED ([SectionDecline]) rather than an anonymous ok=false, because each one is
// a conditioning demotion — the closed form applies to the pair, it just cannot name its own answer at
// these numbers — and the caller records it as a defect. The four are: a station with no azimuth
// dependence at all; extremum tracks that cross or change in number over the turn, which makes "which
// branch pair" a guess; a pair that never folds, which is four independent full-period branches rather
// than a folded pair; and a window whose branches never separate past the stitch resolution.
//
// The fifth certificate is on the RESULT rather than on any one step: the loops must account for every
// azimuth the stations carry, two per covering loop. Windows are skipped along the way — a pair that
// merges at a FLANKING extremum is the complementary arc of a neighbouring lane, which carries it
// itself — and that skip rests on a premise about a neighbour that nothing else checks. Counting the
// roots checks it, and a section loop can no longer go missing quietly.

// torusSkewSection returns the exact intersection of a torus with a quadric that is not invariant about
// the torus axis, on the torus's own chart. An empty result with ok=true is the honest "they do not
// meet"; ok=false always carries the reason.
//
//	curves, why, ok := torusSkewSection(ring, rod.QuadricForm(), geom.ResolutionForBox(box))
func torusSkewSection(t Torus, q Quadric, res Resolution) ([]Curve3, SectionDecline, bool) {
	anchors, ok := torusLaneAnchors(t, q)
	if !ok {
		return nil, DeclineTorusLaneTracks, false
	}
	var out []Curve3
	for _, anchor := range anchors {
		loops, why := torusLaneLoops(t, q, anchor, res)
		if why != DeclineNone {
			return nil, why, false
		}
		out = append(out, loops...)
	}
	if why := torusLoopsAccountForEveryAzimuth(t, q, out); why != DeclineNone {
		return nil, why, false
	}
	return out, DeclineNone, true
}

// torusLaneLoops returns one folded loop per tube-angle window of ONE lane.
func torusLaneLoops(t Torus, q Quadric, anchor float64, res Resolution) ([]Curve3, SectionDecline) {
	readable := true
	spans, folds := periodicRootWindows(func(v float64) float64 {
		l, ok := torusLaneAt(torusSecondHarmonicAt(t, q, v), anchor)
		readable = readable && ok
		return l.discriminant()
	}, torusStationProbes)
	switch {
	case !readable:
		return nil, DeclineTorusLaneStation
	case !folds:
		return nil, DeclineTorusLaneFullTurn
	}
	return torusWindowLoops(t, q, anchor, spans, res)
}

// torusWindowLoops builds the loop of every window this lane OWNS. A window whose branch pair merges at
// a flanking extremum instead of the lane's own belongs to a neighbouring lane and is skipped here; the
// azimuth count on the finished set is what proves that neighbour really carried it.
func torusWindowLoops(t Torus, q Quadric, anchor float64, spans [][2]float64, res Resolution) ([]Curve3, SectionDecline) {
	var out []Curve3
	for _, w := range spans {
		loop := TorusQuadricLoop{Torus: t, Quad: q, V0: w[0], V1: w[1], UA: anchor}
		owns, ok := torusLaneOwnsWindow(loop)
		if !ok {
			return nil, DeclineTorusLaneStation
		}
		if !owns {
			continue
		}
		if !torusWindowConditioning(loop, res) {
			return nil, DeclineTorusLaneSeparation
		}
		out = append(out, loop)
	}
	return out, DeclineNone
}

// torusLaneOwnsWindow reports that the window's branch pair merges at the lane's OWN extremum at both
// folds — the certificate that this lane, and not a neighbour, carries the pair. It is read just inside
// each end, because periodicRootWindows returns the fold on the non-positive side where the pair has
// already merged and every candidate reads the same azimuth. ok=false is an unreadable station.
func torusLaneOwnsWindow(l TorusQuadricLoop) (owns, ok bool) {
	step := (l.V1 - l.V0) / torusWindowProbes
	first, okA := torusLaneOwnsStation(l, l.V0+step)
	last, okB := torusLaneOwnsStation(l, l.V1-step)
	return first && last, okA && okB
}

// torusLaneOwnsStation reads the lane at one tube angle and asks whether its own extremum is the one
// closest to vanishing there.
func torusLaneOwnsStation(l TorusQuadricLoop, v float64) (owns, ok bool) {
	lane, ok := torusLaneAt(torusSecondHarmonicAt(l.Torus, l.Quad, v), l.UA)
	return ok && lane.mergesAtItsCenter(), ok
}

// torusLoopsAccountForEveryAzimuth certifies the finished loop SET against the stations themselves: at
// every probe tube angle, the loops whose window covers it must account for exactly the azimuths that
// station carries, two each. It is what turns "a neighbouring lane carries this window" from a premise
// into a measurement — a dropped loop leaves two azimuths belonging to nothing, and a doubled one two
// too many.
//
// A probe within half a step of a window END is skipped: an end IS a fold, where the two azimuths have
// merged and the station carries one rather than two, so counting there would report a mismatch that is
// the fold's arithmetic rather than a missing loop. Every window the discriminant sampler found has an
// interior probe of its own, which is where a genuinely dropped loop shows up.
func torusLoopsAccountForEveryAzimuth(t Torus, q Quadric, loops []Curve3) SectionDecline {
	step := twoPi / torusStationProbes
	for i := range torusStationProbes {
		v := step * float64(i)
		if nearAWindowEnd(loops, v, step/2) {
			continue
		}
		if len(torusSecondHarmonicAt(t, q, v).azimuths()) != 2*loopsCovering(loops, v) {
			return DeclineTorusLaneUnaccounted
		}
	}
	return DeclineNone
}

// loopsCovering counts the loops whose tube-angle window contains v, folded onto one period so a window
// that straddles the chart's seam counts like any other.
func loopsCovering(loops []Curve3, v float64) int {
	n := 0
	for _, cv := range loops {
		if l, ok := cv.(TorusQuadricLoop); ok && wrapAngle(v-l.V0) < l.V1-l.V0 {
			n++
		}
	}
	return n
}

// nearAWindowEnd reports v sitting within reach of some loop's fold, where the station's two azimuths
// have merged into one.
func nearAWindowEnd(loops []Curve3, v, reach float64) bool {
	for _, cv := range loops {
		l, ok := cv.(TorusQuadricLoop)
		if ok && stdmath.Min(foldedGap(v, l.V0), foldedGap(v, l.V1)) <= reach {
			return true
		}
	}
	return false
}

// foldedGap is the distance between two tube angles the short way round the period.
func foldedGap(a, b float64) float64 { return stdmath.Abs(shortestTurnDelta(a, b)) }

// torusAzimuthAt is the ONE azimuth reader both torus section forms evaluate through. It classifies the
// station and takes exactly one reduction: the one-harmonic arccos where the quadric's form is
// invariant about the torus axis, and the anchor's lane where it is not. A station neither reduction can
// read answers NaN, as [torusHarmonic.root] does for the same reason: an azimuth that is not a root of
// this station is not an answer, and a plausible-looking one is worse than none.
func torusAzimuthAt(t Torus, q Quadric, v, anchor float64, upper bool) float64 {
	st := torusStationAt(t, q, v)
	if st.invariant {
		return st.harmonic().root(upper)
	}
	l, ok := torusLaneAt(st.secondHarmonic(), anchor)
	if !ok {
		return stdmath.NaN()
	}
	return l.root(upper)
}

// torusFoldAzimuth is the MERGED azimuth of a station's branch pair: the one azimuth both branches
// share at a fold. It makes the same classification torusAzimuthAt does and then reads the pair's
// LABEL instead of one of its roots — the lane's own extremum, or the one-harmonic form's fold phase.
//
// It exists because a fold's window end is a bisected root of the discriminant: the discriminant there
// is zero only to rounding, so root() still separates the pair by half a square root of that rounding
// (~1e-8 in azimuth) instead of returning the merged value. Only a caller that KNOWS it is at a fold
// can say so, which is why this is a second entry rather than a branch inside torusAzimuthAt.
func torusFoldAzimuth(t Torus, q Quadric, v, anchor float64) float64 {
	st := torusStationAt(t, q, v)
	if st.invariant {
		return st.harmonic().foldRoot()
	}
	l, ok := torusLaneAt(st.secondHarmonic(), anchor)
	if !ok {
		return stdmath.NaN()
	}
	return l.center
}
