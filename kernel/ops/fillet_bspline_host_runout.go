// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"
	"os"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// OPEN-edge body builder for the B-spline-host canal: the prolonged band is trimmed at
// both capping planes (fillet_bspline_host_trim.go), the two host faces recede onto the
// band's rail spans, and the capping faces are bitten by the trim curves — the single-arm
// runout weld (fillet_curved_single_runout.go) rebuilt for a numeric band, including the
// double-bite capping face (G5-class: both ends terminate on ONE side wall, which the
// one-bite-per-face analytic path cannot express).

// bsplineHostRunoutBody welds one OPEN B-spline-host canal pick. Empty reason ⇒ the body
// is the weld; else the reason names the obstruction and the body is nil (do-no-harm).
func bsplineHostRunoutBody(body *topo.Body, ef edgeFillet, canal *bsplineHostCanal, res Resolution) (*topo.Body, string) {
	if canal.concave {
		return nil, "bspline-host runout: open CONCAVE B-spline-host edges are not yet supported (closed concave rims are)"
	}
	end0, end1, reason := bsplineHostResolveEnds(ef, canal, res)
	if reason != "" {
		return nil, reason
	}
	railA, railB, reason := bsplineHostRailSpans(canal, end0, end1)
	if reason != "" {
		return nil, reason
	}
	vLo := stdmath.Min(stdmath.Min(end0.vA, end0.vB), stdmath.Min(end1.vA, end1.vB))
	vHi := stdmath.Max(stdmath.Max(end0.vA, end0.vB), stdmath.Max(end1.vA, end1.vB))
	if e := bsplineHostRetainedEnvelopeError(canal, vLo, vHi, res); e > bsplineHostEnvelopeBound(res) {
		return nil, fmt.Sprintf("bspline-host runout: retained band envelope error %g over bound %g", e, bsplineHostEnvelopeBound(res))
	}
	faces, reason := bsplineHostRunoutFaces(body, ef, canal, railA, railB, end0, end1, res)
	if reason != "" {
		return nil, reason
	}
	b := assembleBody(faces)
	wgBsplineRunoutValidateDebug(b)
	return b, ""
}

// wgBsplineRunoutValidateDebug traces the assembled runout body's validity verdict under
// WG_BSPLINE_DEBUG=1 (temporary; the caller's Validate is the real gate).
func wgBsplineRunoutValidateDebug(b *topo.Body) {
	if os.Getenv("WG_BSPLINE_DEBUG") != "1" {
		return
	}
	rep := Validate(b)
	wgBsplineDebug(fmt.Sprintf("runout body: valid=%t closed=%t manifold=%t orient=%t euler=%t holes=%t solid=%t issues=%v",
		rep.Valid, rep.Closed, rep.Manifold, rep.OrientationOK, rep.EulerConsistent, rep.HolesContained, b.IsSolid(), rep.Issues), nil)
	for _, e := range b.Edges() {
		if n := len(e.Faces()); n != 2 {
			wgBsplineDebug(fmt.Sprintf("  edge %d uses=%d %v -> %v", e.ID(), n, e.StartVertex().Point(), e.EndVertex().Point()), nil)
		}
	}
}

// bsplineHostEnvelopeBound is the model-relative geometric bound every end-trim landing,
// rail span and fitted curve must meet — the same bound the band itself was refined to.
func bsplineHostEnvelopeBound(res Resolution) float64 { return bsplineHostEnvelopeCoef * res.Weld() }

// bsplineHostResolveEnds solves both end trims: capping plane, rail crossings, snapped
// landings and the fitted crossing curve, with end0 at the picked edge's START vertex.
func bsplineHostResolveEnds(ef edgeFillet, canal *bsplineHostCanal, res Resolution) (end0, end1 bsplineHostEndTrim, reason string) {
	vp := bsplineHostStationV(canal.stations)
	lowIsStart := bsplineHostLowSideIsStart(ef, canal)
	e0Idx, e1Idx := canal.plan.iEdge0, canal.plan.iEdge1
	if !lowIsStart {
		e0Idx, e1Idx = e1Idx, e0Idx
	}
	i0lo, i0hi := bsplineHostEndWindow(canal, e0Idx)
	i1lo, i1hi := bsplineHostEndWindow(canal, e1Idx)
	end0, reason = bsplineHostSolveEnd(ef, canal, ef.edge.StartVertex(), vp, i0lo, i0hi, res)
	if reason != "" {
		return end0, end1, reason
	}
	end1, reason = bsplineHostSolveEnd(ef, canal, ef.edge.EndVertex(), vp, i1lo, i1hi, res)
	return end0, end1, reason
}

// bsplineHostLowSideIsStart reports whether the band's LOW-v side anchors at the picked
// edge's start vertex (the walk-direction resolution may have reversed the march).
func bsplineHostLowSideIsStart(ef edgeFillet, canal *bsplineHostCanal) bool {
	sv := ef.edge.StartVertex().Point()
	lowAnchor := canal.stations[canal.plan.iEdge0].Anchor
	highAnchor := canal.stations[canal.plan.iEdge1].Anchor
	return lowAnchor.DistanceTo(sv) <= highAnchor.DistanceTo(sv)
}

// bsplineHostEndWindowArcSpan bounds the cap-crossing search around one edge end, as a
// multiple of r: an oblique cap's crossing sits within ~r·tanθ of the end on either side.
const bsplineHostEndWindowArcSpan = 1.5

// bsplineHostEndWindow is the station-index search window around one edge-end station:
// every station within bsplineHostEndWindowArcSpan·r of that end's arc coordinate.
func bsplineHostEndWindow(canal *bsplineHostCanal, endIdx int) (lo, hi int) {
	span := bsplineHostEndWindowArcSpan * canal.r
	at := canal.plan.arcs[endIdx]
	lo, hi = endIdx, endIdx
	for lo > 0 && stdmath.Abs(canal.plan.arcs[lo-1]-at) <= span {
		lo--
	}
	for hi < len(canal.plan.arcs)-1 && stdmath.Abs(canal.plan.arcs[hi+1]-at) <= span {
		hi++
	}
	return lo, hi
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// bsplineHostSolveEnd resolves one end: capping plane, both rail crossings (snapped onto
// the host boundary edges) and the fitted trim curve.
func bsplineHostSolveEnd(ef edgeFillet, canal *bsplineHostCanal, v *topo.Vertex, vp []float64, iLo, iHi int, res Resolution) (bsplineHostEndTrim, string) {
	cap, reason := newBsplineHostCapPlane(v, ef)
	if reason != "" {
		return bsplineHostEndTrim{}, reason
	}
	end := bsplineHostEndTrim{cap: cap}
	snapTol := 4 * bsplineHostEnvelopeBound(res)
	if reason = bsplineHostEndCrossings(&end, ef, canal, vp, iLo, iHi, snapTol); reason != "" {
		return bsplineHostEndTrim{}, reason
	}
	trim, reason := bsplineHostTrimCurve(canal.surf, end, vp, iLo, iHi, bsplineHostEnvelopeBound(res))
	if reason != "" {
		return bsplineHostEndTrim{}, reason
	}
	end.trim = trim
	return end, ""
}

// bsplineHostEndCrossings solves and snaps both rail crossings of one end.
func bsplineHostEndCrossings(end *bsplineHostEndTrim, ef edgeFillet, canal *bsplineHostCanal, vp []float64, iLo, iHi int, snapTol float64) string {
	vA, qA, reason := railCapCrossing(canal.surf, 0, vp, iLo, iHi, end.cap)
	if reason != "" {
		return reason
	}
	vB, qB, reason := railCapCrossing(canal.surf, 1, vp, iLo, iHi, end.cap)
	if reason != "" {
		return reason
	}
	pA, reason := snapLandingToHostEdge(ef.a, ef.edge, qA, snapTol)
	if reason != "" {
		return reason
	}
	pB, reason := snapLandingToHostEdge(ef.b, ef.edge, qB, snapTol)
	if reason != "" {
		return reason
	}
	end.vA, end.vB, end.pA, end.pB = vA, vB, pA, pB
	return ""
}

// bsplineHostRailSpans extracts both rails' exact sub-spans between the two end
// crossings, oriented start→end and snapped onto the landings.
func bsplineHostRailSpans(canal *bsplineHostCanal, end0, end1 bsplineHostEndTrim) (railA, railB endSeg, reason string) {
	railA, reason = bsplineHostRailSpanSeg(canal.railA, end0.vA, end1.vA, end0.pA, end1.pA)
	if reason != "" {
		return railA, railB, reason
	}
	railB, reason = bsplineHostRailSpanSeg(canal.railB, end0.vB, end1.vB, end0.pB, end1.pB)
	return railA, railB, reason
}

// bsplineHostRailSpanSeg is one rail's sub-span vFrom→vTo with its terminal control
// points snapped onto the (host-boundary) landing points — an envelope-bound-sized end
// adjustment that makes the loop splice exact while leaving the interior curve untouched.
func bsplineHostRailSpanSeg(rail geom.Curve3, vFrom, vTo float64, pFrom, pTo math.Point3) (endSeg, string) {
	bs, ok := rail.(geom.BSplineCurve)
	if !ok {
		return endSeg{}, fmt.Sprintf("bspline-host runout: rail is %T, not a B-spline isocurve", rail)
	}
	sub, err := orientedSubSpan(bs, vFrom, vTo)
	if err != nil {
		return endSeg{}, fmt.Sprintf("bspline-host runout: rail sub-span failed: %v", err)
	}
	snapped, err := snapCurveEnds(sub, pFrom, pTo)
	if err != nil {
		return endSeg{}, fmt.Sprintf("bspline-host runout: rail end snap failed: %v", err)
	}
	return endSeg{from: pFrom, to: pTo, curve: snapped}, ""
}

// orientedSubSpan is the exact sub-span from→to, reversing when the parameters descend.
func orientedSubSpan(bs geom.BSplineCurve, vFrom, vTo float64) (geom.BSplineCurve, error) {
	if vFrom < vTo {
		return geom.SubSpanBSplineCurve(bs, vFrom, vTo)
	}
	sub, err := geom.SubSpanBSplineCurve(bs, vTo, vFrom)
	if err != nil {
		return geom.BSplineCurve{}, err
	}
	return geom.ReverseBSplineCurve(sub)
}

// snapCurveEnds rebuilds a curve with its two terminal control points replaced by the
// landing points (clamped ends: only the terminal spans move, by at most the envelope
// bound the snap tolerance enforced).
func snapCurveEnds(c geom.BSplineCurve, pFrom, pTo math.Point3) (geom.BSplineCurve, error) {
	ctrl := append([]math.Point3(nil), c.Ctrl...)
	ctrl[0], ctrl[len(ctrl)-1] = pFrom, pTo
	return geom.NewBSplineCurve(c.Degree, ctrl, append([]float64(nil), c.Weights...), append([]float64(nil), c.Knots...))
}

// hostBite is one splice a bitten face receives: the bite segment and the loop vertex the
// removed span carries.
type hostBite struct {
	seg   endSeg
	avoid math.Point3
}

// bsplineHostRunoutFaces builds every result face: the double-trimmed band, the two
// receded hosts, the bitten capping face(s) — supporting TWO bites on one capping face —
// and every untouched face verbatim.
func bsplineHostRunoutFaces(body *topo.Body, ef edgeFillet, canal *bsplineHostCanal, railA, railB endSeg, end0, end1 bsplineHostEndTrim, res Resolution) ([]filletFace, string) {
	loop := append([]endSeg{railA, end1.trim}, reverseEndSegs([]endSeg{railB})...)
	loop = append(loop, reverseEndSegs([]endSeg{end0.trim})...)
	bandFace := filletFace{surface: canal.surf, loops: []filletLoop{loopFromSegs(loop)}, parent: filletEdgeProvenance(ef.edge)}
	bites := bsplineHostBiteMap(ef, railA, railB, end0, end1)
	out := make([]filletFace, 0, len(body.Faces())+1)
	tol := res.Weld() * stdmath.Max(canal.r, 1)
	for _, f := range body.Faces() {
		ff, reason := bsplineHostFaceRetrim(f, bites[f], tol)
		if reason != "" {
			return nil, reason
		}
		out = append(out, ff)
	}
	return append(out, bandFace), ""
}

// bsplineHostBiteMap routes each bite to its face: the hosts recede onto the rails
// (removed span carries the picked edge, identified by its start vertex), each capping is
// bitten by its end's trim (removed corner carries that end's vertex). Two ends on one
// capping face append TWO bites.
func bsplineHostBiteMap(ef edgeFillet, railA, railB endSeg, end0, end1 bsplineHostEndTrim) map[*topo.Face][]hostBite {
	v0, v1 := ef.edge.StartVertex().Point(), ef.edge.EndVertex().Point()
	bites := map[*topo.Face][]hostBite{
		ef.a: {{seg: railA, avoid: v0}},
		ef.b: {{seg: railB, avoid: v0}},
	}
	bites[end0.cap.face] = append(bites[end0.cap.face], hostBite{seg: end0.trim, avoid: v0})
	bites[end1.cap.face] = append(bites[end1.cap.face], hostBite{seg: end1.trim, avoid: v1})
	return bites
}

// bsplineHostFaceRetrim splices a face's bites sequentially into its bitten loop (the
// retrimBittenLoop algebra generalized to multiple bites), or passes an untouched face
// through verbatim.
func bsplineHostFaceRetrim(f *topo.Face, bites []hostBite, tol float64) (filletFace, string) {
	if len(bites) == 0 {
		return wgPassthroughFace(f), ""
	}
	bitten := hostBittenLoop(f, bites[0].avoid, tol)
	outer := outerHostLoop(f)
	if bitten == nil || outer == nil {
		return filletFace{}, fmt.Sprintf("bspline-host runout: host %T has no loops to retrim", f.Geometry())
	}
	segs := segsFromLoop(bitten)
	for _, b := range bites {
		far, ok := farPathSegs(segs, b.seg.to, b.seg.from, b.avoid, tol)
		if !ok {
			return filletFace{}, fmt.Sprintf("bspline-host runout: host %T retrim declined (bite %v→%v off its loop)", f.Geometry(), b.seg.from, b.seg.to)
		}
		segs = append([]endSeg{b.seg}, far...)
	}
	return filletFace{surface: f.Geometry(), loops: wgHostLoopsWithRetrim(f, bitten, wgLoopFromSegs(segs)), parent: f.Lineage()}, ""
}

// wgHostLoopsWithRetrim rebuilds the host's loop set — the bitten loop replaced by its
// retrim, every other loop carried through with source identities, outer first (the
// hostLoopsWithRetrim contract with wgLoopFromSegs identity carriage).
func wgHostLoopsWithRetrim(host *topo.Face, bitten *topo.Loop, retrim filletLoop) []filletLoop {
	outer := outerHostLoop(host)
	out := []filletLoop{wgLoopForRole(outer, bitten, retrim)}
	for _, l := range host.Loops() {
		if l != outer {
			out = append(out, wgLoopForRole(l, bitten, retrim))
		}
	}
	return out
}

// wgLoopForRole yields the retrim for the bitten loop, an identity-carrying pass-through
// for any other.
func wgLoopForRole(l, bitten *topo.Loop, retrim filletLoop) filletLoop {
	if l == bitten {
		return retrim
	}
	return wgLoopFromSegs(segsFromLoop(l))
}
