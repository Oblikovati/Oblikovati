// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The torus∩quadric bucket of [IntersectSurfacesAnalytic] (ADR-0061 stage 5). The reduction and the two
// curve types are in torus_quadric_arc.go; what is left here is the same TOPOLOGY question the ruled
// bucket asks, and periodicRootWindows answers it for both: which connected pieces the two azimuths
// form over the tube's own period.
//
// There are three shapes, and the third is the one a type-driven dispatch would have missed. Where the
// quadric reaches the tube at EVERY tube angle the section is two closed curves, one per branch. Where
// it reaches it over part of the turn the section is one closed loop per window, folded at the ends.
// And where the quadric is COAXIAL with the torus its constraint has no azimuth dependence at all — the
// section is whole circles at the tube angles that satisfy it, which is a boss or a shaft standing in
// the ring's hole.

// TorusQuadricSection returns the exact intersection of a torus with an implicit quadric, on the
// torus's own chart. ok=false when the quadric's quadratic form is not invariant about the torus axis —
// a skew rod, a tilted cone — because the azimuth dependence is then a second harmonic whose roots are
// not this closed form's.
//
//	curves, ok := geom.TorusQuadricSection(ring, drill.QuadricForm(), geom.ResolutionForBox(box))
func TorusQuadricSection(t Torus, q Quadric, res Resolution) ([]Curve3, bool) {
	_, e1, e2 := torusAxisFrame(t)
	if _, ok := quadricIsAxisInvariant(q, e1, e2); !ok {
		return nil, false
	}
	if coaxialTorusQuadric(t, q) {
		return torusCoaxialCircles(t, q)
	}
	spans, ok := periodicRootWindows(func(v float64) float64 {
		h, _ := torusHarmonicAt(t, q, v)
		return h.discriminant()
	}, torusStationProbes)
	if !ok {
		return torusFullTurnSection(t, q, res) // the quadric reaches the tube at every station
	}
	if len(spans) == 0 {
		return nil, true // it reaches the tube nowhere: they do not meet, and that is an answer
	}
	out := make([]Curve3, 0, len(spans))
	for _, w := range spans {
		loop := TorusQuadricLoop{Torus: t, Quad: q, V0: w[0], V1: w[1]}
		if !torusWindowConditioning(loop, res) {
			return nil, false
		}
		out = append(out, loop)
	}
	return out, true
}

// torusStationProbes is how many tube angles the window finder samples. The harmonic's discriminant is a
// low-order trigonometric polynomial in v for every axis-invariant quadric, so this brackets every sign
// change; it matches the ruled bucket's azimuth sweep so the two forms resolve at the same rate.
const torusStationProbes = ruledQuadricAzimuthProbes

// coaxialTorusQuadric reports that the quadric's constraint on the torus carries NO azimuth dependence:
// its reach is zero at every station, so the two roots are not two azimuths but a whole circle. That is
// the coaxial cylinder, cone or centred sphere, and its section is circles rather than curves.
func coaxialTorusQuadric(t Torus, q Quadric) bool {
	for i := range torusStationProbes {
		h, ok := torusHarmonicAt(t, q, twoPi*float64(i)/torusStationProbes)
		if !ok || h.reach != 0 {
			return false
		}
	}
	return true
}

// torusCoaxialCircles returns the tube circles where a coaxial quadric meets the torus: the roots of the
// harmonic's level term, each a full azimuth sweep at one tube angle. A root the level only GRAZES — a
// tangency, where the level touches zero without crossing — is not a section and is left out.
func torusCoaxialCircles(t Torus, q Quadric) ([]Curve3, bool) {
	level := func(v float64) float64 {
		h, _ := torusHarmonicAt(t, q, v)
		return h.level
	}
	var out []Curve3
	prev := level(0)
	for i := 1; i <= torusStationProbes; i++ {
		v := twoPi * float64(i) / torusStationProbes
		cur := level(v)
		if (prev > 0) != (cur > 0) {
			out = append(out, torusStationCircle(t, bisectLevelRoot(level, twoPi*float64(i-1)/torusStationProbes, v)))
		}
		prev = cur
	}
	return out, true
}

// torusStationCircle is the full azimuth sweep at one tube angle: a circle about the torus axis, of the
// radial distance the tube reaches there, at the height the tube reaches there.
func torusStationCircle(t Torus, v float64) Circle {
	cv, sv := cosSin(v)
	axis := t.AxisDir.AsVector()
	centre := t.Center.TranslateBy(axis.Scale(math.Scalar(t.MinorRadius * sv)))
	return Circle{
		Center: centre,
		Normal: t.AxisDir,
		RefDir: t.Ref,
		Radius: t.MajorRadius + t.MinorRadius*cv,
	}
}

// bisectLevelRoot refines a bracketed sign change of the coaxial level term to the tube angle itself.
func bisectLevelRoot(level func(float64) float64, lo, hi float64) float64 {
	loPositive := level(lo) > 0
	for range foldBisectionSteps {
		mid := (lo + hi) / 2
		if (level(mid) > 0) == loPositive {
			lo = mid
			continue
		}
		hi = mid
	}
	return (lo + hi) / 2
}

// torusFullTurnSection returns the two branches as full-period arcs, for a quadric that reaches the tube
// at every station. ok=false when the two branches come close enough to be one curve at the modelling
// resolution — the same separation certificate the ruled wrap form applies, and for the same reason: two
// branches the stitch cannot tell apart are not two curves.
func torusFullTurnSection(t Torus, q Quadric, res Resolution) ([]Curve3, bool) {
	least := stdmath.Inf(1)
	for i := range torusStationProbes {
		h, ok := torusHarmonicAt(t, q, twoPi*float64(i)/torusStationProbes)
		if !ok {
			return nil, false
		}
		least = stdmath.Min(least, torusBranchGap(t, h))
	}
	if least <= res.Stitch() {
		return nil, false
	}
	return []Curve3{
		TorusQuadricArc{Torus: t, Quad: q, Upper: false, V0: 0, V1: twoPi},
		TorusQuadricArc{Torus: t, Quad: q, Upper: true, V0: 0, V1: twoPi},
	}, true
}

// torusBranchGap is the arc length between the two azimuths at one station — the branches' separation
// measured as a LENGTH, so it compares against the stitch resolution on the same footing the ruled
// form's ruling-parameter gap does.
func torusBranchGap(t Torus, h torusHarmonic) float64 {
	if h.reach == 0 {
		return 0
	}
	arg := stdmath.Max(-1, stdmath.Min(1, -h.level/h.reach))
	return 2 * stdmath.Acos(arg) * (t.MajorRadius + t.MinorRadius)
}

// torusWindowConditioning certifies one tube-angle window before a loop is built on it: its two azimuths
// must separate, somewhere inside, by more than the stitch resolution. It is the mirror of the wrap
// form's gate — that one reads the MINIMUM across the turn because a wrap has no fold, this one the
// MAXIMUM inside the window because a window's branches meet at both ends by construction.
func torusWindowConditioning(l TorusQuadricLoop, res Resolution) bool {
	widest := 0.0
	for i := 1; i < torusWindowProbes; i++ {
		v := l.V0 + (l.V1-l.V0)*float64(i)/torusWindowProbes
		h, ok := torusHarmonicAt(l.Torus, l.Quad, v)
		if !ok {
			return false
		}
		widest = stdmath.Max(widest, torusBranchGap(l.Torus, h))
	}
	return widest > res.Stitch()
}

// torusWindowProbes samples a window's interior for its widest branch separation, which has one interior
// maximum for every axis-invariant quadric, so a coarse sweep finds it.
const torusWindowProbes = 64
