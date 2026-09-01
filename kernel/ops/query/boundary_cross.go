// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// BoundariesCross reports whether the boundary of body a crosses the boundary of body b, analytically
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
func BoundariesCross(a, b *topo.Body) bool {
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
	overlap := probe.BoxOverlap(boxA, boxB)
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
	lo, hi, ok := probe.SampleRange(c, overlap)
	if !ok {
		return true // an unbracketable intersection curve is treated as a crossing → demote conservatively
	}
	if hi <= lo {
		return false
	}
	for i := 1; i < probe.CurveTrimSamples; i++ {
		p := c.PointAt(lo + (hi-lo)*float64(i)/probe.CurveTrimSamples)
		if brep.PointInFaceTrim(fa, p) && brep.PointInFaceTrim(fb, p) {
			return true
		}
	}
	return false
}
