// SPDX-License-Identifier: GPL-2.0-only

// Package ops — n-valent runout solver. PURE: imports only geom + math (no topo/diag). It turns an
// endCornerFan into a runoutSpread (per-far-face arc pieces + per-far-edge split points) or an
// error (the n-valent generalisation of the #1800 over-radius reject).
package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// runoutSpread is the solved n-valent runout: an arc piece per far face the cap actually touches,
// plus the single split point on every far edge the fillet cylinder crosses. Tasks 4-5 populate
// pieces/splits; this task only declares the shape they land in.
type runoutSpread struct {
	pieces map[uint64]cornerPiece // far-face id -> the arc piece it carries (absent = no arc)
	splits map[uint64]math.Point3 // far-edge id -> its single split point
}

// cornerPiece is the arc-fit of one far face's elliptical cap section, from its A-side tangency to
// its B-side tangency. A nil curve means Task 5 found the piece degenerates to a straight segment.
type cornerPiece struct {
	curve     geom.Curve3 // arc-fit of the elliptical section (nil ⇒ straight)
	tIn, tOut math.Point3
}

// splitOnFarEdge solves d²(x, axis) = r² for x = from + t·(to-from), returning the crossing
// nearest fe.from (the fan apex) among the roots t ∈ (0,1) — it does not verify the crossing is
// singular. d²(x,ℓ) = |x-c|² - ((x-c)·û)². The quadratic in t is A t² + 2B t + C with
// A = |w|² - (w·û)², B = (u0·w) - (u0·û)(w·û), C = |u0|² - (u0·û)² - r², where u0 = from-center,
// w = to-from, û = normalized axis. Returns ok=false if no root lies in (0,1) — the far edge
// doesn't graze or miss the fillet tube.
func splitOnFarEdge(fan endCornerFan, fe fanEdge) (math.Point3, bool) {
	uhat := unit(fan.axis)
	u0 := fan.center.VectorTo(fe.from)
	w := fe.from.VectorTo(fe.to)
	wu, u0u := w.Dot(uhat), u0.Dot(uhat)
	a := w.LengthSquared() - wu*wu
	b := u0.Dot(w) - u0u*wu
	c := u0.LengthSquared() - u0u*u0u - fan.radius*fan.radius
	t, ok := smallestRootIn01(a, b, c)
	if !ok {
		return math.Point3{}, false
	}
	return fe.from.TranslateBy(w.Scale(t)), true
}

// smallestRootIn01 returns the smallest real root of A t² + 2B t + C = 0 lying strictly in (0,1),
// with the linear fallback when |A| is tiny (axis parallel to the edge, so the quadratic term
// vanishes). ok=false if no root lies in that range.
func smallestRootIn01(a, b, c float64) (float64, bool) {
	const eps = 1e-12
	if stdmath.Abs(a) < eps {
		if stdmath.Abs(b) < eps {
			return 0, false
		}
		t := -c / (2 * b)
		return t, t > eps && t < 1-eps
	}
	disc := b*b - a*c
	if disc < 0 {
		return 0, false
	}
	s := stdmath.Sqrt(disc)
	for _, t := range []float64{(-b - s) / a, (-b + s) / a} {
		if t > eps && t < 1-eps {
			return t, true
		}
	}
	return 0, false
}

// solveRunoutSpread turns a fan into the per-face arc pieces + per-far-edge split points, or an
// error on a validity-certificate failure (Task 5). Membership is three-tier: a far face bounded by
// two crossings gets an arc; one only touched by a neighbour's split gets a split-pullback; an
// untouched face is omitted (its vertex survives). This slice implements the arc tier only — every
// fan face gets a piece — leaving boundaryPoint as the seam Task 6 uses to add the other two tiers.
// Every interior far edge yields exactly one split shared by its two faces — the weld-twice
// invariant, asserted by TestSolveRunoutSpreadChainCloses.
func solveRunoutSpread(fan endCornerFan) (runoutSpread, error) {
	sp := runoutSpread{pieces: map[uint64]cornerPiece{}, splits: map[uint64]math.Point3{}}
	for _, fe := range fan.farEdges {
		p, ok := splitOnFarEdge(fan, fe)
		if !ok {
			return runoutSpread{}, filletRunoutError(fan, "no single crossing on far edge", fe.edge)
		}
		sp.splits[fe.edge] = p
	}
	for i, ff := range fan.fan {
		tIn := boundaryPoint(fan, sp, ff.entryEdge, i == 0, fan.ta)
		tOut := boundaryPoint(fan, sp, ff.exitEdge, i == len(fan.fan)-1, fan.tb)
		sp.pieces[ff.face] = cornerPiece{curve: nil, tIn: tIn, tOut: tOut} // curve filled in Task 5
	}
	return sp, nil
}

// boundaryPoint resolves one end of a far face's piece: the rail point (ta or tb) at the flank
// (sentinel edge==0), else the split shared with the adjacent far face on the bounding far edge —
// the read that makes the weld-twice invariant hold (both neighbours read the same sp.splits entry).
func boundaryPoint(fan endCornerFan, sp runoutSpread, edge uint64, isFlank bool, rail math.Point3) math.Point3 {
	if isFlank && edge == 0 {
		return rail
	}
	return sp.splits[edge]
}

// filletRunoutError reports an n-valent runout certificate failure with the offending fillet edge,
// vertex valence, apex location, and the far edge that failed, plus the standard remediation — the
// generalisation of the #1800 over-radius reject to N>3 corners. Task 5/7 add more certificate
// checks that funnel through this constructor.
func filletRunoutError(fan endCornerFan, reason string, edge uint64) error {
	return fmt.Errorf("fillet: cannot round edge %d at a %d-valent runout vertex %v — %s (edge %d); reduce the radius or fillet the neighbours first",
		fan.filletEdge, len(fan.fan)+2, fan.apex, reason, edge)
}
