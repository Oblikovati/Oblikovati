// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Chorded torus-band rim tracer (M4). The M3 setback-runout rebuild reproduces a torus wall's
// footprint rim as a CHAIN of chord/mixed edges (≈30 geom.LineSegment sub-edges), not the single
// full-circle edge bandRingsAndSeam's single-edge recognition accepts — so the band is dropped to
// fullDomainGridMesh and meshed as the whole donut (spike §CRITICAL, .superpowers/sdd/m4-spike.md).
//
// chainBoundaryRings recovers the band curve-type-agnostically: it isolates the meridian tube seam by
// its non-zero tube-parameter (v) span (the robust seam rule — both rings are iso-v; §b/§c), chains the
// remaining edges head-to-tail by welded endpoints into two closed point-rings (reusing traceClosedRings
// — the same head-to-tail chaining the closed-in-u nurbs tracer uses), and GATES the pair on around-u
// congruence so a malformed out-and-back slit rim is honest-rejected (ok=false) rather than lofted into
// garbage. It is a strict, additive fallback: plain single-edge rim fillets never reach it.

// bandMonotoneRatio is the minimum |U|/V for a ring to count as monotone around the band (no
// out-and-back): a clean full loop has ratio 1, the malformed 118° doubled slit ≈0. It is a
// dimensionless radians ratio — scale-free by construction, NOT a model-relative length tolerance.
const bandMonotoneRatio = 0.99

// meridianTolK, congruenceTolK are chord-to-angle factors (ε_θ = k·weld/r) for the two model-relative
// angular tolerances (ADR-0042): the seam v-span discriminator and the around-u range congruence.
const (
	meridianTolK   = 4.0
	congruenceTolK = 8.0
)

// chainBoundaryRings meshes a torus band whose footprint rim is a chorded/mixed-edge chain (the Task-1
// setback rebuild) rather than one full-circle edge. It returns the same shape as bandRingsAndSeam —
// two boundary point-rings, the seam's sample count and midpoint — with ok=true only when exactly two
// congruent rings survive the tessellation congruence gate and the seam carries ≥2 points.
func chainBoundaryRings(f *topo.Face, q Quality) (rings [][]math.Point3, seamN int, seamMid math.Point3, ok bool) {
	tor, isTorus := f.Geometry().(geom.Torus)
	if !isTorus {
		return nil, 0, math.Point3{}, false
	}
	two, seamPts, chained := torusBandRingsAndSeam(f, tor, q)
	if !chained || !ringsCongruent(tor, two[0], two[1]) {
		return nil, 0, math.Point3{}, false
	}
	return two, len(seamPts), seamPts[len(seamPts)/2], true
}

// torusBandRingsAndSeam isolates the meridian seam by its tube v-span, chains the remaining boundary
// edges into exactly two closed point-rings (traceClosedRings — its "removing the seam leaves two
// closed cycles" is the topological cross-check the spike §c prescribes), and returns the seam's own
// tessellation. chained=false unless exactly one seam, two closed rings, and a ≥2-point seam survive.
func torusBandRingsAndSeam(f *topo.Face, tor geom.Torus, q Quality) (rings [][]math.Point3, seamPts []math.Point3, chained bool) {
	seam, ok := torusMeridianSeam(f, tor, q)
	if !ok {
		return nil, nil, false
	}
	rings, ok = traceClosedRings(orientedEdgePolylines(f, q, func(e *topo.Edge) bool { return e.ID() == seam.ID() }))
	if !ok || len(rings) != 2 {
		return nil, nil, false
	}
	seamPts = TessellateEdge(seam, q)
	return rings, seamPts, len(seamPts) >= 2
}

// torusMeridianSeam returns the boundary edge whose endpoints differ in the torus tube parameter v —
// the meridian seam bridging the two iso-v rings (spike §b: seam vSpan≈π/2, both rings vSpan≈0). It is
// unique on a clean rim-fillet band; ok=false unless exactly one such edge exists.
func torusMeridianSeam(f *topo.Face, tor geom.Torus, q Quality) (*topo.Edge, bool) {
	vTol := meridianTolK * faceBoundaryWeld(f, q) / tor.MinorRadius
	var seam *topo.Edge
	for _, e := range f.Edges() {
		if edgeTubeVSpan(tor, e) <= vTol {
			continue
		}
		if seam != nil {
			return nil, false // more than one meridian edge → not a clean two-ring band
		}
		seam = e
	}
	return seam, seam != nil
}

// edgeTubeVSpan is the absolute difference in torus tube parameter v between an edge's endpoints:
// ≈0 for an iso-v ring edge (both endpoints on one rim), ≈π/2 for the meridian seam.
func edgeTubeVSpan(tor geom.Torus, e *topo.Edge) float64 {
	_, v0 := tor.ParamAt(e.StartVertex().Point())
	_, v1 := tor.ParamAt(e.EndVertex().Point())
	return stdmath.Abs(wrapPi(v1 - v0))
}

// faceBoundaryWeld is the model-relative weld length (ADR-0042) of the face's whole boundary — the
// length scale the seam-v-span and range-congruence angular tolerances are derived from.
func faceBoundaryWeld(f *topo.Face, q Quality) float64 {
	var pts []math.Point3
	for _, e := range f.Edges() {
		pts = append(pts, TessellateEdge(e, q)...)
	}
	return ResolutionForPoints(pts).Weld()
}

// orientedEdgePolylines returns the face's edge-use polylines, each oriented to its use so consecutive
// ring edges chain head-to-tail, dropping every edge for which drop reports true. Shared by the
// closed-in-u nurbs tracer (drops the used-twice seam) and the torus-band tracer (drops the v-span seam).
func orientedEdgePolylines(f *topo.Face, q Quality, drop func(*topo.Edge) bool) [][]math.Point3 {
	var segs [][]math.Point3
	for _, l := range f.Loops() {
		for _, eu := range l.EdgeUses() {
			if drop(eu.Edge()) {
				continue
			}
			pts := discretizeEdge(eu.Edge(), q)
			if eu.Reversed() {
				pts = reverse3(pts)
			}
			if len(pts) >= 2 {
				segs = append(segs, pts)
			}
		}
	}
	return segs
}

// ringsCongruent gates the two chained rings on the tessellation congruence test (M4 derivation
// §"Tessellation congruence gate"): in the band's around-parameter u (the torus major sweep) each ring
// must advance MONOTONICALLY (|U|/V ≥ bandMonotoneRatio — no out-and-back) and cover an EQUAL u-range.
// The malformed 118° doubled slit has |U|/V≈0 (rejected); a clean full-circle rim has ratio 1, U=2π.
func ringsCongruent(tor geom.Torus, a, b []math.Point3) bool {
	ua, va := aroundAdvance(tor, a)
	ub, vb := aroundAdvance(tor, b)
	if !monotoneAround(ua, va) || !monotoneAround(ub, vb) {
		return false
	}
	tol := congruenceTolK * bothRingsWeld(a, b) / tor.MajorRadius
	return stdmath.Abs(stdmath.Abs(ua)-stdmath.Abs(ub)) <= tol
}

// aroundAdvance walks a ring's around-parameter u (the torus major sweep) in traversal order and
// returns the signed total advance U=Σ unwrap(Δu) and the unsigned variation V=Σ|Δu|, including the
// closing step back to the first point. A monotone full loop gives U=V=2π; an out-and-back slit U≈0.
func aroundAdvance(tor geom.Torus, ring []math.Point3) (signed, variation float64) {
	us := make([]float64, len(ring))
	for i, p := range ring {
		us[i], _ = tor.ParamAt(p)
	}
	for i := range us {
		d := wrapPi(us[(i+1)%len(us)] - us[i])
		signed += d
		variation += stdmath.Abs(d)
	}
	return signed, variation
}

// monotoneAround reports whether a ring's around-u advance is monotone (no out-and-back): |U|/V is
// near 1. A degenerate zero-variation ring (all points at one angle) is not a liftable rim.
func monotoneAround(signed, variation float64) bool {
	if variation == 0 {
		return false
	}
	return stdmath.Abs(signed)/variation >= bandMonotoneRatio
}

// bothRingsWeld is the model-relative weld length over both rings' points — the scale the range-
// congruence angular tolerance is derived from.
func bothRingsWeld(a, b []math.Point3) float64 {
	pts := make([]math.Point3, 0, len(a)+len(b))
	pts = append(pts, a...)
	pts = append(pts, b...)
	return ResolutionForPoints(pts).Weld()
}
