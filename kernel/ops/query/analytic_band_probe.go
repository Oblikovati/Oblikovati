// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"

	"sort"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// The BAND PROBE: how a face whose loops WRAP the parameter seam is handed a representative interior
// point, and how the side test reads one (Oblikovati/Oblikovati#2247, #3447, #3553).
//
// It is one responsibility and it lives in its own file because analytic_face_region.go had reached
// 864 lines against this repo's 500-line rule (review 2, M1). Nothing here changed in the move.

// bandSideOfEnclosedRegion is the side test for a face whose loops WRAP the seam, where one probe is
// not enough (Oblikovati/Oblikovati#3553).
//
// A wrapping region's chart is recorded as contours split at an artificial SLIT, and the material lies
// on both sides of one, so a classifier asked exactly ON it answers by which side its ray was cast
// from. The band probe used to propose the midpoint of the boundary's span, which on a region symmetric
// about its slit IS the slit, at every station — so the answer was a rounding artefact and came out
// differently for each of the four figure-eight rings in the corpus.
//
// So a STATION's several candidates are read together and must AGREE. They are placed at fractions of
// the span that no one constant-parameter line can share (bandAcrossFractions), so a slit can spoil at
// most one of them; a station whose candidates disagree is one where a probe landed on a real boundary,
// and it is skipped rather than believed. When no station is unanimous the side is not certified and
// the face declines, which is what it did before for an unprobeable face.
func bandSideOfEnclosedRegion(f *topo.Face, loops []faceLoop) (holds, certain bool) {
	s := f.Geometry()
	axis := bandAxisOf(loops)
	if u, v, ok := capInteriorUV(s, loops); ok {
		return brep.PointInFaceTrim(f, s.PointAt(u, v)), true
	}
	for _, station := range bandInteriorCandidates(loops, axis) {
		verdict, unanimous := unanimousTrimVerdict(f, s, axis, station)
		if unanimous {
			return verdict, true
		}
	}
	return false, false
}

// unanimousTrimVerdict reads brep.PointInFaceTrim at every candidate of one station and reports their
// common answer, or unanimous=false when they differ.
func unanimousTrimVerdict(f *topo.Face, s geom.Surface, axis bandAxis, station [][2]float64) (verdict, unanimous bool) {
	for i, c := range station {
		u, v := axis.pointOf(c[0], c[1])
		in := brep.PointInFaceTrim(f, s.PointAt(u, v))
		if i == 0 {
			verdict = in
			continue
		}
		if in != verdict {
			return false, false
		}
	}
	return verdict, len(station) > 0
}

// bandInteriorUV returns a point deep inside a band whose loops WRAP the parameter seam. Such a
// band is bounded in the parameter that closes and unbounded in the one that wraps, so at any fixed
// station of the wrapping parameter its boundary curves sit above and below: the midpoint between
// them is interior by construction, and it is far from the boundary rather than a hair off it.
//
// Stepping inward off a boundary chord instead — the previous construction — could land OUTSIDE the
// true region and go undetected, because the check available at that point compares against the
// loops' SAMPLED polygon, which does not track the true trim curve at that scale. A probe 5.2e-3
// outside the operand read as in-trim, the per-face gate then correctly refused a correct body, and
// a blind hole fell to a 1830-face rescue (Oblikovati/Oblikovati#2247).
func bandInteriorUV(loops []faceLoop) (u, v float64, ok bool) {
	axis := bandAxisOf(loops)
	along, across, found := bandInterior(loops, axis)
	if !found {
		return 0, 0, false
	}
	u, v = axis.pointOf(along, across)
	return u, v, true
}

// bandInterior is bandInteriorUV in the band's own (along, across) frame.
func bandInterior(loops []faceLoop, axis bandAxis) (along, across float64, ok bool) {
	for _, st := range bandInteriorCandidates(loops, axis) {
		if len(st) > 0 {
			return st[0][0], st[0][1], true
		}
	}
	return 0, 0, false // every station tried put the point in a hole
}

// bandInteriorCandidates are the interior points the band rule proposes, best first: at each station
// of the wrapping parameter, the across coordinate at several fractions of the boundary's span.
//
// Several fractions and not just the middle, because the middle is a SYMMETRY AXIS and a chart's
// artificial slit sits on one (Oblikovati/Oblikovati#3553). A region that wraps a whole period is
// recorded as contours split at a seam, and the slit is a constant-parameter line the material lies on
// BOTH sides of. The figure-eight cut face is symmetric about its slit, so the span's midpoint landed on
// it at every station — u = π, measured on all four rings of the corpus — and an even-odd count there
// answers by which side the ray was cast from. Three of the four rings then read their own interior as
// outside, took the complement of their own region, and integrated the far lobe: 70.92 mm² where
// 225.17 is the face. The fourth read it as inside, by nothing but its aspect ratio.
//
// The fractions are coprime-ish thirds and quarters around the middle rather than a nudge off it: a
// point at 1/3 of the span is as interior as the middle by the same argument, and no one line can be
// the midpoint, the third and the quarter of the same span at once. The caller decides between them by
// requiring the classifier to give the SAME answer at all of them (faceHoldsEnclosedRegion), so a
// candidate that lands on a real boundary shows up as a disagreement and the face declines rather than
// taking the answer a degenerate probe gave.
func bandInteriorCandidates(loops []faceLoop, axis bandAxis) [][][2]float64 {
	samples := allLoopSamples(loops, axis)
	if len(samples) < 2 {
		return nil
	}
	holes, per := nonWrappingPolygons(loops), loopsUVPeriod(loops)
	var out [][][2]float64
	for _, i := range bandStationOrder(len(samples)) {
		lo, hi, found := vSpanAt(samples, samples[i].u)
		if !found {
			continue
		}
		if st := acrossCandidatesAt(samples[i].u, lo, hi, axis, holes, per); len(st) > 0 {
			out = append(out, st)
		}
	}
	return out
}

// acrossCandidatesAt is one station's accepted across coordinates, in the order they are preferred.
func acrossCandidatesAt(station, lo, hi float64, axis bandAxis, holes [][]arcSample, per uvPeriod) [][2]float64 {
	var out [][2]float64
	for _, f := range bandAcrossFractions {
		across := lo + (hi-lo)*f
		hu, hv := axis.pointOf(station, across)
		if !uvCrossingsOdd(holes, hu, hv, per) {
			out = append(out, [2]float64{station, across})
		}
	}
	return out
}

// bandAcrossFractions are where across the boundary's span the band rule places its probes, and the
// MIDDLE is deliberately not among them.
//
// The middle is the one fraction a chart's artificial slit can occupy at every station at once: the
// slit is a constant-parameter line, its fractional position in the span is (slit − lo)/(hi − lo), and
// on a region symmetric about it — which the figure-eight cut face is — that is 1/2 everywhere.
// Measured on all four rings: the midpoint probe read the face's own interior as outside at every
// station, while the four fractions below read it as inside at every station. No single line can be
// the third, the two-thirds, the quarter and the three-quarters of one span, so a slit can take at
// most one of them, and a disagreement among them is what tells the caller the probe is not safe.
var bandAcrossFractions = []float64{1.0 / 3, 2.0 / 3, 0.25, 0.75}

// nonWrappingPolygons are the loops that close in the plane — the face's HOLES on a band, since a
// band's own bounding curves are the ones that wrap. The midpoint of the boundary's span can fall
// inside one of these, which is outside the face, so a station that does is rejected rather than
// trusted.
func nonWrappingPolygons(loops []faceLoop) [][]arcSample {
	var out [][]arcSample
	for i, fl := range loops {
		if closeUV(fl.netU, fl.netV, 0, 0) {
			out = append(out, loopUVPolygons(loops)[i])
		}
	}
	return out
}

// bandStationOrder walks candidate stations from the middle of the sample range outward, so an
// ordinary band answers on the first try and a holed one still gets alternatives to try.
func bandStationOrder(n int) []int {
	out := make([]int, 0, bandStationTries)
	for k := range bandStationTries {
		out = append(out, n*(2*k+1)/(2*bandStationTries))
	}
	return out
}

// bandStationTries is how many stations across the band are tried before declining. A band with
// holes needs more than one; a plain band answers on the first.
const bandStationTries = 9

// allLoopSamples flattens every loop's uv samples, ordered by the wrapping parameter so the median
// is a station the band actually spans. The u values are folded into ONE period first: the two rims
// of a band are unwrapped onto different branches of the covering space — one walks 0→2π, the next
// 2π→4π — so their raw parameters never meet even though the rims sit directly above one another,
// and a station window over raw u would see only one of them.
func allLoopSamples(loops []faceLoop, axis bandAxis) []arcSample {
	period := loopsPeriod(loops, axis)
	var out []arcSample
	for _, fl := range loops {
		for _, le := range fl.edges {
			for _, sp := range le.samples {
				a, x := axis.coordsOf(sp)
				out = append(out, arcSample{t: sp.t, u: foldU(a, period), v: x})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].u < out[j].u })
	return out
}

// loopsPeriod is the face's period in the wrapping parameter, or 0 when it does not wrap.
func loopsPeriod(loops []faceLoop, axis bandAxis) float64 {
	for _, fl := range loops {
		if p := axis.periodOf(fl); p > 0 {
			return p
		}
	}
	return 0
}

// vSpanAt returns the lowest and highest v the boundary reaches near the given u station. found is
// false when the boundary has no thickness there — a station at the very end of the band, where the
// midpoint would sit on the boundary itself.
func vSpanAt(samples []arcSample, station float64) (lo, hi float64, found bool) {
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	window := bandStationWindow * uSpanOf(samples)
	for _, s := range samples {
		if stdmath.Abs(s.u-station) <= window {
			lo, hi = stdmath.Min(lo, s.v), stdmath.Max(hi, s.v)
		}
	}
	return lo, hi, hi-lo > 0
}

// bandStationWindow is how much of the band's wrapping extent counts as "at this station" when
// reading the boundary's v span. Wide enough to catch both boundary curves through their sampling,
// narrow enough that the span is local rather than the band's whole height.
const bandStationWindow = 0.02 // tol:parametric — station window, relative to the band's u extent

// uSpanOf is the total extent the samples cover in the wrapping parameter.
func uSpanOf(samples []arcSample) float64 {
	return samples[len(samples)-1].u - samples[0].u
}
