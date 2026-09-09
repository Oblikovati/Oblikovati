// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The chart-metric handedness reader: which way does a face's own boundary walk, seen along the chart
// normal S_u×S_v? It is ONE question — "is the face's material to the left of this segment" — asked at a
// few dozen stations and settled by majority, and it is the sole authority for a face's stored sense
// (senseFromLoopWinding). It lives apart from the shell probing in orient_consistent.go because it is a
// different responsibility: that file decides a SHELL's role and its one free bit, this one decides a
// FACE's own reading, and the shell file only multiplies the two together.
//
// Everything here turns on the step being a quarter of the segment's own ARC LENGTH. It used to be a
// quarter TURN in (u, v), which is the same thing only where the chart is metrically isotropic — see
// quarterArcLeftOf, and Oblikovati/Oblikovati#3512 for the bore that integrated as ADDED because of it.

// loopHandedness is the sign the face's own loop traversal implies for S_u×S_v: +1 when that normal is
// the outward one, −1 when the outward normal is its negative.
//
// It asks the one question the definition is made of. A boundary is traversed with the material on its
// LEFT as seen from outside — so step a hair to the left of a boundary edge and ask the region whether
// that step landed IN the face. "To the left" is [quarterArcLeftOf]: a quarter of that edge's own ARC
// LENGTH, which is a quarter turn in (u, v) only where the chart is isotropic. Every sampled edge
// votes, so a single mis-stepped sample near a corner cannot decide it.
//
// It used to be a shoelace over the projected rings, and that needed a rule of its own: a band's two
// rims are open polylines in the covering space and each shoelaces to ZERO whichever way the band is
// oriented, so a plate's bore wall reported the handedness of its own opposite and integrated the bore
// as ADDED, 564.36 against a true 354.92 (#3506). The rims were reassembled into one circuit to fix it,
// and the complement flag negated the answer for an outerless face. Neither survives: the region
// answers "is this in the face" on its own (ADR-0063), and the traversal is read against it directly.
//
// A boundaryless face (a whole sphere/torus, its own closed body) has no boundary to walk; it returns
// +1 and leans on the volume sign.
func loopHandedness(f curvedFace, r trimRegion) float64 {
	if len(r.contours) == 0 {
		return 1
	}
	votes := 0
	for _, ring := range trimPolys(f, r.uPeriodic, r.vPeriodic) {
		votes += materialSideVotes(f.surface, ring, r)
	}
	if votes < 0 {
		return -1
	}
	return 1
}

// materialSideVotes counts, over one projected loop ring, the edges whose LEFT side holds the face's
// material less those whose right side does. The probe steps a quarter of the edge's own ARC LENGTH, so
// it scales with the sampling and reads no absolute distance; an edge whose two sides answer alike (both
// in, both out — a step across a thin neck) abstains.
func materialSideVotes(s geom.Surface, ring []math.Point2, r trimRegion) int {
	votes := 0
	for _, i := range ringVoteStations(len(ring)) {
		a, b := ring[i-1], ring[i]
		mid := a.TranslateBy(a.VectorTo(b).Scale(0.5))
		left, ok := quarterArcLeftOf(s, mid, a.VectorTo(b))
		if !ok {
			continue
		}
		switch inLeft, inRight := r.contains(mid.TranslateBy(left)), r.contains(mid.TranslateBy(left.Scale(-1))); {
		case inLeft && !inRight:
			votes++
		case inRight && !inLeft:
			votes--
		}
	}
	return votes
}

// quarterArcLeftOf is the (u, v) offset that steps a quarter of the segment's own ARC LENGTH to the left
// of its travel, seen along the chart normal S_u×S_v. ok=false for a degenerate segment, a chart with no
// local frame (a pole, where S_u vanishes and det is 0 EXACTLY), or a step the chart cannot hold as a
// local one (stepStaysOnOneBranch).
//
// A quarter turn in (u, v) — the offset this used to take — is "left" only on a metrically ISOTROPIC
// chart. On a cylinder u is an angle and v a length, so the same numeric step is R·du one way and dv the
// other: at a 30 µm bore radius a rim segment's du/4 ≈ 0.05 lands far outside a chart 6e-5 tall, every
// rim station abstained, and the seam stations decided the vote the wrong way — the drilled plate's bore
// integrated as ADDED at 1e-4 and 1e-3 scale (Oblikovati/Oblikovati#3512, the #1610 scale sweep). The
// direction is therefore taken in SPACE, as N × T, and mapped back through the first fundamental form,
// which is the chart-agnostic statement of the same question. On an orthonormal chart (a plane) E = G =
// 1, F = 0 and |N| = 1, so it reduces to the quarter turn it replaces.
func quarterArcLeftOf(s geom.Surface, at math.Point2, d math.Vector2) (math.Vector2, bool) {
	pu, pv := s.DerivativesAt(float64(at.X), float64(at.Y))
	tangent := pu.Scale(d.X).Add(pv.Scale(d.Y))
	e, f, g := pu.Dot(pu), pu.Dot(pv), pv.Dot(pv)
	det := float64(e*g - f*f)
	normal := pu.Cross(pv)
	scale := 4 * float64(normal.Length())
	if det <= 0 || scale == 0 || float64(tangent.Length()) == 0 {
		return math.Vector2{}, false
	}
	left := normal.Cross(tangent) // ⊥ to the travel, in the tangent plane, of length |N|·|T|
	du := float64(g*left.Dot(pu)-f*left.Dot(pv)) / det
	dv := float64(e*left.Dot(pv)-f*left.Dot(pu)) / det
	off := math.V2(math.Scalar(du/scale), math.Scalar(dv/scale))
	return off, stepStaysOnOneBranch(s, off)
}

// stepStaysOnOneBranch reports whether a (u, v) offset is still a LOCAL step: on a PERIODIC axis it has
// to be shorter than half a turn, or the two sides of the boundary land on the same branch — or on
// crossed ones — and the probe is reading a point somewhere else on the surface entirely. The chart's
// own period says how long that is; nothing is chosen.
//
// This is the conditioning gate the exact-zero frame guards above cannot be. det is 0 EXACTLY where the
// frame collapses (S_u vanishes at a sphere pole and a cone apex), so those stations are refused there;
// a station a HAIR off one has a tiny-but-positive det and maps a perfectly legitimate quarter-arc to an
// unbounded du, which is arithmetically right and geometrically useless. On a non-periodic axis an
// over-long step needs no gate — the chart is finite, the probe lands outside it, and both sides then
// answer alike, which every caller already reads as "measured nothing". Only a periodic axis can wrap
// round and answer confidently about the wrong place.
func stepStaysOnOneBranch(s geom.Surface, off math.Vector2) bool {
	uPer, vPer := surfacePeriodic(s)
	halfTurn := twoPi / 2
	return (!uPer || stdmath.Abs(float64(off.X)) < halfTurn) &&
		(!vPer || stdmath.Abs(float64(off.Y)) < halfTurn)
}

// ringVoteStations picks the edges that vote: every edge of a short ring, and materialSideStations
// spread evenly over a long one. A majority over a few dozen samples settles a sign as surely as one
// over thousands, and the ring of a sampled arrangement boundary is thousands.
func ringVoteStations(n int) []int {
	if n <= materialSideStations {
		out := make([]int, 0, n)
		for i := 1; i < n; i++ {
			out = append(out, i)
		}
		return out
	}
	out := make([]int, 0, materialSideStations)
	for k := range materialSideStations {
		out = append(out, 1+k*(n-1)/materialSideStations)
	}
	return out
}

// materialSideStations bounds the votes per ring. It is a sampling count, not a tolerance.
const materialSideStations = 32

// twoPi is one full turn, the period of every angular surface parameter in the kernel.
const twoPi = 2 * stdmath.Pi
