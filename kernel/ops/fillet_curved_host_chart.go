// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// M5 Slice B, Task N2 (C1): the host intrinsic CHARTS. A rolling arm's contact ruling is a straight
// 1-D ray in the developed chart of the host it rolls on — the plane's isometric (u,v) or the
// cylinder's (θ,z) — so the crossing with the trimmed boundary loop is a 2-D ray/segment test rather
// than a 3-D SSI (derivation R.1; Patrikalakis & Maekawa §point-in-face). This file owns "1-D crossing
// in the host's own coordinates": the hostChart interface, its plane/cyl implementations, the single
// hostChartFor dispatch, and the shared chartRulingExit ray-cast that replaces axialExtremeEnd's
// global-axial-extreme scan (which overshot any intermediate trim edge — N7 s_4: global rim z=130, true
// runout z=80). planeChart + its ray helpers were moved here verbatim from fillet_curved_retrim_loop.go
// (behavior-preserving) and retargeted to the interface.

// hostChart is a host surface's intrinsic 2-D developed chart: the plane's isometric (u,v) or the
// cylinder's (θ,z). A rolling arm's contact ruling is a 1-D ray in this chart, so the crossing with
// the trimmed boundary is a 2-D ray/segment test rather than a 3-D SSI.
type hostChart interface {
	to2(p math.Point3) math.Point2
	to3(q math.Point2) math.Point3
}

// planeChart maps between 3D points on a plane and its isometric 2D coordinates (so in-plane ray /
// circle intersections reuse the tested 2D primitives, and a shoelace over the chart is true area).
type planeChart struct{ pl geom.Plane }

func (c planeChart) to2(p math.Point3) math.Point2 {
	w := c.pl.Origin.VectorTo(p)
	return math.P2(w.Dot(c.pl.UAxis.AsVector()), w.Dot(c.pl.VAxis.AsVector()))
}

func (c planeChart) to3(q math.Point2) math.Point3 {
	return c.pl.Origin.TranslateBy(c.pl.UAxis.AsVector().Scale(q.X)).TranslateBy(c.pl.VAxis.AsVector().Scale(q.Y))
}

// cylChart develops a cylinder wall to (θ, z): θ = atan2 about AxisDir relative to Ref (seam at ±π),
// z = signed axial coordinate. A cylinder arm's wall ruling is θ = θ₀ constant (a vertical chart ray);
// a horizontal rim (a plane cut ⊥ the axis) is z = const (a horizontal chart line). This one chart
// handles both and replaces axialExtremeEnd's global-extreme scan.
type cylChart struct {
	origin math.Point3
	axis   math.Vector3 // unit
	ref    math.Vector3 // unit, θ=0 direction (⊥ axis)
	bi     math.Vector3 // unit, axis × ref — θ=+π/2 direction
	radius float64
}

// newCylChart builds the (θ,z) chart for cyl. AxisDir/Ref are already unit (geom.Cylinder invariant),
// so no re-normalization is needed. Example: newCylChart(wall).to2(p) is p's (θ, z) on the wall.
func newCylChart(cyl geom.Cylinder) cylChart {
	axis := cyl.AxisDir.AsVector()
	ref := cyl.Ref.AsVector()
	return cylChart{origin: cyl.Origin, axis: axis, ref: ref, bi: axis.Cross(ref), radius: cyl.Radius}
}

func (c cylChart) to2(p math.Point3) math.Point2 {
	w := c.origin.VectorTo(p)
	theta := stdmath.Atan2(float64(w.Dot(c.bi)), float64(w.Dot(c.ref)))
	return math.P2(theta, float64(w.Dot(c.axis)))
}

func (c cylChart) to3(q math.Point2) math.Point3 {
	theta, z := float64(q.X), float64(q.Y)
	radial := c.ref.Scale(math.Scalar(stdmath.Cos(theta) * c.radius)).
		Add(c.bi.Scale(math.Scalar(stdmath.Sin(theta) * c.radius)))
	return c.origin.TranslateBy(radial).TranslateBy(c.axis.Scale(math.Scalar(z)))
}

// hostChartFor is the SINGLE plane/cylinder chart dispatch, reused by the arm-ruling terminator (N2),
// the far-path splice (N4), and the closure gate (N5). ok=false for any other surface (only planar and
// cylindrical hosts carry a developed chart in Slice A/B).
func hostChartFor(surf geom.Surface) (hostChart, bool) {
	switch s := surf.(type) {
	case geom.Plane:
		return planeChart{s}, true
	case geom.Cylinder:
		return newCylChart(s), true
	}
	return nil, false // host is neither a plane nor a cylinder — no intrinsic chart
}

// chartRulingExit returns the nearest FORWARD (t>tol) 3D point where the chart ruling (chart origin o2,
// chart direction d2) leaves the loop segs — the outer end of a straight ruling rail on a host. It is
// the generalized planeRayLoopExit body: chart-agnostic over hostChart, so a cylinder wall (θ,z) and a
// plane (u,v) share one first-crossing scan.
//
// N3 (C2): besides each edge's INTERIOR line/arc crossing, it also tries each edge's ENDPOINTS as
// candidate exits (rayPointHit2d) and keeps the nearest across both. This is the N7 x=50 plane fix: a
// bitten loop is opened at its corner, so near that corner only ONE of the two (formerly-adjacent) edges
// may remain in segs — an independently-computed ruling target (a few thousandths off the loop's own
// vertex, same provenance gap runoutAgrees tolerates) can then miss that lone edge's own [0,1] interior
// span with nothing on the "other side" to catch it, though the vertex itself is well within tol
// (derivation R.1 pitfalls). The winning landing is then snapped onto an exact loop vertex when one is
// within tol, so the far-path splice reuses it rather than the ray's not-quite-exact numeric landing.
func chartRulingExit(ch hostChart, segs []endSeg, o2 math.Point2, d2 math.Vector2, tol float64) (math.Point3, bool) {
	hit, found := nearestRulingCandidate(ch, segs, o2, d2, tol)
	if !found {
		return math.Point3{}, false
	}
	if v, ok := snapToVertex(hit, segs, tol); ok {
		return v, true
	}
	return hit, true
}

// nearestRulingCandidate scans every seg's interior crossing AND both its endpoints, keeping the
// nearest forward (t>tol) hit — the merged interior+endpoint scan chartRulingExit's doc describes.
func nearestRulingCandidate(ch hostChart, segs []endSeg, o2 math.Point2, d2 math.Vector2, tol float64) (math.Point3, bool) {
	best, found := stdmath.Inf(1), false
	var bestPt math.Point3
	keep := func(t float64, q math.Point3, ok bool) {
		if ok && t > tol && t < best {
			best, bestPt, found = t, q, true
		}
	}
	for _, s := range segs {
		keep(rayEdgeHit2d(ch, s, o2, d2, tol))
		keep(rayPointHit2d(ch, o2, d2, s.from, tol))
		keep(rayPointHit2d(ch, o2, d2, s.to, tol))
	}
	return bestPt, found
}

// rayPointHit2d tests one loop-edge ENDPOINT p as a candidate ruling exit, independent of whether p's
// own edge(s) claim it as an interior crossing: p counts as a forward hit at ray parameter t when its
// chart projection onto the ray lands within tol of p in 3D (the to3 round trip keeps the check
// dimensionally correct on a cylinder chart, where u=θ is an angle and v=z is a length — a raw chart-
// space distance would wrongly mix the two). This is what lets chartRulingExit land on a vertex that a
// single one-sided open edge (post-bite) cannot interior-cross (N7 x=50 plane, see chartRulingExit doc).
func rayPointHit2d(ch hostChart, o2 math.Point2, d2 math.Vector2, p math.Point3, tol float64) (float64, math.Point3, bool) {
	d2Len2 := float64(d2.Dot(d2))
	if d2Len2 < sinFloor*sinFloor { // degenerate ruling direction — no forward projection to test against
		return 0, math.Point3{}, false
	}
	p2 := ch.to2(p)
	t := float64(o2.VectorTo(p2).Dot(d2)) / d2Len2
	if t <= 0 {
		return 0, math.Point3{}, false // p sits behind the ruling's origin, not a forward exit
	}
	closest := ch.to3(o2.TranslateBy(d2.Scale(math.Scalar(t))))
	if float64(closest.DistanceTo(p)) > tol {
		return 0, math.Point3{}, false // p is not near the ruling's line
	}
	return t, p, true
}

// snapToVertex returns the loop vertex within tol of p (checking both endpoints of every seg — a
// landing can coincide with an edge's "from" OR "to", e.g. an open post-bite chain where the target
// vertex is only ever a "from"), so a landing that coincides with an existing corner reuses it instead
// of splitting the edge into a zero-length sliver (derivation R.1 pitfalls; insertSplits/splitSeg
// already no-op on a non-interior point, so this is what makes the exit point BE that exact vertex
// rather than a near-miss the split logic would then reject anyway).
func snapToVertex(p math.Point3, segs []endSeg, tol float64) (math.Point3, bool) {
	for _, s := range segs {
		if float64(s.from.DistanceTo(p)) <= tol {
			return s.from, true
		}
		if float64(s.to.DistanceTo(p)) <= tol {
			return s.to, true
		}
	}
	return math.Point3{}, false
}

// rayEdgeHit2d intersects the chart ray (o2 + t·d2) with one loop edge, returning the forward hit's
// ray parameter and 3D point. A straight edge uses a 2D line/segment solve (valid on both charts: our
// straight wall edges are axial → vertical chart segments, our straight plane edges are in-plane). An
// arc edge on a plane uses the analytic 2D line/circle crossing; on a cylinder wall a horizontal rim
// arc develops to a z=const chart line, handled seam-safely by rayCylArc2d (θ-seam via arcParam's
// wrapToSweep).
func rayEdgeHit2d(ch hostChart, s endSeg, o2 math.Point2, d2 math.Vector2, tol float64) (float64, math.Point3, bool) {
	if !s.arc {
		return raySegment2d(ch, o2, d2, s)
	}
	if cyl, ok := ch.(cylChart); ok {
		return rayCylArc2d(cyl, o2, d2, s, tol)
	}
	return rayArc2d(ch, o2, d2, s, tol)
}

// raySegment2d solves the ray o2+t·d2 against the straight edge s in the chart.
func raySegment2d(ch hostChart, o2 math.Point2, d2 math.Vector2, s endSeg) (float64, math.Point3, bool) {
	a2, b2 := ch.to2(s.from), ch.to2(s.to)
	e := a2.VectorTo(b2)
	denom := d2.Cross(e)
	if collinear(d2, e, denom) {
		return 0, math.Point3{}, false
	}
	ao := o2.VectorTo(a2)
	t := float64(ao.Cross(e) / denom)
	u := float64(ao.Cross(d2) / denom)
	if u < 0 || u > 1 {
		return 0, math.Point3{}, false
	}
	return t, ch.to3(o2.TranslateBy(d2.Scale(math.Scalar(t)))), true
}

// collinear reports whether ray direction d2 and edge direction e are parallel within the scale-free
// angular floor sinFloor: |d2×e|/(|d2||e|) is exactly sin(angle between them), so below sinFloor the
// edge is numerically indistinguishable from running ALONG the ruling — solving the line intersection
// would divide by a ~0 denominator (N3, R.1 pitfalls). It replaces a prior raw |denom|<1e-15 check,
// which was not scale-invariant: |denom|=|d2||e|·sinθ also shrinks with short d2/e even at a real
// (non-parallel) angle, so a fixed absolute floor could wrongly reject a short-but-crossing edge, or
// (dually) accept a long-but-grazing one. A grazing/collinear edge is skipped here, not fatal — the
// caller's far-vertex authority (armRulingEnd's runoutAgrees, C1) is the fallback when a ruling runs
// along an edge instead of crossing it.
func collinear(d2, e math.Vector2, cross math.Scalar) bool {
	denomLen := float64(d2.Length()) * float64(e.Length())
	if denomLen < 1e-15 { // near-zero-length direction(s) — floating-underflow guard, not a model tolerance
		return true
	}
	return stdmath.Abs(float64(cross))/denomLen < sinFloor
}

// rayArc2d solves the ray against an arc edge in the PLANE chart, keeping the nearest forward crossing
// that lies inside the arc's sweep. (An in-plane arc develops to a true 2D circle only on an isometric
// plane chart — the cylinder wall's horizontal rim arc goes through rayCylArc2d instead.)
func rayArc2d(ch hostChart, o2 math.Point2, d2 math.Vector2, s endSeg, tol float64) (float64, math.Point3, bool) {
	line, err := geom.NewLine2d(o2, d2)
	if err != nil {
		return 0, math.Point3{}, false
	}
	arc := s.curve.(geom.Arc3d)
	c2 := geom.NewCircle2d(ch.to2(arc.Center), arc.Radius)
	best, found := stdmath.Inf(1), false
	var bestPt math.Point3
	for _, p2 := range geom.LineCircle2dIntersection(line, c2, tol) {
		t := float64(o2.VectorTo(p2).Dot(d2) / d2.Dot(d2))
		q := ch.to3(p2)
		if _, ok := arcParam(arc, q, tol); ok && t < best {
			best, bestPt, found = t, q, true
		}
	}
	return best, bestPt, found
}

// rayCylArc2d crosses a vertical wall ruling (θ=θ₀ in the cyl chart) with a horizontal rim arc (const z
// on the wall): the crossing is the wall point at (θ₀, z_rim), where z_rim is the rim height read off
// the arc centre's chart z (a rim circle is centred on the axis). It counts only when θ₀ lies inside
// the arc's sweep — arcParam is 2π-seam-safe via wrapToSweep, so a rim straddling the ±π seam is not
// missed and no divide-by-near-zero occurs (d2=(0,±1) here, |d2|²=1).
func rayCylArc2d(ch cylChart, o2 math.Point2, d2 math.Vector2, s endSeg, tol float64) (float64, math.Point3, bool) {
	arc := s.curve.(geom.Arc3d)
	zRim := ch.to2(arc.Center).Y // rim height: the rim circle's centre sits on the axis, so its chart z is z_rim
	cand := ch.to3(math.P2(o2.X, zRim))
	if _, ok := arcParam(arc, cand, tol); !ok {
		return 0, math.Point3{}, false // θ₀ outside the rim arc's sweep — the vertical ruling misses this rim
	}
	t := float64(o2.VectorTo(ch.to2(cand)).Dot(d2)) / float64(d2.Dot(d2))
	return t, cand, true
}
