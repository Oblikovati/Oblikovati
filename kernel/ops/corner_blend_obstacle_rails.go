// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// obstacleRails builds the four FillSurface boundary curves for the obstacle patch and makes them
// pairwise-compatible (c0/c1 share degree+knots, d0/d1 share degree+knots — FillSurface's
// precondition). It FIRST nil-checks the wing pointers: the end rails MUST be the abutting
// cylinder wings' section arcs (reused for G1-by-identity and to kill the T-junction crack). A
// missing wing (or an under-populated rim sample list) => ok=false => the provider declines and
// the caller honest-rejects (ADR-3) rather than build a fresh, crack-inducing arc.
func obstacleRails(of *ObstacleFeature) (c0, c1, d0, d1 geom.BSplineCurve, ok bool) {
	if invalidObstacle(of) {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, false
	}
	a, d, pMinus, pPlus := railCorners(of)
	c0, c1, d0, d1, ok = pinRawRails(of, a, d, pMinus, pPlus)
	if !ok {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, false
	}
	return finishRails(c0, c1, d0, d1, a, d, pMinus, pPlus)
}

// invalidObstacle is the mandatory nil-check (regression-crack defense): a missing wing/wall
// pointer or an under-populated rim sample list means obstacleRails must decline rather than hand
// FillSurface a bad quad.
func invalidObstacle(of *ObstacleFeature) bool {
	return of == nil || of.WingStart == nil || of.WingEnd == nil || of.WallLine == nil || len(of.RimArcPts) < 4
}

// railCorners derives the four exact FillSurface corners from of: P-, P+ are the detector's Nodes
// (canonical, spec §3); A, D are each wing arc's WALL-side endpoint (the one NOT already at its
// node) — the point WallLine must also pass through for the patch to close without a crack.
func railCorners(of *ObstacleFeature) (a, d, pMinus, pPlus math.Point3) {
	pMinus, pPlus = of.Nodes[0], of.Nodes[1]
	return farEndpoint(of.WingStart, pMinus), farEndpoint(of.WingEnd, pPlus), pMinus, pPlus
}

// farEndpoint returns whichever of c's two endpoints is farther from node — by construction the
// node-side endpoint sits AT node, so the far one is the wall end (A or D).
func farEndpoint(c geom.Curve3, node math.Point3) math.Point3 {
	lo, hi := c.Domain()
	p0, p1 := c.PointAt(lo), c.PointAt(hi)
	if p0.DistanceTo(node) > p1.DistanceTo(node) {
		return p0
	}
	return p1
}

// pinRawRails converts each source curve to a BSplineCurve and orients+pins it to the canonical
// corners (railCorners) — the step that guarantees a shared endpoint is the SAME math.Point3 value
// across rails, not merely a close one. The endpoint-match weld is model-relative (ADR-0042):
// ResolutionForPoints over the four corners scales it on a µm-part or a km-part, never a bare
// cm-anchored absolute epsilon.
func pinRawRails(of *ObstacleFeature, a, d, pMinus, pPlus math.Point3) (c0, c1, d0, d1 geom.BSplineCurve, ok bool) {
	rawC0, ok0 := asBSplineCurve(of.WallLine)
	rawC1, ok1 := obstacleRimArc(of)
	rawD0, ok2 := asBSplineCurve(of.WingStart)
	rawD1, ok3 := asBSplineCurve(of.WingEnd)
	if !ok0 || !ok1 || !ok2 || !ok3 {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, false
	}
	tol := ResolutionForPoints([]math.Point3{a, d, pMinus, pPlus}).Weld()
	c0, okC0 := pinnedRail(rawC0, a, d, tol)
	c1, okC1 := pinnedRail(rawC1, pMinus, pPlus, tol)
	d0, okD0 := pinnedRail(rawD0, a, pMinus, tol)
	d1, okD1 := pinnedRail(rawD1, d, pPlus, tol)
	return c0, c1, d0, d1, okC0 && okC1 && okD0 && okD1
}

// finishRails makes each rail pair (c0/c1 in u, d0/d1 in v) compatible and then re-pins the four
// corners to the canonical points, so the shared corners stay bit-identical even if degree
// elevation or knot refinement introduced any rounding — CoonsFill's corner check is a strict
// 1e-9 equality.
func finishRails(c0, c1, d0, d1 geom.BSplineCurve, a, d, pMinus, pPlus math.Point3) (geom.BSplineCurve, geom.BSplineCurve, geom.BSplineCurve, geom.BSplineCurve, bool) {
	c0, c1, ok := makeRailPair(c0, c1)
	if !ok {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, false
	}
	d0, d1, ok = makeRailPair(d0, d1)
	if !ok {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, false
	}
	pinEnds(&c0, a, d)
	pinEnds(&c1, pMinus, pPlus)
	pinEnds(&d0, a, pMinus)
	pinEnds(&d1, d, pPlus)
	return c0, c1, d0, d1, true
}

// asBSplineCurve converts a rail source to the non-rational BSplineCurve representation
// FillSurface consumes: a LineSegment converts exactly (2-point degree-1 curve, no fitting
// error); any other curve (Arc3d etc.) is rebuilt to a clean degree-3, 8-control-point B-spline
// (RebuildCurve) — well within the 1% shape-deviation gate for a quarter-circle wing arc.
func asBSplineCurve(c geom.Curve3) (geom.BSplineCurve, bool) {
	if seg, isLine := c.(geom.LineSegment); isLine {
		bs, err := geom.NewBSplineCurveUniformWeights(1, []math.Point3{seg.StartPoint, seg.EndPoint}, []float64{0, 0, 1, 1})
		return bs, err == nil
	}
	bs, _, err := geom.RebuildCurve(c, 3, 8, 24)
	return bs, err == nil
}

// obstacleRimArc fits the base-rim sub-arc between the two Nodes directly from of.RimArcPts — the
// detector's ordered P- -> P+ dip-side samples (task 6) — so no point-inversion is needed, only a
// least-squares fit that interpolates the (already exact) endpoint samples.
func obstacleRimArc(of *ObstacleFeature) (geom.BSplineCurve, bool) {
	nctrl := min(8, len(of.RimArcPts))
	bs, err := geom.NewApproximatedBSplineCurve(of.RimArcPts, 3, nctrl, geom.FitCentripetal)
	return bs, err == nil
}

// makeRailPair wraps geom.MakeCompatible (degree-elevate + knot-merge) as a boolean gate —
// obstacleRails only ever needs "did it work", not the underlying error (ADR-3 honest-reject).
func makeRailPair(a, b geom.BSplineCurve) (geom.BSplineCurve, geom.BSplineCurve, bool) {
	ra, rb, err := geom.MakeCompatible(a, b)
	return ra, rb, err == nil
}

// pinnedRail orients raw so it runs from->to (reversing if it instead runs to->from within the
// model-relative weld tol) and then overwrites its two end control points with the exact from/to
// values — what makes a corner shared between two rails bit-identical, not merely close (see
// finishRails). tol is a model-scaled weld from ResolutionForPoints (ADR-0042), never an absolute
// epsilon, so it holds on µm and km parts alike.
func pinnedRail(raw geom.BSplineCurve, from, to math.Point3, tol float64) (geom.BSplineCurve, bool) {
	last := len(raw.Ctrl) - 1
	switch {
	case raw.Ctrl[0].DistanceTo(from) <= tol && raw.Ctrl[last].DistanceTo(to) <= tol:
	case raw.Ctrl[0].DistanceTo(to) <= tol && raw.Ctrl[last].DistanceTo(from) <= tol:
		rev, ok := reverseBSplineCurve(raw)
		if !ok {
			return geom.BSplineCurve{}, false
		}
		raw = rev
	default:
		return geom.BSplineCurve{}, false
	}
	pinEnds(&raw, from, to)
	return raw, true
}

// pinEnds forces c's first and last control points to exact values — a clamped B-spline's end
// control point already equals PointAt(domain end), so this only removes floating-point noise
// left by fitting/degree-elevation/knot-refinement; it never changes the curve's shape.
func pinEnds(c *geom.BSplineCurve, from, to math.Point3) {
	c.Ctrl[0] = from
	c.Ctrl[len(c.Ctrl)-1] = to
}

// reverseBSplineCurve returns c traced start-to-end reversed — same geometry, control points and
// weights reversed and the knot vector reflected about the domain (lo+hi-k). geom has no curve
// Reverse (grepped kernel/geom); this package's rail orientation (pinnedRail) is the only caller.
// It rebuilds through the validating geom.NewBSplineCurve constructor (validateBSpline +
// requirePositiveWeights); ok=false if the reversed net is somehow rejected (a shape-preserving
// reversal of a valid curve never should — reflected weights stay positive, knots stay ascending).
func reverseBSplineCurve(c geom.BSplineCurve) (geom.BSplineCurve, bool) {
	n := len(c.Ctrl)
	ctrl, weights := make([]math.Point3, n), make([]float64, n)
	for i := range c.Ctrl {
		ctrl[n-1-i], weights[n-1-i] = c.Ctrl[i], c.Weights[i]
	}
	lo, hi := c.Knots[0], c.Knots[len(c.Knots)-1]
	knots := make([]float64, len(c.Knots))
	for i, k := range c.Knots {
		knots[len(c.Knots)-1-i] = lo + hi - k
	}
	rev, err := geom.NewBSplineCurve(c.Degree, ctrl, weights, knots)
	return rev, err == nil
}

// obstacleSides returns the four FillSide continuity orders: the wall (c0) and both wings
// (d0, d1) are G1 (Order 1) — the fill matches tangent to the wall and wing neighbour RIBBONS
// (extrudeRibbon), killing the T-junction crack. Each neighbour is a degree-(p,1) extrusion whose
// rail edge is its VMinEdge, so ALL three share on VMinEdge. The RIM (c1) is G0 (Order 0): the
// fillet meets the vertical obstacle wall at a SHARP base-rim crease, and forcing G1 to a vertical
// wall inflates the patch into a sliver (spec §Item 2, resolved: G0).
//
//nolint:unparam // of is part of the fixed signature Task 4's provider calls (brief resolution); unused today, reserved for a future HostPlane-driven side.
func obstacleSides(of *ObstacleFeature, wingL, wingR, wall geom.BSplineSurface) [4]geom.FillSide {
	return [4]geom.FillSide{
		{Adjacent: wall, AdjEdge: geom.VMinEdge, Order: 1}, // c0 wall  -> G1 (ribbon VMinEdge = c0)
		{Order: 0}, // c1 rim -> G0 (no ribbon)
		{Adjacent: wingL, AdjEdge: geom.VMinEdge, Order: 1}, // d0 wingL -> G1 (ribbon VMinEdge = d0)
		{Adjacent: wingR, AdjEdge: geom.VMinEdge, Order: 1}, // d1 wingR -> G1 (ribbon VMinEdge = d1)
	}
}
