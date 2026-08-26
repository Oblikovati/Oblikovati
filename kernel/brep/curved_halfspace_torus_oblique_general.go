// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
)

// General oblique torus half-space cut (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375). A plane TILTED
// relative to the torus axis (C = m·axis neither 0 nor ±1) cuts a spiric section whose w(v) carries the
// extra C·r·sin v term, so it is asymmetric in v — but it is still single-valued, u(v) = Phi ± arccos w(v).
// This handles the common single-OVAL tilt (one bite off the tube): the section is one closed oval and the
// kept solid is the small contractible CAP or its genus-1 COMPLEMENT, reusing the axis-parallel builders and
// meshers with the oval's v-range found from the closed-form w(v) = ±1 crossings. A tilted cut through the
// hole makes TWO oblique ovals — [torusTwoObliqueOval] routes those through the two-oval band path. A tilt
// enclosing more than half the torus (the disk-larger-than-half case) still demotes to CSG.

// torusW evaluates the section function w(v) = (K − C·r·sin v) / (M·(R + r·cos v)).
func torusW(t geom.Torus, m, k, c, v float64) float64 {
	return (k - c*t.MinorRadius*stdmath.Sin(v)) / (m * (t.MajorRadius + t.MinorRadius*stdmath.Cos(v)))
}

// wUnitCrossings returns the tube angles v∈[0,2π) where |w(v)| = 1 (the section's branch pinches), from
// the closed forms K ∓ M·R = r·√(C²+M²)·sin(v ± δ), δ = atan2(M, C). Up to four (two per ±1 level).
func wUnitCrossings(t geom.Torus, m, k, c float64) []float64 {
	a := stdmath.Hypot(c, m)
	delta := stdmath.Atan2(m, c)
	var out []float64
	addLevel := func(s, phase float64) {
		if stdmath.Abs(s) > 1 {
			return
		}
		base := stdmath.Asin(s)
		for _, x := range []float64{base + phase, stdmath.Pi - base + phase} {
			out = append(out, normalizeAngle(x))
		}
	}
	addLevel((k-m*t.MajorRadius)/(t.MinorRadius*a), -delta) // w = +1
	addLevel((k+m*t.MajorRadius)/(t.MinorRadius*a), +delta) // w = −1
	return out
}

// normalizeAngle folds an angle into [0, 2π).
func normalizeAngle(x float64) float64 {
	x = stdmath.Mod(x, 2*stdmath.Pi)
	if x < 0 {
		x += 2 * stdmath.Pi
	}
	return x
}

// torusObliqueOvalRange returns the single-oval tube-angle interval [v0,v1] (v0 may be negative so the
// sweep is continuous across the seam) and the pinch sign — +1 when the branches meet at u=Phi (the
// crossings are at w=+1) or −1 when they meet at u=Phi+π (w=−1), which is where the bounded disk is centred.
// ok=false unless the section is exactly one oval — two pinch crossings bounding one valid (|w|≤1) interval
// (zero or four crossings are a clearing/two-oval/figure-eight topology handled elsewhere or deferred).
func torusObliqueOvalRange(t geom.Torus, m, k, c float64) (v0, v1, pinch float64, ok bool) {
	cross := wUnitCrossings(t, m, k, c)
	if len(cross) != 2 {
		return 0, 0, 0, false
	}
	sort.Float64s(cross)
	a, b := cross[0], cross[1]
	v0, v1 = a, b
	if stdmath.Abs(torusW(t, m, k, c, (a+b)/2)) > 1 {
		v0, v1 = b, a+2*stdmath.Pi // the valid stretch wraps the seam
	}
	pinch = 1
	if torusW(t, m, k, c, v0) < 0 {
		pinch = -1 // the branches meet at Phi+π, so the disk is centred there
	}
	return v0, v1, pinch, true
}

// figureEightWrapTol is the tube-angle arc over which a single oval's section may be ABSENT before it counts
// as a near-full-wrap figure-eight (the oblique analogue of the axis-parallel tangent cut) and routes to the
// two-oval band path instead. A single oval whose section exists over all but this sliver of the tube has
// wrapped nearly the whole tube; its genus-1 complement is then a thin strip the chart can't mesh, so the
// band loft (which meshes the sliver as the band's zero-width pinch) handles it robustly instead.
//
// The near-full-wrap test SAMPLES where the section exists (|w(v)|≤1) rather than reading the analytic
// w(v)=±1 crossings: at the exact transition the two crossings collide into a double root whose asin is
// platform-sensitive, so the analytic span flickers either side of the threshold across platforms.
const figureEightWrapTol = 0.15

// torusSectionAbsentArc returns the tube-angle arc length over which the spiric section does NOT exist
// (|w(v)| > 1), sampled around the tube. Zero means the section wraps the whole tube (two ovals); a small
// value means a near-full-wrap single oval; a large value a clear single-oval bite.
func torusSectionAbsentArc(t geom.Torus, m, k, c float64) float64 {
	const n = 720
	absent := 0
	for i := range n {
		if stdmath.Abs(torusW(t, m, k, c, 2*stdmath.Pi*float64(i)/n)) > 1 {
			absent++
		}
	}
	return 2 * stdmath.Pi * float64(absent) / n
}
