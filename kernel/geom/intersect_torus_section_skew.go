// SPDX-License-Identifier: GPL-2.0-only

package geom

import stdmath "math"

// The GENERAL half of the torus bucket (ADR-0061 stage 5, third slice): a form whose constraint on the
// tube circle is NOT invariant about the torus axis — a rod driven across a ring, a tilted drill, an
// off-axis cone, and (ADR-0066) very nearly every second TORUS. Its azimuth dependence is the second harmonic torus_section_harmonic2.go derives, and
// its branch pairing the lanes torus_section_lane.go names.
//
// The TOPOLOGY question is the one the ruled and one-harmonic buckets already ask, and it is asked here
// per LANE and per TRACK. A lane is one extremum with the root on each side of it; each of those two
// roots is a TRACK, the arc of azimuths between two neighbouring extrema. The lane's pair FOLDS where
// its two tracks merge, and periodicRootWindows returns the tube-angle windows that bounds — each one a
// loop of exactly the shape the one-harmonic form builds, on the same [TorusSectionLoop]. A track that
// never merges with a neighbour folds nowhere, and it is a branch running the tube's whole turn: a
// [TorusSectionArc] over [0, 2π), exactly as the one-harmonic wrap builds (Oblikovati/Oblikovati#3515).
//
// Consecutive lanes SHARE the track between them, so each lane reads its UPPER track only. Reading every
// lane's upper track once therefore names every azimuth the station carries exactly once, however many
// there are. The two questions are independent: a lane can contribute an arc AND a folded loop, which is
// what a rod that reaches the ring over part of its turn and swallows it over the rest looks like. A rod
// FATTER than the tube it crosses is the whole-turn end of that — four lanes, four tracks, four arcs,
// no folds anywhere — and it used to be a named decline.
//
// Nothing counts branches. The count is whatever the station polynomial's certified real roots and the
// extremum tracks that label them turn out to be, and the azimuth census below is what proves the two
// agree at every station rather than at the one the construction was written for. The one shape this
// does NOT reach is a lane whose upper track wraps while the census cannot be balanced — a root count
// that changes across the turn in a way the loops and arcs do not add up to. That refuses by name; it
// is not built wrongly.
//
// Every refusal here is NAMED ([SectionDecline]) rather than an anonymous ok=false, because each one is
// a conditioning demotion — the closed form applies to the pair, it just cannot name its own answer at
// these numbers — and the caller records it as a defect. The three are: a station with no azimuth
// dependence at all; extremum tracks that cross or change in number over the turn, which makes "which
// branch pair" a guess; and branches that never separate past the stitch resolution. A branch pair that
// never folds used to be a fourth. It is not a refusal any more — it is the wrap, and it is built —
// and the decline that named it is deleted rather than left standing over a case that no longer
// reaches it (Oblikovati/Oblikovati#3515).
//
// The FOURTH certificate is on the RESULT rather than on any one step, and it is the one that makes the
// branch count a RUNTIME certificate instead of a construction: at every probe station the finished
// curves must account for exactly the azimuths that station's quartic certifies there — two for a
// folded loop covering it, one for a full-period arc. Windows are skipped along the way — a pair that
// merges at a FLANKING extremum is the complementary arc of a neighbouring lane, which carries it
// itself — and that skip rests on a premise about a neighbour that nothing else checks. Counting the
// roots checks it, and a section curve can no longer go missing, or be doubled, quietly.

// torusSkewSection returns the exact intersection of a torus with a quadric that is not invariant about
// the torus axis, on the torus's own chart. An empty result with ok=true is the honest "they do not
// meet"; ok=false always carries the reason.
//
//	curves, why, ok := torusSkewSection(ring, rod.QuadricForm(), geom.ResolutionForBox(box))
func torusSkewSection(t Torus, co TorusCoForm, res Resolution) ([]Curve3, SectionDecline, bool) {
	anchors, ok := torusLaneAnchors(t, co)
	if !ok {
		return nil, DeclineTorusLaneTracks, false
	}
	var out []Curve3
	for _, anchor := range anchors {
		curves, why := torusLaneCurves(t, co, anchor, res)
		if why != DeclineNone {
			return nil, why, false
		}
		out = append(out, curves...)
	}
	if why := torusCurvesAccountForEveryAzimuth(t, co, out); why != DeclineNone {
		return nil, why, false
	}
	return out, DeclineNone, true
}

// torusLaneCurves returns ONE lane's share of the section: the full-period arc its upper track traces
// when that track never merges with a neighbour, and a folded loop for each tube-angle window the
// lane's PAIR owns. The two are independent — a lane's upper track can run the whole turn while its
// lower one folds — so both are asked, and either may come back empty.
func torusLaneCurves(t Torus, co TorusCoForm, anchor float64, res Resolution) ([]Curve3, SectionDecline) {
	arcs, why := torusLaneWrapArc(t, co, anchor, res)
	if why != DeclineNone {
		return nil, why
	}
	spans, readable := torusLaneWindows(t, co, anchor)
	if !readable {
		return nil, DeclineTorusLaneStation
	}
	loops, why := torusWindowLoops(t, co, anchor, spans, res)
	if why != DeclineNone {
		return nil, why
	}
	return append(arcs, loops...), DeclineNone
}

// torusLaneWindows are the tube-angle spans over which the lane's branch PAIR exists, bounded by the
// folds where the two merge. A pair that exists at every station has no fold and so no window, which
// periodicRootWindows reports as an empty list here exactly as it reports a pair that exists nowhere:
// either way this lane builds no loop, and its upper track is read separately.
func torusLaneWindows(t Torus, co TorusCoForm, anchor float64) (spans [][2]float64, readable bool) {
	readable = true
	spans, _ = periodicRootWindows(func(v float64) float64 {
		l, ok := torusLaneAt(torusSecondHarmonicAt(t, co, v), anchor)
		readable = readable && ok
		return l.discriminant()
	}, torusStationProbes)
	return spans, readable
}

// torusLaneWrapArc is the ONE full-period arc a lane contributes when its upper track never merges with
// a neighbour, and nothing at all when that track folds — then the loop the fold bounds is built by
// whichever lane's own extremum it merges onto. The census on the finished set is what proves that
// division rather than assuming it.
func torusLaneWrapArc(t Torus, co TorusCoForm, anchor float64, res Resolution) ([]Curve3, SectionDecline) {
	wraps, clearance := torusUpperTrackSweep(t, co, anchor)
	switch {
	case !wraps:
		return nil, DeclineNone
	case clearance <= res.Stitch():
		return nil, DeclineTorusLaneSeparation
	}
	return []Curve3{TorusSectionArc{Torus: t, Co: co, Upper: true, UA: anchor, V0: 0, V1: twoPi}}, DeclineNone
}

// torusUpperTrackSweep walks the tube's whole turn once and answers both questions an arc rests on:
// whether the lane's upper track exists at EVERY station, and how close it comes to any other azimuth
// the station carries. It stops at the first station the track is missing from, because there is no arc
// to certify past that point.
func torusUpperTrackSweep(t Torus, co TorusCoForm, anchor float64) (wraps bool, clearance float64) {
	clearance = stdmath.Inf(1)
	for i := range torusStationProbes {
		h := torusSecondHarmonicAt(t, co, float64(twoPi*float64(i)/torusStationProbes))
		l, ok := torusLaneAt(h, anchor)
		if !ok || l.trackDiscriminant(true) <= 0 {
			return false, 0
		}
		clearance = stdmath.Min(clearance, torusArcClearanceAt(t, h, l.upper))
	}
	return true, clearance
}

// torusArcClearanceAt is the arc length between one station's azimuth u and the NEAREST other azimuth
// the same station carries — the separation certificate for a single branch, measured the way
// torusBranchGapAt measures a pair's: as a length, so it compares against the stitch resolution.
//
// A station carrying u alone answers a whole turn: there is no second branch for the stitch to confuse
// it with, so nothing about it is ill-conditioned.
func torusArcClearanceAt(t Torus, h torusSecondHarmonic, u float64) float64 {
	least := twoPi
	for _, r := range h.azimuths() {
		if r != u {
			least = stdmath.Min(least, stdmath.Abs(shortestTurnDelta(u, r)))
		}
	}
	return float64(least * (t.MajorRadius + t.MinorRadius))
}

// torusWindowLoops builds the loop of every window this lane OWNS. A window whose branch pair merges at
// a flanking extremum instead of the lane's own belongs to a neighbouring lane and is skipped here; the
// azimuth count on the finished set is what proves that neighbour really carried it.
func torusWindowLoops(t Torus, co TorusCoForm, anchor float64, spans [][2]float64, res Resolution) ([]Curve3, SectionDecline) {
	var out []Curve3
	for _, w := range spans {
		loop := TorusSectionLoop{Torus: t, Co: co, V0: w[0], V1: w[1], UA: anchor}
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
func torusLaneOwnsWindow(l TorusSectionLoop) (owns, ok bool) {
	step := float64((l.V1 - l.V0) / torusWindowProbes)
	first, okA := torusLaneOwnsStation(l, l.V0+step)
	last, okB := torusLaneOwnsStation(l, l.V1-step)
	return first && last, okA && okB
}

// torusLaneOwnsStation reads the lane at one tube angle and asks whether its own extremum is the one
// closest to vanishing there.
func torusLaneOwnsStation(l TorusSectionLoop, v float64) (owns, ok bool) {
	lane, ok := torusLaneAt(torusSecondHarmonicAt(l.Torus, l.Co, v), l.UA)
	return ok && lane.mergesAtItsCenter(), ok
}

// torusCurvesAccountForEveryAzimuth certifies the finished curve SET against the stations themselves,
// and it is where the section's branch count is DECIDED rather than declared: at every probe tube angle
// the curves covering it must account for exactly the azimuths the station's quartic certifies there.
// The reduction never counts branches — it reads whatever real roots the station polynomial has and
// whatever extremum tracks label them — so this is the runtime certificate that the two agree.
//
// It is also what turns "a neighbouring lane carries this window" from a premise into a measurement: a
// dropped loop leaves two azimuths belonging to nothing, a doubled arc leaves one too many, and a lane
// whose root count changes across the turn shows up at the station where it changed.
//
// A probe within half a step of a window END is skipped: an end IS a fold, where the two azimuths have
// merged and the station carries one rather than two, so counting there would report a mismatch that is
// the fold's arithmetic rather than a missing loop. Every window the discriminant sampler found has an
// interior probe of its own, which is where a genuinely dropped loop shows up.
func torusCurvesAccountForEveryAzimuth(t Torus, co TorusCoForm, curves []Curve3) SectionDecline {
	step := twoPi / torusStationProbes
	for i := range torusStationProbes {
		v := float64(step * float64(i))
		if nearAWindowEnd(curves, v, float64(step/2)) {
			continue
		}
		if len(torusSecondHarmonicAt(t, co, v).azimuths()) != azimuthsCarriedAt(curves, v) {
			return DeclineTorusLaneUnaccounted
		}
	}
	return DeclineNone
}

// azimuthsCarriedAt is how many of a station's azimuths the section's curves account for at tube angle
// v: TWO for every folded loop whose window contains it — a loop runs out along one branch of its pair
// and back along the other — and ONE for every full-period arc, which carries a single branch at every
// station there is. A window is folded onto one period, so one that straddles the chart's seam counts
// like any other.
func azimuthsCarriedAt(curves []Curve3, v float64) int {
	n := 0
	for _, cv := range curves {
		switch c := cv.(type) {
		case TorusSectionLoop:
			if wrapAngle(v-c.V0) < c.V1-c.V0 {
				n += 2
			}
		case TorusSectionArc:
			n++
		}
	}
	return n
}

// nearAWindowEnd reports v sitting within reach of some loop's fold, where the station's two azimuths
// have merged into one. A full-period arc has no fold, so it never puts a station out of the census.
func nearAWindowEnd(curves []Curve3, v, reach float64) bool {
	for _, cv := range curves {
		l, ok := cv.(TorusSectionLoop)
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
func torusAzimuthAt(t Torus, co TorusCoForm, v, anchor float64, upper bool) float64 {
	st := co.stationOn(t, v)
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
func torusFoldAzimuth(t Torus, co TorusCoForm, v, anchor float64) float64 {
	st := co.stationOn(t, v)
	if st.invariant {
		return st.harmonic().foldRoot()
	}
	l, ok := torusLaneAt(st.secondHarmonic(), anchor)
	if !ok {
		return stdmath.NaN()
	}
	return l.center
}
