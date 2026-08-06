// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A coaxial ball-and-rod boolean is a ONE-DIMENSIONAL problem, and this file is that dimension
// (Oblikovati#2036, #2061). Both operands are surfaces of revolution about the SAME axis, so a point's
// membership in the other solid depends on nothing but its axial station s (measured from the ball
// centre along `out`) — or, on a rod cap, on its radius. Every face of every result is therefore one
// maximal run of constant membership, and finding those runs is all the "classification" a boolean of
// this family needs. No (u,v) arrangement, no SSI tracing: two scalar predicates and a sort.
//
//	a BALL-surface point at station s is inside the rod  ⟺  |s| > d  and  sLo ≤ s ≤ sHi
//	a ROD-WALL point at station s is inside the ball      ⟺  |s| < d
//	a ROD-CAP point at station s*, radius r, is inside    ⟺  r < ρ(s*),  ρ(s) = √(R²−s²)
//
// where d = √(R²−r_c²) is OCCT's seam offset. The first two use d because the ball's own radius at
// station s is ρ(s), and ρ(s) < r_c is exactly |s| > d.

// coaxialSpan is one maximal run of a scalar parameter over which membership does not change. The
// parameter is the axial STATION for the ball's surface and the rod's wall, and the RADIUS on a rod cap
// — one type, because a run is a run.
type coaxialSpan struct {
	lo, hi float64
	inside bool // the run lies inside the OTHER solid
}

// ballRadiusAt is the ball's cross-section radius at axial station s — 0 at or beyond the poles.
func (r coaxialRod) ballRadiusAt(s float64) float64 {
	if rr := r.sph.Radius*r.sph.Radius - s*s; rr > 0 {
		return stdmath.Sqrt(rr)
	}
	return 0
}

// ballSpans splits the ball's surface, parameterised by station from pole to pole, wherever its
// membership in the rod changes: at the two seam stations ±d, and at either rod cap that lands between
// the poles.
func (r coaxialRod) ballSpans() []coaxialSpan {
	R := r.sph.Radius
	cuts := []float64{-R, -r.seamOffset, r.seamOffset, R}
	for _, s := range []float64{r.sLo, r.sHi} {
		if s > -R && s < R {
			cuts = append(cuts, s)
		}
	}
	return r.runsOver(cuts, -R, R, func(s float64) bool {
		return stdmath.Abs(s) > r.seamOffset && s >= r.sLo && s <= r.sHi
	})
}

// wallSpans splits the rod's cylindrical side over its own extent, wherever it crosses the ball's
// surface — the two seam stations.
func (r coaxialRod) wallSpans() []coaxialSpan {
	return r.runsOver([]float64{r.sLo, r.sHi, -r.seamOffset, r.seamOffset}, r.sLo, r.sHi,
		func(s float64) bool { return stdmath.Abs(s) < r.seamOffset })
}

// capSpans splits one rod cap radially at the circle where the ball's surface crosses it.
func (r coaxialRod) capSpans(station float64) []coaxialSpan {
	rc := r.cyl.Radius
	return r.runsOver([]float64{0, rc, r.ballRadiusAt(station)}, 0, rc,
		func(rad float64) bool { return rad < r.ballRadiusAt(station) })
}

// runsOver sorts the cut values, keeps those strictly inside (lo, hi), and returns the maximal runs
// over which `inside` (sampled at each run's midpoint) does not change. Adjacent runs of equal verdict
// merge, so a cut that changes nothing — a rod cap outside the ball, a seam outside the rod — leaves no
// spurious face boundary.
func (r coaxialRod) runsOver(cuts []float64, lo, hi float64, inside func(float64) bool) []coaxialSpan {
	stations := r.interiorCuts(cuts, lo, hi)
	var out []coaxialSpan
	prev := lo
	for _, s := range append(stations, hi) {
		at := inside((prev + s) / 2)
		if n := len(out); n > 0 && out[n-1].inside == at {
			out[n-1].hi = s
		} else {
			out = append(out, coaxialSpan{lo: prev, hi: s, inside: at})
		}
		prev = s
	}
	return out
}

// interiorCuts returns the sorted, deduplicated cut values strictly inside (lo, hi). "Strictly" is
// model-relative: a cut within the ball's own resolution of an endpoint or of another cut would carve a
// sliver face, so it is dropped rather than emitted (ADR-0042).
func (r coaxialRod) interiorCuts(cuts []float64, lo, hi float64) []float64 {
	tol := r.stationTol()
	var out []float64
	for _, c := range sortedFloats(cuts) {
		if c <= lo+tol || c >= hi-tol {
			continue
		}
		if n := len(out); n > 0 && c-out[n-1] <= tol {
			continue
		}
		out = append(out, c)
	}
	return out
}

// stationTol is the axial resolution of this pair: below it two stations are the same station, so no
// face may be thinner. It scales with the ball, so a µm ball and a metre ball split identically.
func (r coaxialRod) stationTol() float64 {
	return geom.ResolutionForSize(r.sph.Radius).Plane()
}

// sortedFloats returns the values in ascending order without mutating the input.
func sortedFloats(xs []float64) []float64 {
	out := append([]float64(nil), xs...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// stationPoint is the point on the axis at the given station.
func (r coaxialRod) stationPoint(s float64) math.Point3 {
	return r.sph.Center.TranslateBy(r.out.Scale(math.Scalar(s)))
}
