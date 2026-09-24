// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The TORUS bucket of [IntersectSurfacesAnalytic] (ADR-0061 stage 5, generalised by ADR-0066). The
// reduction and the two curve types are in torus_section_arc.go, and what the other side has to supply
// is in torus_section_form.go — an implicit form whose restriction to a circle is degree two, which is
// every quadric AND a second torus. What is left here is the same TOPOLOGY question the ruled bucket
// asks, and periodicRootWindows answers it for both: which connected pieces the azimuths form over the
// tube's own period.
//
// There are three shapes, and the third is the one a type-driven dispatch would have missed. Where the
// quadric reaches the tube at EVERY tube angle the section is two closed curves, one per branch. Where
// it reaches it over part of the turn the section is one closed loop per window, folded at the ends.
// And where the quadric is COAXIAL with the torus its constraint has no azimuth dependence at all — the
// section is whole circles at the tube angles that satisfy it, which is a boss or a shaft standing in
// the ring's hole.

// TorusSection returns the exact intersection of a torus with another surface's implicit form, on the
// torus's own chart. The form is CLASSIFIED once — no azimuth dependence, one harmonic, or two
// — and exactly one reduction runs: the coaxial circles, the one-harmonic arccos of this file, or the
// general second-harmonic lanes of intersect_torus_section_skew.go. ok=false always carries the reason it
// refused ([SectionDecline]), so a caller can tell a CONDITIONING demotion — a closed form that applies
// but cannot name its answer at these numbers — from "no closed form claims this pair", and record the
// first as the degradation it is.
//
//	curves, why, ok := geom.TorusSection(ring, linkedRing, geom.ResolutionForBox(box))
func TorusSection(t Torus, co TorusCoForm, res Resolution) ([]Curve3, SectionDecline, bool) {
	curves, why, ok := torusSectionOfFamily(t, co, res)
	if !ok || len(curves) == 0 {
		return curves, why, ok
	}
	if !torusSectionSatisfiesItsForm(t, co, curves) {
		return nil, DeclineTorusSectionOffItsForm, false
	}
	return curves, why, ok
}

// torusSectionOfFamily runs the ONE reduction the classification selects.
func torusSectionOfFamily(t Torus, co TorusCoForm, res Resolution) ([]Curve3, SectionDecline, bool) {
	switch co.sectionFamily(t) {
	case torusFamilyCoaxial:
		curves, ok := torusCoaxialCircles(t, co)
		return curves, noClosedFormWhen(ok), ok
	case torusFamilyLanes:
		return torusSkewSection(t, co, res)
	}
	return torusOneHarmonicSection(t, co, res)
}

// torusSectionSatisfiesItsForm is the POST-CONDITION of every torus section: each curve, sampled along
// its OWN parameter, must lie on the OTHER surface — measured as a DISTANCE, against the modelling weld
// the operands' own size sets.
//
// It exists because every certificate before it certifies a PART. A root is certified where it is
// solved; a FOLD azimuth is not a root at all but the station's own extremum, read because the two
// branches have merged there; a lane is labelled by an anchor carried from another station; and the
// azimuth census counts branches rather than placing them. Each is sound on its own and the composition
// can still put a point off the surface.
//
// Two co-centred PERPENDICULAR rings do exactly that, and they are the reason this exists. Their branch
// pair is tangent at v = 0 and v = π — the station drops from four roots to two there — so the turn
// splits into two windows whose folds are that tangency, and the fold reads a lane extremum that is not
// the merged root. Measured: the section came back ok=true, why=none, with points 1.353e-5 off the
// surface they claimed to be on, past the anchor gate, the ownership gate, the separation gate and the
// azimuth census alike (torus_torus_test.go).
//
// The measure is a DISTANCE and not the station polynomial's residual, and that is the whole point. At
// a fold df/du is zero by definition, so f falls off QUADRATICALLY in the azimuth error: that same
// 1.353e-5 displacement reads as a residual far under what a certified root is allowed, and a residual
// gate is blindest exactly where this failure lives. "Certify a root or branch choice at runtime against
// the geometry (position, second-order test)" is a statement about where the curve IS, and only a length
// reads it. The point is on the CHART by construction — it is the chart's own PointAt — so the one
// surface it can be off is the co-form's.
func torusSectionSatisfiesItsForm(chart Torus, co TorusCoForm, curves []Curve3) bool {
	weld := torusChartWeld(chart)
	for _, c := range curves {
		for i := range torusSectionCertificateSamples {
			at := float64(float64(i) / (torusSectionCertificateSamples - 1))
			if !withinWeld(co.distanceTo(c.PointAt(at)), weld) {
				return false
			}
		}
	}
	return true
}

// withinWeld reports d being a real distance no larger than the weld. It is written as a positive test
// rather than as `d > weld` so that a NON-FINITE distance refuses: an unreadable station makes the lane
// reader answer NaN by design, that NaN propagates into a coordinate, and `NaN > weld` is FALSE — a
// comparison written the other way round would admit exactly the point that has no position at all.
//
// This makes the reading right where it looks; it does not make the gate a NaN detector. An unreadable
// station is a measure-zero event, and a point sample steps over one: the case that found this
// (a cylinder at origin (−3.3617, 6.2435, 0.0778), axis (0.2568, −0.9663, 0.0158), radius 0.8992,
// against the corpus ring) shows no NaN at 257 samples per curve and two at 1025. It is a PRE-EXISTING
// defect of the quadric family, recorded with its fixture in ADR-0066's follow-up, and its fix is that
// the lane reader should refuse rather than answer NaN — not a finer grid.
func withinWeld(d, weld float64) bool { return d <= weld }

// torusChartWeld is the length the post-condition judges an off-surface point against: the modelling
// weld at the CHART TORUS'S OWN reach, R + r.
//
// It is derived from the operand rather than read from the caller's Resolution, and that is deliberate.
// The closed-surface pairings hand in geom.ResolutionForBox(faceLoopBox(f)), and a BOUNDARY-LESS face —
// the bare ball and the bare torus, which is every torus pair before anything has been imprinted on it
// — has no loops, so that box is EMPTY and the resolution collapses to a weld of ~1e-18. That is below
// the operands' own float noise; a gate reading it would refuse every exact torus section there is.
// kernel/brep's declineOpenSection records the same defect and works around it by loosening its class
// to Sew(); loosening is not available here, because Sew() at these sizes is 2e-3 and the excursion
// this gate exists to catch is 1e-5.
//
// The chart's reach is the right scale on its own terms: the points measured are the chart's own
// PointAt, so what "close to the other surface" means is set by the size of the surface they came from.
// It is model-relative in ADR-0042's sense — the same pair in metres and in millimetres gates the same
// — and it does not depend on a caller getting its box right.
func torusChartWeld(chart Torus) float64 {
	return ResolutionForSize(chart.MajorRadius + chart.MinorRadius).Weld()
}

// torusSectionCertificateSamples is how many points of each section curve the post-condition reads, and
// it is a MEASURED number rather than a round one.
//
// The failure it exists for is narrow: the perpendicular rings' fold excursion peaks at 1.353e-5 around
// t = 0.9875 and has fallen to 5.6e-10 by t = 0.999, so a grid of 65 steps clean over it (worst 3.6e-15)
// and a grid of 97 does not. 257 catches it with a 500-fold margin on the weld, and costs a third more
// than the section it certifies (193 ms → 257 ms for twenty skew-rod sections). A spike narrower than
// one part in 256 of a curve's own parameter is still stepped over; that is a bound this gate has, and
// refining it belongs with the fold refinement ADR-0065 already owes.
const torusSectionCertificateSamples = 257

// torusSectionFamily is what ONE classification of a (torus, form) pair selects. Exactly one of the
// three runs; there is no order to fall through and nothing is tried twice.
type torusSectionFamily uint8

const (
	// torusFamilyLanes keeps the second harmonic: up to four azimuths per station, paired into lanes.
	torusFamilyLanes torusSectionFamily = iota
	// torusFamilyOneHarmonic collapses to level + reach·cos(u − phase): two ordered azimuths, an arccos.
	torusFamilyOneHarmonic
	// torusFamilyCoaxial has no azimuth dependence at all: the section is whole tube circles.
	torusFamilyCoaxial
)

// torusOneHarmonicSection is the arccos family's topology: the maximal tube-angle windows where the
// two azimuths exist, or two full-period branches when they exist everywhere.
//
// THE ok OF torusHarmonicAt CANNOT BE FALSE ANYWHERE ON THIS PATH, and that is a construction rather
// than an assumption (#3527). The flag is st.invariant, which for a quadric is axisInvariantEntries over
// the in-plane tensor entries — and those do not depend on the tube angle at all, so it is one answer for
// the whole turn. sectionFamily has already read exactly that answer (quadricIsAxisInvariant) to select
// this family, and the torus co-form's own sectionFamily never returns torusFamilyOneHarmonic, so nothing
// else reaches here.
//
// This file reads torusHarmonicAt THREE times and they do not agree about that, which is worth saying
// rather than leaving a reader to find (#3527 review 1, Important 3). Two — here and in
// torusHarmonicLoops — discard the flag, which is right by the paragraph above: testing it would ask one
// question twice. The third, in torusFullTurnSection, TESTS it and returns DeclineNoClosedForm; that
// function is reached only from this one's `case !folded`, under the same classification, so its decline
// is UNREACHABLE — dead in the same way crossingSegment's ok=false is dead in kernel/brep. It is kept
// there rather than dropped because it guards a public-looking entry that a future caller could reach
// from outside the classification, and because a bare anchor read past a non-invariant station would be
// silent nonsense rather than a decline. If that entry ever gains a second caller, the flag is load
// bearing again and this paragraph is what says so.
func torusOneHarmonicSection(t Torus, co TorusCoForm, res Resolution) ([]Curve3, SectionDecline, bool) {
	spans, folded := periodicRootWindows(func(v float64) float64 {
		h, _ := torusHarmonicAt(t, co, v)
		return h.discriminant()
	}, torusStationProbes)
	switch {
	case !folded:
		return torusFullTurnSection(t, co, res) // the form reaches the tube at every station
	case len(spans) == 0:
		return nil, DeclineNone, true // it reaches the tube nowhere: they do not meet, and that is an answer
	}
	return torusHarmonicLoops(t, co, spans, res)
}

// sectionFamily classifies a QUADRIC against a torus, from the quadric's own tensor. The tensor test is
// a statement about every tube angle at once, which is why it is made here and not station by station.
func (q Quadric) sectionFamily(t Torus) torusSectionFamily {
	_, e1, e2 := torusAxisFrame(t)
	if _, invariant := quadricIsAxisInvariant(q, e1, e2); !invariant {
		return torusFamilyLanes
	}
	if quadricReachesNoAzimuth(t, q) {
		return torusFamilyCoaxial
	}
	return torusFamilyOneHarmonic
}

// sectionFamily classifies a second TORUS against the chart. A torus co-form is one-harmonic exactly
// when it is COAXIAL with the chart, and then it carries no azimuth dependence at all — so there is no
// middle family for it, and the two questions the quadric asks separately are one question here.
//
// Why: the second harmonic of a torus station is Cos2 = (g₁²−g₂²)/2 + βρ²(n₁²−n₂²)/2 and
// Sin2 = g₁g₂ + βρ²n₁n₂ (torus_torus_harmonic.go), which is the traceless part of ggᵀ + βρ²nnᵀ — a sum
// of two positive-semidefinite rank-one forms. That vanishes only when the two are orthogonal with
// equal weight, or when both are zero. ρ varies over the turn and the weights do not track it, so
// "orthogonal with equal weight" cannot hold at every station: over the whole turn, both must be zero.
// g = 0 puts the chart's tube-circle centre on the other torus's axis and n = 0 makes the axes
// parallel, which is coaxial, and then the FIRST harmonic (Cos1, Sin1) is zero as well.
//
// So the classification is that geometric statement, read ONCE — it is not sampled, because how many
// curves the section has is a topological question and a grid cannot answer one.
func (t Torus) sectionFamily(chart Torus) torusSectionFamily {
	if torusIsCoaxialWith(t, chart) {
		return torusFamilyCoaxial
	}
	return torusFamilyLanes
}

// torusIsCoaxialWith reports the two tori sharing an axis: their axes parallel, and the co-form's centre
// on the chart's axis. The first is a dimensionless direction test and the second a LENGTH against the
// chart's own weld, which is the class each belongs to.
//
// A pair this admits that is not exactly coaxial does not escape: [torusSectionSatisfiesItsForm] then
// measures the circles it built against the co-form's surface and refuses them by name.
func torusIsCoaxialWith(co, chart Torus) bool {
	axis := chart.AxisDir.AsVector()
	if float64(co.AxisDir.AsVector().Cross(axis).Length()) > coincidentNormalSinTol {
		return false
	}
	off := chart.Center.VectorTo(co.Center)
	perp := off.Sub(axis.Scale(math.Scalar(float64(off.Dot(axis)))))
	return float64(perp.Length()) <= torusChartWeld(chart)
}

// noClosedFormWhen names the ordinary refusal for a step whose only answer is a bool.
func noClosedFormWhen(ok bool) SectionDecline {
	if ok {
		return DeclineNone
	}
	return DeclineNoClosedForm
}

// torusHarmonicLoops builds one folded loop per tube-angle window of the one-harmonic reduction. It
// discards torusHarmonicAt's ok for the reason torusOneHarmonicSection records above: the family
// classification has already established it, for the whole turn, before either was called.
func torusHarmonicLoops(t Torus, co TorusCoForm, spans [][2]float64, res Resolution) ([]Curve3, SectionDecline, bool) {
	out := make([]Curve3, 0, len(spans))
	for _, w := range spans {
		anchor, _ := torusHarmonicAt(t, co, float64((w[0]+w[1])/2))
		loop := TorusSectionLoop{Torus: t, Co: co, V0: w[0], V1: w[1], UA: anchor.phase}
		if !torusWindowConditioning(loop, res) {
			return nil, DeclineTorusLaneSeparation, false
		}
		out = append(out, loop)
	}
	return out, DeclineNone, true
}

// torusStationProbes is how many tube angles the window finder samples. The harmonic's discriminant is a
// low-order trigonometric polynomial in v for every axis-invariant quadric, so this brackets every sign
// change; it matches the ruled bucket's azimuth sweep so the two forms resolve at the same rate.
const torusStationProbes = ruledQuadricAzimuthProbes

// quadricReachesNoAzimuth reports that an AXIS-INVARIANT quadric's constraint on the torus carries no
// azimuth dependence AT ANY tube angle, so the section is whole tube circles rather than two azimuths.
//
// It is decided in CLOSED FORM, not sampled. The reach's two components are T(v)·ê with
// T(v) = 2(M·W₀(v) + G) and W₀(v) = w + r·sin v·â, so each is
//
//	2(M w + G)·ê + r·sin v · 2(M â)·ê
//
// — of the form A + B·sin v, which vanishes at every v exactly when A and B both vanish. Four scalars,
// read once. The predicate this replaces asked the same question at 720 tube angles; a function of that
// shape with 720 zeros is identically zero, so the two agree wherever the old one was right, and this
// one cannot be stepped over.
func quadricReachesNoAzimuth(t Torus, q Quadric) bool {
	axis, e1, e2 := torusAxisFrame(t)
	base := q.M.Apply(q.Anchor.VectorTo(t.Center)).Add(q.G).Scale(2)
	swing := q.M.Apply(axis).Scale(math.Scalar(2 * t.MinorRadius))
	for _, e := range [2]math.Vector3{e1, e2} {
		if float64(base.Dot(e)) != 0 || float64(swing.Dot(e)) != 0 {
			return false
		}
	}
	return true
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
		Radius: t.MajorRadius + float64(t.MinorRadius*cv),
	}
}

// torusFullTurnSection returns the ONE-harmonic form's two branches as full-period arcs, for a co-form
// that reaches the tube at every station. Its lane is the harmonic's phase — recorded for the same
// reason [TorusSectionLoop] records one, and read by nothing on this path, because the arccos names its
// own two branches.
//
// Its DeclineNoClosedForm exit is UNREACHABLE from the one caller it has: torusOneHarmonicSection's
// `case !folded` runs under a classification that has already established the tensor is axis-invariant,
// which is the whole of what torusHarmonicAt's ok reports (see that function's doc). The test stays
// because reading an anchor past a non-invariant station would be silent nonsense rather than a decline,
// and a second caller would make the flag load-bearing again (#3527 review 1).
func torusFullTurnSection(t Torus, co TorusCoForm, res Resolution) ([]Curve3, SectionDecline, bool) {
	anchor, ok := torusHarmonicAt(t, co, 0)
	if !ok {
		return nil, DeclineNoClosedForm, false
	}
	if !torusWrapConditioning(t, co, anchor.phase, res) {
		return nil, DeclineTorusLaneSeparation, false
	}
	return []Curve3{
		TorusSectionArc{Torus: t, Co: co, Upper: false, UA: anchor.phase, V0: 0, V1: twoPi},
		TorusSectionArc{Torus: t, Co: co, Upper: true, UA: anchor.phase, V0: 0, V1: twoPi},
	}, DeclineNone, true
}

// torusWrapConditioning certifies the ONE-harmonic form's branch pair before full-period arcs are built
// on it: the pair must span more than the stitch resolution at EVERY station. It reads the MINIMUM
// across the turn because a wrap has no fold — the mirror of [torusWindowConditioning], which reads the
// maximum inside a window because a window's branches meet at both ends by construction.
//
// The general reduction certifies its own wrap differently, and has to: it builds ONE arc per track,
// not a pair, so what it must know is each branch's clearance from its neighbours rather than a pair's
// span (torusArcClearanceAt). The two are the same statement only where a station carries exactly two
// azimuths, which is what makes this form the one-harmonic form's.
func torusWrapConditioning(t Torus, co TorusCoForm, anchor float64, res Resolution) bool {
	least := stdmath.Inf(1)
	for i := range torusStationProbes {
		v := float64(twoPi * float64(i) / torusStationProbes)
		least = stdmath.Min(least, torusBranchGapAt(t, co, v, anchor))
	}
	return least > res.Stitch()
}

// torusBranchGap is the arc length between the two azimuths at one station — the branches' separation
// measured as a LENGTH, so it compares against the stitch resolution on the same footing the ruled
// form's ruling-parameter gap does.
func torusBranchGap(t Torus, h torusHarmonic) float64 {
	if h.reach == 0 {
		return 0
	}
	arg := stdmath.Max(-1, stdmath.Min(1, float64(-h.level/h.reach)))
	return float64(2 * stdmath.Acos(arg) * (t.MajorRadius + t.MinorRadius))
}

// torusWindowConditioning certifies one tube-angle window before a loop is built on it: its two azimuths
// must separate, somewhere inside, by more than the stitch resolution. It is the mirror of the wrap
// form's gate — that one reads the MINIMUM across the turn because a wrap has no fold, this one the
// MAXIMUM inside the window because a window's branches meet at both ends by construction.
func torusWindowConditioning(l TorusSectionLoop, res Resolution) bool {
	widest := 0.0
	for i := 1; i < torusWindowProbes; i++ {
		v := l.V0 + float64((l.V1-l.V0)*float64(i)/torusWindowProbes)
		widest = stdmath.Max(widest, torusBranchGapAt(l.Torus, l.Co, v, l.UA))
	}
	return widest > res.Stitch()
}

// torusBranchGapAt is the arc length a branch pair spans at one tube angle, whichever reduction the
// station takes: the one-harmonic arccos, or the lane the anchor names in the general one. It is the
// one place the two forms' separations are read, so the conditioning gates above apply the same
// certificate to both.
func torusBranchGapAt(t Torus, co TorusCoForm, v, anchor float64) float64 {
	st := co.stationOn(t, v)
	if st.invariant {
		return torusBranchGap(t, st.harmonic())
	}
	l, ok := torusLaneAt(st.secondHarmonic(), anchor)
	if !ok {
		return 0 // an unreadable station: a zero gap fails the gate, which is the decline
	}
	return float64(l.separation() * (t.MajorRadius + t.MinorRadius))
}

// torusWindowProbes samples a window's interior for its widest branch separation, which has one interior
// maximum for every axis-invariant quadric, so a coarse sweep finds it.
const torusWindowProbes = 64
