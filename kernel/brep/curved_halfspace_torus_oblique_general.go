// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// General oblique torus half-space cut (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375). A plane TILTED
// relative to the torus axis (C = m·axis neither 0 nor ±1) cuts a spiric section whose w(v) carries the
// extra C·r·sin v term, so it is asymmetric in v — but it is still single-valued, u(v) = Phi ± arccos w(v).
// This handles the common single-OVAL tilt (one bite off the tube): the section is one closed oval and the
// kept solid is the small contractible CAP or its genus-1 COMPLEMENT, reusing the axis-parallel builders and
// meshers with the oval's v-range found from the closed-form w(v) = ±1 crossings. Two oblique ovals, the
// figure-eight pinch, and a tilt enclosing more than half the torus still demote to CSG.

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

// torusObliqueOval reports whether plane makes a single small oval bite we can build exactly: genuinely
// tilted (cylinderAxisTol < |C| so it is not the axis-parallel case, |C| < 1 so it is not perpendicular),
// one oval, and the BOUNDED disk under half the torus (so it is the small cap and its complement the large
// genus-1 region — keeping the meshers well-conditioned). A tilt enclosing a larger disk defers to CSG.
func torusObliqueOval(t geom.Torus, plane geom.Plane) bool {
	_, m, k, c := geom.TorusSectionCoeffs(t, plane)
	if stdmath.Abs(c) <= cylinderAxisTol || stdmath.Abs(c) >= 1-cylinderAxisTol || m <= cylinderAxisTol {
		return false
	}
	v0, v1, pinch, ok := torusObliqueOvalRange(t, m, k, c)
	return ok && ovalDiskArea(t, m, k, c, v0, v1, pinch) < 2*stdmath.Pi*stdmath.Pi
}

// ovalDiskArea returns the (u,v) area of the disk the oval bounds — the lens around the pinch, whose
// half-width is arccos(pinch·w(v)) (arccos w around Phi, or π−arccos w around Phi+π). ∫ 2·that over [v0,v1].
func ovalDiskArea(t geom.Torus, m, k, c, v0, v1, pinch float64) float64 {
	const n = 200
	var area float64
	for i := 0; i < n; i++ {
		v := v0 + (v1-v0)*(float64(i)+0.5)/n
		area += 2 * stdmath.Acos(clampUnitF(pinch*torusW(t, m, k, c, v))) * (v1 - v0) / n
	}
	return area
}

// torusObliqueOvalHalfSpace keeps the cap or the genus-1 complement of the single oval a tilted plane bites
// from the torus. The kept side is decided by the sign of g at the disk's interior (u = the pinch, mid-v):
// g≤0 there keeps the contractible cap (oval as the torus face's outer loop), g>0 keeps the complement.
func torusObliqueOvalHalfSpace(t geom.Torus, plane geom.Plane) (*topo.Body, error) {
	phi, m, k, c := geom.TorusSectionCoeffs(t, plane)
	v0, v1, pinch, _ := torusObliqueOvalRange(t, m, k, c)
	vMid := (v0 + v1) / 2
	gDisk := (t.MajorRadius+t.MinorRadius*stdmath.Cos(vMid))*m*pinch + c*t.MinorRadius*stdmath.Sin(vMid) - k
	return buildTorusOvalSolid(t, phi, m, k, c, v0, v1, unit(plane.Normal()), plane, gDisk > 0)
}
