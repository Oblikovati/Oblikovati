// SPDX-License-Identifier: GPL-2.0-only

package validate

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// nonAnalyticEdgeSegments is the polyline fallback density used ONLY for an edge that has no
// closed-form plane projection (a spline, or a circle/arc seen edge-on). Every LINE, CIRCLE, and ARC
// projects ANALYTICALLY (geom.ProjectCurveToPlane, ADR-0055) and never touches this count — the
// containment decision for those is exact, retiring the tuned 64-sample chord test (#3478). A spline
// has no exact plane conic, so it is discretized here; this is the codebase's standard treatment of a
// non-analytic curve, not a tolerance that drives the analytic decision.
const nonAnalyticEdgeSegments = 48

// checkHoleContainment flags any PLANAR face whose hole loop is not strictly inside its outer loop — the
// B-rep invariant that a hole is an interior void, not a protrusion. A fillet that shrinks a face's outer
// loop into a coplanar hole leaves the hole poking through its own boundary (the base-plane defect behind
// the elliptical-prism blend cases): the tessellator then meshes malformed input and emits phantom
// "fill"/crack artifacts that look like meshing bugs. Reported as a distinct HolesContained flag + issues
// rather than folded into Valid, so it is a diagnostic tripwire until the fillet trim that fixes it lands.
//
// The containment test is EXACT over the loops' analytic 2D geometry (#3478): each edge projects into the
// face's own (u,v) frame as a line/arc/circle/ellipse (never a sampled polygon), so a curved outer rim is
// its true conic, not an inscribed chord that under-approximates its own boundary. Curved-face containment
// lives in surface (u,v) space and is a separate concern, out of scope here.
func (r *ValidationReport) checkHoleContainment(b *topo.Body) {
	r.HolesContained = true
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok || len(f.Loops()) < 2 {
			continue
		}
		ol := OuterLoopOf(f)
		outer := loopCurves2d(ol, pl)
		if len(outer) == 0 {
			continue
		}
		r.flagProtrudingHoles(f, ol, outer, pl)
	}
}

// flagProtrudingHoles records a HolesContained defect for every inner loop of f that escapes outer.
func (r *ValidationReport) flagProtrudingHoles(f *topo.Face, ol *topo.Loop, outer []geom.Curve2, pl geom.Plane) {
	outerVerts := loopVertices2d(ol, pl)
	for _, l := range f.Loops() {
		if l.IsOuter() || holeInsideOuter(loopCurves2d(l, pl), outer, loopVertices2d(l, pl), outerVerts) {
			continue
		}
		r.HolesContained = false
		r.Issues = append(r.Issues, fmt.Sprintf(
			"hole loop %d protrudes outside the outer loop of planar face %d (malformed B-rep face)",
			l.ID(), f.ID()))
	}
}

// OuterLoopOf returns the face's outer loop, or nil if it has none.
func OuterLoopOf(f *topo.Face) *topo.Loop {
	for _, l := range f.Loops() {
		if l.IsOuter() {
			return l
		}
	}
	return nil
}

// loopCurves2d projects every edge of the loop into pl's (u,v) frame as an analytic 2D curve. Edge
// traversal direction is dropped: containment is decided by point-in-region parity and boundary
// crossings, both orientation-independent, so a per-edge sense would only add noise.
func loopCurves2d(l *topo.Loop, pl geom.Plane) []geom.Curve2 {
	if l == nil {
		return nil
	}
	var cs []geom.Curve2
	for _, u := range l.EdgeUses() {
		cs = append(cs, edgeCurves2d(pl, u.Edge().Geometry())...)
	}
	return cs
}

// edgeCurves2d returns the analytic 2D projection of one edge, or its polyline fallback when the curve
// has no closed-form plane projection (spline / edge-on conic; see nonAnalyticEdgeSegments).
func edgeCurves2d(pl geom.Plane, c geom.Curve3) []geom.Curve2 {
	if c2, ok := geom.ProjectCurveToPlane(pl, c); ok {
		return []geom.Curve2{c2}
	}
	pts := geom.SampleCurve3(c, nonAnalyticEdgeSegments)
	segs := make([]geom.Curve2, 0, len(pts))
	for i := 0; i+1 < len(pts); i++ {
		segs = append(segs, geom.NewLineSegment2d(planeUV2d(pl, pts[i]), planeUV2d(pl, pts[i+1])))
	}
	return segs
}

// planeUV2d gives p's coordinates in pl's orthonormal (u,v) frame (the exact in-plane position of an
// edge that already lies on pl).
func planeUV2d(pl geom.Plane, p math.Point3) math.Point2 {
	d := pl.Origin.VectorTo(p)
	return math.P2(d.Dot(pl.UAxis.AsVector()), d.Dot(pl.VAxis.AsVector()))
}

// holeInsideOuter reports whether the hole loop lies inside the region bounded by outer. It is exact
// over the analytic edges (#3478): if any hole edge crosses the outer boundary away from a point where
// the two loops TOUCH, the hole pokes out; otherwise the hole is wholly on one side of that boundary
// (Jordan), so a single hole point decides it. A hole meeting the outer boundary at a shared vertex is
// the two loops kissing — a rim fillet's bore lip leaves a hole circle internally tangent to its face's
// outer circle — and stays contained.
func holeInsideOuter(hole, outer []geom.Curve2, holeVerts, outerVerts []math.Point2) bool {
	if len(hole) == 0 {
		return true
	}
	res := regionResolution2d(outer)
	if loopsCross2d(hole, outer, res, coincidentPoints2d(holeVerts, outerVerts, res.Plane())) {
		return false
	}
	return pointInRegion2d(holeProbePoint(hole), outer, res)
}

// loopVertices2d returns the loop's vertex positions in pl's frame. These are the TOPOLOGICAL points a
// loop is pinned to; a projected curve's parametric endpoints are not — a full circle's seam sits
// wherever the chart's u-axis happens to point, so it moves with the plane's frame and names no vertex.
func loopVertices2d(l *topo.Loop, pl geom.Plane) []math.Point2 {
	if l == nil {
		return nil
	}
	out := make([]math.Point2, 0, len(l.EdgeUses()))
	for _, u := range l.EdgeUses() {
		out = append(out, planeUV2d(pl, u.Edge().StartVertex().Point()), planeUV2d(pl, u.Edge().EndVertex().Point()))
	}
	return out
}

// coincidentPoints2d returns the points the two loops share within tol — the vertices at which they
// touch. tol is the on-boundary classification band, the same band the crossing hits are located to.
func coincidentPoints2d(a, b []math.Point2, tol float64) []math.Point2 {
	var out []math.Point2
	for _, p := range a {
		for _, q := range b {
			if float64(p.DistanceTo(q)) <= tol {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

// holeProbePoint returns an interior point of the hole's first edge — the decisive sample once the
// hole is known not to cross the outer boundary.
func holeProbePoint(hole []geom.Curve2) math.Point2 {
	return hole[0].PointAt(curveMidParam(hole[0]))
}

// curveMidParam returns the midpoint parameter of c's domain, defaulting an unbounded domain to 0.5.
func curveMidParam(c geom.Curve2) float64 {
	lo, hi := c.Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return 0.5
	}
	return (lo + hi) / 2
}

// regionResolution2d derives the size-relative tolerance scale (ADR-0042) from the region's extent.
// The three-point-per-edge sampling only sizes the tolerance and the parity ray; it never decides
// containment.
func regionResolution2d(edges []geom.Curve2) geom.Resolution {
	var pts []math.Point2
	for _, e := range edges {
		lo, hi := e.Domain()
		if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
			lo, hi = 0, 1
		}
		pts = append(pts, e.PointAt(lo), e.PointAt((lo+hi)/2), e.PointAt(hi))
	}
	return geom.ResolutionForPoints2D(pts)
}

// loopsCross2d reports whether any hole edge crosses any outer edge away from a shared vertex.
func loopsCross2d(hole, outer []geom.Curve2, res geom.Resolution, touch []math.Point2) bool {
	for _, h := range hole {
		for _, o := range outer {
			if edgesCross2d(h, o, res, touch) {
				return true
			}
		}
	}
	return false
}

// edgesCross2d reports whether the two analytic edges meet anywhere other than a point at which the
// loops touch. A meeting at a shared vertex is the two loops kissing, not a protrusion; any other
// meeting is the hole crossing through the outer boundary.
func edgesCross2d(h, o geom.Curve2, res geom.Resolution, touch []math.Point2) bool {
	for _, p := range curveCurve2dHits(h, o, res.Plane()) {
		if !nearAnyPoint2d(p, touch, res.Plane()) {
			return true
		}
	}
	return false
}

// nearAnyPoint2d reports whether p sits within tol of any of the given points.
func nearAnyPoint2d(p math.Point2, pts []math.Point2, tol float64) bool {
	for _, q := range pts {
		if float64(p.DistanceTo(q)) <= tol {
			return true
		}
	}
	return false
}

// curveCurve2dHits returns the intersection points of two analytic 2D curves, reusing the exact
// analytic primitives (#3478): a segment/line against any curve, a circle against any curve, and an arc
// through its support circle filtered onto its sweep. Only ellipse-vs-ellipse (no closed form in the
// codebase) falls back to sampling one operand — a case a planar line/arc/circle face boundary never
// reaches, so the arc discrimination stays exact.
func curveCurve2dHits(a, b geom.Curve2, tol float64) []math.Point2 {
	if seg, ok := a.(geom.LineSegment2d); ok {
		return geom.SegmentCurve2dIntersection(seg, b)
	}
	if seg, ok := b.(geom.LineSegment2d); ok {
		return geom.SegmentCurve2dIntersection(seg, a)
	}
	if ln, ok := a.(geom.Line2d); ok {
		return geom.LineCurve2dIntersection(ln, b)
	}
	if ln, ok := b.(geom.Line2d); ok {
		return geom.LineCurve2dIntersection(ln, a)
	}
	return circleFamilyHits(a, b, tol)
}

// circleFamilyHits dispatches the curved operands: a full circle against any curve is exact, and an arc
// reuses that through its support circle, keeping only the hits that fall on the arc's real sweep.
func circleFamilyHits(a, b geom.Curve2, tol float64) []math.Point2 {
	if c, ok := a.(geom.Circle2d); ok {
		return geom.CircleCurve2dIntersection(c, b)
	}
	if c, ok := b.(geom.Circle2d); ok {
		return geom.CircleCurve2dIntersection(c, a)
	}
	if arc, ok := a.(geom.Arc2d); ok {
		return arcCurve2dHits(arc, b, tol)
	}
	if arc, ok := b.(geom.Arc2d); ok {
		return arcCurve2dHits(arc, a, tol)
	}
	return ellipsePairHits(a, b)
}

// arcCurve2dHits intersects arc's support circle with other, then keeps the hits that lie on the arc's
// actual sweep (geom.Arc2d.ContainsPoint) — an exact arc-vs-curve crossing without chording the arc.
func arcCurve2dHits(arc geom.Arc2d, other geom.Curve2, tol float64) []math.Point2 {
	support := geom.NewCircle2d(arc.Center, arc.Radius)
	var out []math.Point2
	for _, p := range geom.CircleCurve2dIntersection(support, other) {
		if arc.ContainsPoint(p, tol) {
			out = append(out, p)
		}
	}
	return out
}

// ellipsePairHits is the no-closed-form fallback for two elliptic/spline operands: sample a into
// segments and intersect each against b. Unreached by a line/arc/circle face boundary; see
// curveCurve2dHits.
func ellipsePairHits(a, b geom.Curve2) []math.Point2 {
	lo := a.PointAt(0)
	var out []math.Point2
	for i := 1; i <= nonAnalyticEdgeSegments; i++ {
		hi := a.PointAt(float64(i) / nonAnalyticEdgeSegments)
		out = append(out, geom.SegmentCurve2dIntersection(geom.NewLineSegment2d(lo, hi), b)...)
		lo = hi
	}
	return out
}

// probeRayDirs are three generically-tilted parity-ray directions (#3478): none axis-aligned, so a ray
// never runs along an edge, and mutually well-separated so a vertex or tangency grazed by one is cleared
// by the other two.
var probeRayDirs = []math.Vector2{math.V2(1, 0.11), math.V2(-0.37, 1), math.V2(-1, -0.61)}

// pointInRegion2d reports whether q is inside the closed region bounded by edges, by exact analytic ray
// parity: a ray from q crosses the boundary an odd number of times iff q is inside. Three rays vote so a
// single grazing ray cannot flip the result. Crossings are found on the true analytic curves
// (geom.SegmentCurve2dIntersection refines onto them), never on a chord — the retirement of the
// 64-sample outer polygon (#3478).
func pointInRegion2d(q math.Point2, edges []geom.Curve2, res geom.Resolution) bool {
	length := 4 * res.Size() // a ray this long from any interior point clears the region bbox
	votes := 0
	for _, dir := range probeRayDirs {
		if rayCrossings2d(q, dir, length, edges)%2 == 1 {
			votes++
		}
	}
	return votes >= 2
}

// rayCrossings2d counts how many times the ray from q along dir (of the given length) crosses edges.
func rayCrossings2d(q math.Point2, dir math.Vector2, length float64, edges []geom.Curve2) int {
	ray := geom.NewLineSegment2d(q, q.TranslateBy(dir.Scale(math.Scalar(length))))
	n := 0
	for _, e := range edges {
		n += len(geom.SegmentCurve2dIntersection(ray, e))
	}
	return n
}
