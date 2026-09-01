// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// B-spline loop-edge helpers behind the wave-G additive arms in
// fillet_curved_retrim_loop.go (segParam / subSeg / endSegFromUse): the retained span of a
// B-spline edge is the parent's own exact sub-curve, oriented from→to and with its
// terminal control points snapped onto the caller's split points (which lie ON the curve
// within the split tolerance — the snap is a same-tolerance-class end adjustment that
// makes the rebuilt loop weld exactly).

// bsplineSubSeg is the exact sub-curve of a B-spline loop edge between two on-curve split
// points, oriented from→to. ok=false when either point does not invert onto the parent
// (the caller then ships its base chord — do-no-harm).
func bsplineSubSeg(bs geom.BSplineCurve, from, to math.Point3) (geom.BSplineCurve, bool) {
	t0, _ := geom.CurveParamAtPoint3(bs, from)
	t1, _ := geom.CurveParamAtPoint3(bs, to)
	if t0 == t1 {
		return geom.BSplineCurve{}, false
	}
	sub, err := orientedSubSpan(bs, t0, t1)
	if err != nil {
		return geom.BSplineCurve{}, false
	}
	snapped, err := snapCurveEnds(sub, from, to)
	if err != nil {
		return geom.BSplineCurve{}, false
	}
	return snapped, true
}

// midDomainParam is the curve's mid-domain parameter (endSeg.mid is a display/refit
// anchor, not a weld point, so mid-domain is the right generic choice).
func midDomainParam(c geom.Curve3) float64 {
	lo, hi := c.Domain()
	return (lo + hi) / 2
}

// bsplineSurvivorConcrete is a loop edge's B-spline curve oriented WITH the use, as a
// concrete geom.BSplineCurve (an exact control-net reversal, never the opaque reversed
// wrapper) so segParam/subSeg can split it exactly. ok=false for any other edge kind.
func bsplineSurvivorConcrete(u *topo.EdgeUse) (geom.BSplineCurve, bool) {
	bs, ok := u.Edge().Geometry().(geom.BSplineCurve)
	if !ok {
		return geom.BSplineCurve{}, false
	}
	if !u.Reversed() || u.Edge().StartVertex() == u.Edge().EndVertex() {
		return bs, true
	}
	rev, err := geom.ReverseBSplineCurve(bs)
	if err != nil {
		return geom.BSplineCurve{}, false
	}
	return rev, true
}

// wgLoopFromSegs is loopFromSegs with the segments' SOURCE-EDGE identity carried into
// filletLoop.srcE (srcV stays 0 — points still weld by coordinate, so retrimmed and
// untouched faces merge). The identity is what keeps two coincident survivor edges
// sharing both endpoints (a prism cap's bezier + its closing chord) on SEPARATE edge
// classes through the weld (#1600 method C) — with srcE dropped they collapsed into one
// 4-face non-manifold edge (the G5 signature).
func wgLoopFromSegs(segs []endSeg) filletLoop {
	fl := loopFromSegs(segs)
	fl.srcV = make([]uint64, len(segs))
	fl.srcE = make([]uint64, len(segs))
	for i, s := range segs {
		fl.srcE[i] = s.srcEdge
	}
	return fl
}

// wgPassthroughFace is passthroughFace with source-edge identities kept (points remain
// coordinate-welded, id-0).
func wgPassthroughFace(f *topo.Face) filletFace {
	loops := make([]filletLoop, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		loops = append(loops, wgLoopFromSegs(segsFromLoop(l)))
	}
	return filletFace{surface: f.Geometry(), loops: loops, parent: f.Lineage()}
}
