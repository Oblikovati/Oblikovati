// SPDX-License-Identifier: GPL-2.0-only

package geom

import stdmath "math"

// The BRANCH PAIRING of the second-harmonic torus section (ADR-0061 stage 5, third slice).
//
// The one-harmonic station has exactly two azimuths and they are ordered by construction: phase ±
// arccos, one on each side of the harmonic's own peak. A second-harmonic station has up to FOUR, and
// which two of them are one closed branch pair is a real question. A rod driven ACROSS a ring answers
// it plainly: the infinite rod pierces the tube TWICE at every tube angle it reaches — once near the
// ring's +x flank and once near its −x — so four azimuths, in two pairs that never meet each other.
//
// A pair is named by the EXTREMUM it straddles. df/du is itself a degree-two trigonometric polynomial,
// so a station has two or four extrema alternating round the circle, and between two neighbouring
// extrema f is monotone and carries at most one root. A LANE is one extremum together with the root on
// each side of it: the arc it bounds is where f keeps one sign, and the pair merges — the FOLD — where
// that arc collapses onto the extremum itself. That is exactly what the one-harmonic form does with
// arccos → 0, so the two forms fold the same way and share periodicRootWindows.
//
// The lane's discriminant generalises the harmonic's reach² − level² term for term: with c the lane's
// extremum and c⁻, c⁺ its neighbours, min(−f(c)·f(c⁻), −f(c)·f(c⁺)) is positive exactly while the two
// roots exist, and for a one-harmonic station (whose only extrema are its peak and its trough) it IS
// reach² − level².
//
// Two things are certified rather than assumed. A lane is a lane only if its pair merges at its OWN
// extremum: every extremum has flanks of the opposite sign when four roots are present, so the
// COMPLEMENTARY pairing (the arcs of the other sign) reports the same windows, and it is rejected
// because its own extremum does not vanish at the fold. And the extremum tracks must stay separable
// across the tube's whole turn, or naming a lane by a fixed anchor azimuth would be a guess.

// torusLane is one branch pair of a second-harmonic station: the extremum it straddles and the root on
// each side of it.
type torusLane struct {
	center       float64    // the extremum the pair straddles — the lane's label, and its merged azimuth at a fold
	value        float64    // f at the centre
	flanks       [2]float64 // f at the neighbouring extrema, the decreasing-u side first
	lower, upper float64    // the two azimuths; both the centre at and beyond a fold
}

// torusLaneAt returns the lane of h whose extremum is nearest anchor. Every azimuth it reports is a
// CERTIFIED root of the station polynomial (or, at a fold, the station's OWN extremum, which both
// branches share there) — nothing here picks a root by index or by a hard-coded branch.
//
// ok=false is a station with fewer than two extrema, which is a station polynomial with no azimuth
// dependence at all. There is no lane to read there, and the anchor is a seed from ANOTHER station: it
// is not a root of this one, so returning it would put a point on the torus that is not on the quadric
// and say nothing. The caller declines instead.
func torusLaneAt(h torusSecondHarmonic, anchor float64) (torusLane, bool) {
	ex := h.extrema()
	if len(ex) < 2 {
		return torusLane{}, false
	}
	i, n := nearestAngleIndex(ex, anchor), len(ex)
	prev, next := ex[(i+n-1)%n], ex[(i+1)%n]
	roots := h.azimuths()
	return torusLane{
		center: ex[i],
		value:  h.valueAt(ex[i]),
		flanks: [2]float64{h.valueAt(prev), h.valueAt(next)},
		lower:  arcRootFrom(roots, ex[i], prev, false),
		upper:  arcRootFrom(roots, ex[i], next, true),
	}, true
}

// discriminant is positive exactly where the lane's two azimuths exist and distinct, and crosses zero
// at each fold — the same contract [torusHarmonic.discriminant] has, so periodicRootWindows reads both.
func (l torusLane) discriminant() float64 {
	return stdmath.Min(-l.value*l.flanks[0], -l.value*l.flanks[1])
}

// root returns the lane's upper (increasing-u) or lower azimuth.
func (l torusLane) root(upper bool) float64 {
	if upper {
		return l.upper
	}
	return l.lower
}

// separation is the azimuth the pair spans through its own extremum — zero at the fold where the two
// have merged, and the counterpart of the harmonic form's 2·arccos.
func (l torusLane) separation() float64 {
	return turnBetween(l.center, l.upper, true) + turnBetween(l.center, l.lower, false)
}

// mergesAtItsCenter reports that the pair meets at the lane's OWN extremum rather than at a flanking
// one. At a fold the vanishing quantity is whichever of the three the discriminant's zero came from,
// so this is a comparison of computed values against each other — no absolute floor. A lane whose fold
// belongs to a flank is the COMPLEMENTARY arc of its neighbours' lanes, already carried by them.
func (l torusLane) mergesAtItsCenter() bool {
	return stdmath.Abs(l.value) < stdmath.Min(stdmath.Abs(l.flanks[0]), stdmath.Abs(l.flanks[1]))
}

// arcRootFrom returns the root nearest `from` inside the open arc running from `from` toward `to` in
// the given direction, or `from` itself when that arc holds none. The empty case IS the fold: the two
// roots have merged onto the extremum, both branches read the same azimuth, and a loop built on the
// window closes exactly on its ends.
func arcRootFrom(roots []float64, from, to float64, forward bool) float64 {
	span, best, out := turnBetween(from, to, forward), stdmath.Inf(1), from
	for _, r := range roots {
		d := turnBetween(from, r, forward)
		if d <= 0 || d >= span || d >= best {
			continue
		}
		best, out = d, r
	}
	return out
}

// turnBetween is the angle from a to b travelling in the named direction, in [0, 2π). Azimuths carry
// an arbitrary whole turn, so every comparison between two of them folds onto one period first.
func turnBetween(a, b float64, forward bool) float64 {
	if forward {
		return wrapAngle(b - a)
	}
	return wrapAngle(a - b)
}

// nearestAngleIndex returns the index of the angle closest to a the short way round.
func nearestAngleIndex(xs []float64, a float64) int {
	best, at := stdmath.Inf(1), 0
	for i, x := range xs {
		if d := stdmath.Abs(shortestTurnDelta(a, x)); d < best {
			best, at = d, i
		}
	}
	return at
}

// minimumAngleGap is the smallest shortest-turn separation between any two of the angles, which is the
// spacing a track has to stay well inside to keep its identity.
func minimumAngleGap(xs []float64) float64 {
	least := twoPi
	for i, a := range xs {
		for _, b := range xs[i+1:] {
			least = stdmath.Min(least, stdmath.Abs(shortestTurnDelta(a, b)))
		}
	}
	return least
}

// torusLaneAnchors returns one azimuth per extremum TRACK of the station polynomial — the labels that
// keep a branch pair on the same lane across the tube's turn. ok=false when the tracks are not
// separable at the station spacing: the extremum count changes, a track drifts past half the spacing,
// or two tracks claim the same extremum. Any of those makes the pairing a guess, and the pair demotes
// to the general marcher rather than being named wrongly.
func torusLaneAnchors(t Torus, q Quadric) ([]float64, bool) {
	seed := torusSecondHarmonicAt(t, q, 0).extrema()
	if len(seed) < 2 {
		return nil, false // a station with no extremum to name a lane by
	}
	reach := minimumAngleGap(seed) / 2
	for i := 1; i < torusStationProbes; i++ {
		ex := torusSecondHarmonicAt(t, q, twoPi*float64(i)/torusStationProbes).extrema()
		if !anglesTrackSeeds(seed, ex, reach) {
			return nil, false
		}
	}
	return seed, true
}

// anglesTrackSeeds reports each seed claiming exactly one of this station's extrema, within reach and
// one to one.
func anglesTrackSeeds(seed, ex []float64, reach float64) bool {
	if len(ex) != len(seed) {
		return false
	}
	claimed := make([]bool, len(ex))
	for _, a := range seed {
		i := nearestAngleIndex(ex, a)
		if claimed[i] || stdmath.Abs(shortestTurnDelta(a, ex[i])) > reach {
			return false
		}
		claimed[i] = true
	}
	return true
}
