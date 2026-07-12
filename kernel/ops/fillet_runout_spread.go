// SPDX-License-Identifier: GPL-2.0-only

// Package ops — n-valent runout solver. PURE: imports only geom + math (no topo/diag). It turns an
// endCornerFan into a runoutSpread (per-far-face arc pieces + per-far-edge split points) or an
// error (the n-valent generalisation of the #1800 over-radius reject).
package ops

import (
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
