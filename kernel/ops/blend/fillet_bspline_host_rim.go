// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CLOSED-rim body builder for the B-spline-host canal (J9-class convex pipe-cap rim,
// B2-class pipe-through-box weld ring; T8's CONCAVE variant declines separately — see
// bsplineHostRunoutBody's open-edge sibling, and fillet_rim_build.go's own cap hole-loop
// gap for the closed case): the marched canal band welds through the SAME host-agnostic
// rim rebuild the analytic and elliptic bands use (rebuildRim), with the band's two closed
// rails as the wall/cap contact curves. The re-aimed wall seam TRIES to keep its own curved
// meridian via the B-spline span carry (retainedBsplineSpan, exactRetainedSpanOnParent's
// B-spline arm) — a chord there sits off the pipe surface (the J2/J4 rimhost-carry lesson,
// retired for their axisymmetric sphere/torus hosts). On a GENERAL swept B-spline wall the
// carry's on-parent gate can genuinely decline (the numerically marched contact point has no
// guarantee of landing on the wall's own pre-existing structural seam, unlike an axisymmetric
// host) and falls back to the honest chord — a small, measured, ratcheted residual carried in
// knownOffSurfaceDebt (loopseg_onface_test.go), not silently absorbed.

// bsplineHostClosedRimBody welds one CLOSED B-spline-host rim pick. An empty reason means
// the returned body is the weld; a non-empty one names the obstruction and the body is nil
// (never a partial body — the ellipticClosedRimCanalBody contract).
func bsplineHostClosedRimBody(body *topo.Body, ef edgeFillet, canal *bsplineHostCanal) (*topo.Body, string) {
	e := ef.edge
	wallF, capF, ok := rimBandHosts(e)
	if !ok {
		return nil, fmt.Sprintf("bspline-host rim: edge %d must border one curved wall and one cap plane", e.ID())
	}
	wallRail, capRail, ok := bsplineHostRimRails(canal, wallF)
	if !ok {
		return nil, fmt.Sprintf("bspline-host rim: rails do not resolve against wall %T", wallF.Geometry())
	}
	rimV := e.StartVertex()
	seamEdge, bottomV := wallSeam(wallF, e, rimV)
	if seamEdge == nil {
		return nil, fmt.Sprintf("bspline-host rim: wall %T has no seam edge at the rim vertex to recede", wallF.Geometry())
	}
	rf := &rimFillet{
		cyl: wallF, cap: capF, rimEdge: e, seamEdge: seamEdge, rimV: rimV, bottomV: bottomV,
		cylTan: wallRail, capTan: capRail, band: canal.surf, r: canal.r,
		seamMid: canal.seamMid, concave: canal.concave,
	}
	b, err := rebuildRim(body, rf, canal.concave)
	if err != nil {
		return nil, fmt.Sprintf("bspline-host rim rebuild declined: %v", err)
	}
	wgBsplineRunoutValidateDebug(b)
	return b, ""
}

// bsplineHostRimRails maps the canal's A/B rails onto the rim rebuild's wall/cap roles by
// host-face identity (railA is the foot locus on canal.hostA by construction).
func bsplineHostRimRails(canal *bsplineHostCanal, wallF *topo.Face) (wallRail, capRail geom.Curve3, ok bool) {
	if canal.hostA == wallF {
		return canal.railA, canal.railB, true
	}
	if canal.hostB == wallF {
		return canal.railB, canal.railA, true
	}
	return nil, nil, false
}

// retainedBsplineSpan is the B-spline arm of the exact span-on-parent carry
// (exactRetainedSpanOnParent): the parent's OWN sub-curve between two points that already
// lie ON it, oriented from→to, or nil when either point is off the parent (the caller then
// ships its base chord). The exactness gate is scale-invariant on the span's own chord.
func retainedBsplineSpan(parent geom.BSplineCurve, from, to math.Point3) geom.Curve3 {
	t0, ok0 := bsplineParamOn(parent, from)
	t1, ok1 := bsplineParamOn(parent, to)
	if !ok0 || !ok1 || t0 == t1 {
		return nil
	}
	if t0 < t1 {
		sub, err := geom.SubSpanBSplineCurve(parent, t0, t1)
		if err != nil {
			return nil
		}
		return sub
	}
	sub, err := geom.SubSpanBSplineCurve(parent, t1, t0)
	if err != nil {
		return nil
	}
	rev, err := geom.ReverseBSplineCurve(sub)
	if err != nil {
		return nil
	}
	return rev
}

// bsplineParamOn inverts p on the parent and gates exactness: the foot must coincide with
// p within 1e-9 of the parent's polyline length (the rimSpanIsExact scale-invariance
// pattern, with the curve's own extent as the scale).
func bsplineParamOn(parent geom.BSplineCurve, p math.Point3) (float64, bool) {
	t, _ := geom.CurveParamAtPoint3(parent, p)
	scale := bsplinePolylineLength(parent)
	if scale <= 0 {
		return 0, false
	}
	if float64(parent.PointAt(t).DistanceTo(p)) > 1e-9*scale {
		return 0, false
	}
	return t, true
}

// bsplinePolylineLength is a coarse arc length of the parent (the exactness-gate scale).
func bsplinePolylineLength(c geom.BSplineCurve) float64 {
	lo, hi := c.Domain()
	const n = 64
	total := 0.0
	prev := c.PointAt(lo)
	for i := 1; i <= n; i++ {
		next := c.PointAt(lo + (hi-lo)*float64(i)/n)
		total += float64(prev.DistanceTo(next))
		prev = next
	}
	return total
}
