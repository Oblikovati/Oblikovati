// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// boundariesCross reports whether the boundary of body a crosses the boundary of body b, analytically
// (M48/C3 #3423). It is the missing half of containment classification (#1315): all-vertices-inside does
// NOT imply containment for non-convex operands — an edge or face of the inner body can pass back out
// through the outer body BETWEEN the inner's vertices — so a boundary crossing must demote the pair to
// the general face-splitting boolean instead of the (wrong) contains-fast-path.
//
// The test reads no tessellation — OCCT decides boundary interference the same way, by analytic
// face-face intersection (IntTools_FaceFace). Each face pair whose range boxes overlap is intersected
// with geom.SurfaceIntersect, and a crossing is confirmed only where the intersection curve lies inside
// BOTH trims (facesCross), which catches an interior-interior crossing an edge-pierce test would miss.
// It runs only after allVerticesInside already passed (the candidate-containment case), so it never
// burdens the common intersecting path, which exits classify earlier.
func boundariesCross(a, b *topo.Body) bool {
	res := geom.ResolutionForBox(a.RangeBox().Union(b.RangeBox()))
	bFaces := b.Faces()
	for _, fa := range a.Faces() {
		boxA := fa.RangeBox()
		for _, fb := range bFaces {
			if boxA.Intersects(fb.RangeBox()) && facesCross(fa, fb, res) {
				return true
			}
		}
	}
	return false
}

// facesCross reports whether the trimmed faces fa and fb genuinely cross: their surfaces intersect along
// a curve that lies inside BOTH trims. An empty intersection (parallel or tangent surfaces — the closed
// form returns no curve for a known non-crossing touch) is not a crossing. A pair the analytic
// intersector cannot resolve (handled=false) is treated CONSERVATIVELY as crossing, demoting the
// operands to the general face-splitting boolean — always correct — rather than risking a missed
// crossing and a silently wrong solid.
func facesCross(fa, fb *topo.Face, res geom.Resolution) bool {
	boxA, boxB := fa.RangeBox(), fb.RangeBox()
	curves, handled := geom.SurfaceIntersect(fa.Geometry(), fb.Geometry(), boxA.Union(boxB), res)
	if !handled {
		return true
	}
	overlap := boxOverlap(boxA, boxB)
	for _, c := range curves {
		if curveEntersBothTrims(c, overlap, fa, fb) {
			return true
		}
	}
	return false
}

// curveEntersBothTrims reports whether any interior sample of the intersection curve c lies inside both
// faces' trim regions. Sampling is bounded to `overlap` — the intersection of the two faces' range
// boxes, the only region an in-both-trims crossing can occupy — because the closed-form intersector
// returns an UNBOUNDED curve (a plane∩plane line has parameter domain [-Inf,+Inf]); sampling its raw
// domain would evaluate PointAt(NaN). Each sample is on both surfaces by construction, satisfying
// PointInFaceTrim's on-surface precondition; brep.PointInFaceTrim then keeps only a genuine in-trim hit.
func curveEntersBothTrims(c geom.Curve3, overlap math.Box, fa, fb *topo.Face) bool {
	lo, hi, ok := sampleRange(c, overlap)
	if !ok {
		return true // an unbracketable intersection curve is treated as a crossing → demote conservatively
	}
	if hi <= lo {
		return false
	}
	for i := 1; i < curveTrimSamples; i++ {
		p := c.PointAt(lo + (hi-lo)*float64(i)/curveTrimSamples)
		if brep.PointInFaceTrim(fa, p) && brep.PointInFaceTrim(fb, p) {
			return true
		}
	}
	return false
}

// sampleRange returns a finite parameter interval of curve c to sample, bounded to the box overlap. A
// bounded intersection curve (a closed loop from two curved faces) uses its own finite domain (ok=true).
// An UNBOUNDED curve — the infinite line a closed-form plane∩plane returns, domain [-Inf,+Inf] — is
// bounded by projecting overlap's eight corners onto the line: the line is affine, so the parameter of a
// point q is (q−P0)·d / |d|² with d = P(1)−P(0), and the corner projections bracket the box. An unbounded
// curve that is NOT a straight line (a cone section — parabola/hyperbola) cannot be bracketed this way,
// so ok=false and the caller conservatively treats the pair as crossing rather than risk a missed hit.
func sampleRange(c geom.Curve3, overlap math.Box) (lo, hi float64, ok bool) {
	dlo, dhi := c.Domain()
	if !stdmath.IsInf(dlo, 0) && !stdmath.IsInf(dhi, 0) {
		return dlo, dhi, true
	}
	p0, p1, pmid := c.PointAt(0), c.PointAt(1), c.PointAt(0.5)
	d := p0.VectorTo(p1)
	dd := d.Dot(d)
	if dd == 0 || !isColinearMidpoint(p0, p1, pmid) {
		return 0, 0, false
	}
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	for _, q := range overlap.Corners() {
		t := p0.VectorTo(q).Dot(d) / dd
		lo, hi = stdmath.Min(lo, t), stdmath.Max(hi, t)
	}
	return lo, hi, true
}

// isColinearMidpoint reports whether c's midpoint sample lies on the chord P0→P1 — i.e. the curve is a
// straight line over [0,1]. An intersection conic that is colinear at three parameters is degenerate to
// a line, so this cleanly separates the affine plane∩plane line (bracketable by corner projection) from
// a curved unbounded section that is not.
func isColinearMidpoint(p0, p1, pmid math.Point3) bool {
	chord := p0.VectorTo(p1)
	mid := math.P3((p0.X+p1.X)/2, (p0.Y+p1.Y)/2, (p0.Z+p1.Z)/2)
	return pmid.VectorTo(mid).Length() <= colinearRelTol*chord.Length()
}

// colinearRelTol is the chord-relative straightness bound for an unbounded section curve: a true line's
// midpoint deviates only by float rounding, far below this, while any conic bows well above it.
const colinearRelTol = 1e-9 // tol:relative — dimensionless straightness fraction of the chord length

// boxOverlap returns the intersection box of a and b (their per-axis overlap). Its callers reach it
// only after Box.Intersects reported true, so every axis has Min ≤ Max.
func boxOverlap(a, b math.Box) math.Box {
	return math.Box{
		Min: math.P3(stdmath.Max(a.Min.X, b.Min.X), stdmath.Max(a.Min.Y, b.Min.Y), stdmath.Max(a.Min.Z, b.Min.Z)),
		Max: math.P3(stdmath.Min(a.Max.X, b.Max.X), stdmath.Min(a.Max.Y, b.Max.Y), stdmath.Min(a.Max.Z, b.Max.Z)),
	}
}

// curveTrimSamples is the interior-sample count along an intersection curve (a count, not a tolerance):
// enough that a short in-trim crossing arc of a curve clipped to the whole shared face box is not
// stepped over.
const curveTrimSamples = 16
