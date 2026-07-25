// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// runoutSeamIters bounds the seam bisection. The bracket halves every step, so 200 steps drives the
// interval to machine precision on any model scale — the loop exits on the relative-width test long
// before, and the cap only guarantees termination on a pathological φ.
const runoutSeamIters = 200

// seamStation solves the band boundary between the SURF-RST flank and the RST-RST central band:
//
//	φ(s) = 𝔠_B( F_B(s) ) = 0,   F_B(s) = the surf-rst tangency contact on hostB at station s
//
// i.e. the flank's contact locus on hostB reaches the INNER boss's footprint conic. This is NOT where
// the inner footprint crosses the PLAIN fillet's contact line (what the tiling used before) — measured
// against DRAWEXE the two differ by 32% on S1 (±4.4721 vs OCCT's ±3.38093) and 56% on S4 (±7.141 vs
// ±4.56723). At the root the rst-rst solve returns the SAME centre as the surf-rst solve, so the two
// bands join C¹ (verified on S1: both give y_c = −4 exactly, i.e. distance r from hostB).
//
// The bracket runs from the band midplane (φ<0, inside the inner footprint) to the outer cut station
// (φ>0, outside it). A bracket that does not straddle is an honest reject — this milestone tiles the
// S1 shape only and must not invent a seam for anything else.
func seamStation(env runoutEnvelope, hostA, hostB geom.Plane, outer, inner crossingBoss,
	mid, cut, weld float64) (float64, bool) {
	phi := seamResidual(env, hostA, hostB, outer, inner, weld)
	loVal, ok0 := phi(mid)
	hiVal, ok1 := phi(cut)
	if !ok0 || !ok1 || loVal >= 0 || hiVal <= 0 {
		return 0, false
	}
	lo, hi := mid, cut
	for i := 0; i < runoutSeamIters && stdmath.Abs(hi-lo) > 1e-13*(stdmath.Abs(cut)+stdmath.Abs(mid)+1); i++ {
		m := 0.5 * (lo + hi)
		v, ok := phi(m)
		if !ok {
			return 0, false
		}
		if v > 0 {
			hi = m
		} else {
			lo = m
		}
	}
	return 0.5 * (lo + hi), true
}

// seamResidual is φ(s): the inner boss footprint's implicit value at the surf-rst contact on hostB.
// Negative inside the footprint, positive outside. It closes over the whole surf-rst solve, so any
// station where the run-out ball does not exist reports ok=false and the bisection declines.
func seamResidual(env runoutEnvelope, hostA, hostB geom.Plane, outer, inner crossingBoss,
	weld float64) func(float64) (float64, bool) {
	return func(s float64) (float64, bool) {
		q, ok := footprintPointAtStation(outer, env.cyl, s)
		if !ok {
			return 0, false
		}
		c, ok := env.surfRstCentre(hostB, hostA, s, q, weld)
		if !ok {
			return 0, false
		}
		return footprintMembership(inner, projectOntoPlane(c, hostB))
	}
}

// footprintMembership is the boss footprint conic's implicit value at p, normalised so it is negative
// strictly inside the footprint, zero on it and positive outside — the only property the seam
// bisection uses. It covers both footprint kinds a setback boss can carry: a circle/arc (via
// footprintConic) and the oblique elliptical-cylinder boss's geom.EllipseFull (T7).
func footprintMembership(boss crossingBoss, p math.Point3) (float64, bool) {
	if e, ok := boss.footEdge.Geometry().(geom.EllipseFull); ok {
		d := e.Center.VectorTo(p)
		minorDir := e.Normal.Cross(e.MajorAxis)
		u := float64(d.Dot(e.MajorAxis.AsVector())) / e.MajorRadius
		v := float64(d.Dot(minorDir)) / e.MinorRadius
		return u*u + v*v - 1, true
	}
	c, r, ok := footprintConic(boss.footEdge)
	if !ok {
		return 0, false
	}
	return float64(p.DistanceSquaredTo(c)) - r*r, true
}
