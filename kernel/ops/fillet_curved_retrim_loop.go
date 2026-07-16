// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// M5 Slice A, Task 5.3 — the loop machinery behind retrimCurvedHost. It reads a host face's
// original outer loop, splits its edges where the arm contact rails land, and returns the surviving
// "far" path (the boundary away from the trihedral vertex) so the caller can splice the rails in.
// The two circular splits (the wall bottom-rim sub-arc, the bottom-cap foot arc) are re-emitted as
// exact Arc3d edges, never chords, so the assembled mesh cannot bulge or crack (tessellation-first).

// originalHostSegs reads the host's outer loop as an ordered ring of endSegs (each from→to carrying
// the edge's Arc3d curve oriented to the traversal, or nil for a straight edge).
func originalHostSegs(host *topo.Face) []endSeg {
	loops := host.Loops()
	if len(loops) == 0 {
		return nil
	}
	uses := loops[0].EdgeUses()
	segs := make([]endSeg, 0, len(uses))
	for _, u := range uses {
		segs = append(segs, endSegFromUse(u))
	}
	return segs
}

// endSegFromUse turns one edge use into an endSeg, carrying an Arc3d curve (with its midpoint, for
// re-orientation and splitting) or leaving the curve nil for a straight edge.
func endSegFromUse(u *topo.EdgeUse) endSeg {
	from, to := useFromVertex(u).Point(), useToVertex(u).Point()
	if arc, ok := survivorCurve(u).(geom.Arc3d); ok {
		return endSeg{from: from, to: to, curve: arc, mid: arc.PointAt(0.5), arc: true}
	}
	return endSeg{from: from, to: to}
}

// useToVertex returns the to-vertex of an edge use (honouring reversal) — the sibling of
// useFromVertex.
func useToVertex(u *topo.EdgeUse) *topo.Vertex {
	if u.Reversed() {
		return u.Edge().StartVertex()
	}
	return u.Edge().EndVertex()
}

// bittenVertex is the original loop vertex nearest the corner ball centre C — the trihedral corner
// the retrim bites away. The surviving far path is the loop side that does NOT contain it.
func bittenVertex(segs []endSeg, c math.Point3) math.Point3 {
	best, bestD := segs[0].from, segs[0].from.DistanceTo(c)
	for _, s := range segs {
		if d := s.from.DistanceTo(c); d < bestD {
			best, bestD = s.from, d
		}
	}
	return best
}

// farPathSegs returns the original-loop boundary from fromP to toP that avoids the bitten vertex v —
// the "far" side kept by the retrim, with the two rail landing points spliced in as new vertices.
func farPathSegs(segs []endSeg, fromP, toP, v math.Point3, tol float64) ([]endSeg, bool) {
	ring := insertSplits(segs, []math.Point3{fromP, toP}, tol)
	i, j := indexOfSegFrom(ring, fromP, tol), indexOfSegFrom(ring, toP, tol)
	if i < 0 || j < 0 {
		return nil, false // a rail landing point does not lie on the original loop — cannot close
	}
	if fwd := segsForward(ring, i, j); !pathHasVertex(fwd, v, tol) {
		return fwd, true
	}
	return reverseEndSegs(segsForward(ring, j, i)), true // the other way, oriented fromP→toP
}

// insertSplits rebuilds the ring so every point in pts that lies interior to an edge splits it.
func insertSplits(segs []endSeg, pts []math.Point3, tol float64) []endSeg {
	var out []endSeg
	for _, s := range segs {
		out = append(out, splitSeg(s, pts, tol)...)
	}
	return out
}

// splitSeg splits one edge at every pts point lying interior to it, ordered along the edge.
func splitSeg(s endSeg, pts []math.Point3, tol float64) []endSeg {
	on := onSegPoints(s, pts, tol)
	if len(on) == 0 {
		return []endSeg{s}
	}
	chain := append(append([]math.Point3{s.from}, on...), s.to)
	out := make([]endSeg, 0, len(chain)-1)
	for k := 0; k+1 < len(chain); k++ {
		out = append(out, subSeg(s, chain[k], chain[k+1]))
	}
	return out
}

// onSegPoints returns the pts strictly interior to edge s, ordered by their parameter along it.
func onSegPoints(s endSeg, pts []math.Point3, tol float64) []math.Point3 {
	type keyed struct {
		p   math.Point3
		key float64
	}
	var found []keyed
	for _, p := range pts {
		if key, ok := segParam(s, p, tol); ok {
			found = append(found, keyed{p, key})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].key < found[j].key })
	out := make([]math.Point3, len(found))
	for i, f := range found {
		out[i] = f.p
	}
	return out
}

// segParam returns p's fractional parameter along edge s (0..1), ok only when p lies strictly
// interior to the edge (endpoints excluded), on its line or arc within tol.
func segParam(s endSeg, p math.Point3, tol float64) (float64, bool) {
	if float64(p.DistanceTo(s.from)) <= tol || float64(p.DistanceTo(s.to)) <= tol {
		return 0, false
	}
	if !s.arc {
		return lineParam(s.from, s.to, p, tol)
	}
	return arcParam(s.curve.(geom.Arc3d), p, tol)
}

// lineParam is p's parameter t∈(0,1) on segment a→b, ok only when p lies on the segment within tol.
func lineParam(a, b, p math.Point3, tol float64) (float64, bool) {
	d := a.VectorTo(b)
	l2 := float64(d.Dot(d))
	if l2 == 0 {
		return 0, false
	}
	t := float64(a.VectorTo(p).Dot(d)) / l2
	if t <= 0 || t >= 1 {
		return 0, false
	}
	if float64(a.TranslateBy(d.Scale(math.Scalar(t))).DistanceTo(p)) > tol {
		return 0, false
	}
	return t, true
}

// arcParam is p's fractional parameter on arc within (0,1), ok only when p lies on the arc's circle
// (radius within tol) and inside its sweep.
func arcParam(arc geom.Arc3d, p math.Point3, tol float64) (float64, bool) {
	if stdmath.Abs(float64(p.DistanceTo(arc.Center))-arc.Radius) > tol {
		return 0, false
	}
	w := arc.Center.VectorTo(p)
	bin := arc.Normal.Cross(arc.RefDir)
	raw := stdmath.Atan2(float64(w.Dot(bin)), float64(w.Dot(arc.RefDir.AsVector())))
	delta := wrapToSweep(raw-arc.StartAngle, arc.SweepAngle)
	frac := delta / arc.SweepAngle
	if frac <= 0 || frac >= 1 {
		return 0, false
	}
	return frac, true
}

// wrapToSweep brings the angular offset delta into the interval swept by sweep: [0,2π) for a
// positive sweep, (−2π,0] for a negative one, so delta/sweep lands in [0,1) for an on-arc point.
func wrapToSweep(delta, sweep float64) float64 {
	for delta < 0 {
		delta += 2 * stdmath.Pi
	}
	for delta >= 2*stdmath.Pi {
		delta -= 2 * stdmath.Pi
	}
	if sweep < 0 {
		delta -= 2 * stdmath.Pi
	}
	return delta
}

// subSeg returns the portion of edge s between from and to (both on s): a straight sub-segment, or
// an exact Arc3d sub-arc through the circle's angular midpoint (never a chord).
func subSeg(s endSeg, from, to math.Point3) endSeg {
	if !s.arc {
		return endSeg{from: from, to: to}
	}
	arc := s.curve.(geom.Arc3d)
	mid := arcMidBetween(arc.Center, arc.Radius, from, to)
	sub, err := geom.Arc3dByThreePoints(from, mid, to)
	if err != nil {
		return endSeg{from: from, to: to}
	}
	return endSeg{from: from, to: to, curve: sub, mid: mid, arc: true}
}

// arcMidBetween is the point on the circle (centre, radius) on the shorter arc between from and to —
// the bisector direction unit(from̂+tô) scaled to the radius. Used to re-fit a split sub-arc.
func arcMidBetween(center math.Point3, radius float64, from, to math.Point3) math.Point3 {
	a, b := center.VectorTo(from), center.VectorTo(to)
	bis, err := math.UnitVector3FromVector(a.Add(b))
	if err != nil {
		return from.Midpoint(to) // near-antipodal: degrade to the chord midpoint (not hit in Slice A)
	}
	return center.TranslateBy(bis.AsVector().Scale(math.Scalar(radius)))
}

// indexOfSegFrom returns the ring index whose seg starts at p, or −1.
func indexOfSegFrom(ring []endSeg, p math.Point3, tol float64) int {
	for i, s := range ring {
		if float64(s.from.DistanceTo(p)) <= tol {
			return i
		}
	}
	return -1
}

// segsForward collects the ring segments from index i up to (excluding) index j, cyclically.
func segsForward(ring []endSeg, i, j int) []endSeg {
	n := len(ring)
	var out []endSeg
	for k := i; k != j; k = (k + 1) % n {
		out = append(out, ring[k])
	}
	return out
}

// pathHasVertex reports whether v is one of the path's junctions (any seg endpoint within tol).
func pathHasVertex(path []endSeg, v math.Point3, tol float64) bool {
	for _, s := range path {
		if float64(s.from.DistanceTo(v)) <= tol || float64(s.to.DistanceTo(v)) <= tol {
			return true
		}
	}
	return false
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

// planeRayLoopExit returns the nearest forward point where the in-plane ray from p0 along dir leaves
// the original loop — the outer end of a straight ruling rail on a planar host.
func planeRayLoopExit(pl geom.Plane, segs []endSeg, p0 math.Point3, dir math.Vector3, tol float64) (math.Point3, bool) {
	ch := planeChart{pl}
	o2 := ch.to2(p0)
	d2 := o2.VectorTo(ch.to2(p0.TranslateBy(dir)))
	best, found := stdmath.Inf(1), false
	var bestPt math.Point3
	for _, s := range segs {
		if t, q, ok := rayEdgeHit2d(ch, s, o2, d2, tol); ok && t > tol && t < best {
			best, bestPt, found = t, q, true
		}
	}
	return bestPt, found
}

// rayEdgeHit2d intersects the chart ray (o2 + t·d2) with one loop edge, returning the forward hit's
// ray parameter and 3D point. Straight edges use a 2D line/segment solve; arc edges use the analytic
// 2D line/circle crossing filtered to the arc's sweep.
func rayEdgeHit2d(ch planeChart, s endSeg, o2 math.Point2, d2 math.Vector2, tol float64) (float64, math.Point3, bool) {
	if !s.arc {
		return raySegment2d(ch, o2, d2, s)
	}
	return rayArc2d(ch, o2, d2, s, tol)
}

// raySegment2d solves the ray o2+t·d2 against the straight edge s in the chart.
func raySegment2d(ch planeChart, o2 math.Point2, d2 math.Vector2, s endSeg) (float64, math.Point3, bool) {
	a2, b2 := ch.to2(s.from), ch.to2(s.to)
	e := a2.VectorTo(b2)
	denom := d2.Cross(e)
	if stdmath.Abs(float64(denom)) < 1e-15 {
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

// rayArc2d solves the ray against an arc edge in the chart, keeping the nearest forward crossing that
// lies inside the arc's sweep.
func rayArc2d(ch planeChart, o2 math.Point2, d2 math.Vector2, s endSeg, tol float64) (float64, math.Point3, bool) {
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

// footCircleLoopHits returns every point where the foot circle (centre, radius, in the host plane)
// crosses the original loop — the two ends of the through-arm's foot-bite arc (§B.5).
func footCircleLoopHits(pl geom.Plane, segs []endSeg, center math.Point3, radius, tol float64) []math.Point3 {
	ch := planeChart{pl}
	fc := geom.NewCircle2d(ch.to2(center), radius)
	var out []math.Point3
	for _, s := range segs {
		out = appendDistinct(out, edgeCircleHits2d(ch, s, fc, tol), tol)
	}
	return out
}

// edgeCircleHits2d crosses one loop edge with the foot circle in the chart (segment↔circle for a
// straight edge, circle↔circle filtered to the sweep for an arc edge).
func edgeCircleHits2d(ch planeChart, s endSeg, fc geom.Circle2d, tol float64) []math.Point3 {
	if !s.arc {
		seg2d := geom.NewLineSegment2d(ch.to2(s.from), ch.to2(s.to))
		return lift2d(ch, geom.SegmentCircle2dIntersection(seg2d, fc, tol))
	}
	arc := s.curve.(geom.Arc3d)
	ec := geom.NewCircle2d(ch.to2(arc.Center), arc.Radius)
	var out []math.Point3
	for _, p2 := range geom.Circle2dCircle2dIntersection(fc, ec, tol) {
		if q := ch.to3(p2); onArc3d(arc, q, tol) {
			out = append(out, q)
		}
	}
	return out
}

// onArc3d reports whether p lies on arc's circle and inside its sweep (endpoints allowed).
func onArc3d(arc geom.Arc3d, p math.Point3, tol float64) bool {
	if stdmath.Abs(float64(p.DistanceTo(arc.Center))-arc.Radius) > tol {
		return false
	}
	w := arc.Center.VectorTo(p)
	bin := arc.Normal.Cross(arc.RefDir)
	raw := stdmath.Atan2(float64(w.Dot(bin)), float64(w.Dot(arc.RefDir.AsVector())))
	frac := wrapToSweep(raw-arc.StartAngle, arc.SweepAngle) / arc.SweepAngle
	return frac >= -1e-9 && frac <= 1+1e-9
}

// lift2d maps chart points back to 3D on the plane.
func lift2d(ch planeChart, pts []math.Point2) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[i] = ch.to3(p)
	}
	return out
}

// appendDistinct appends the points not already within tol of an existing one (dedup crossings that
// two adjacent edges share at a vertex).
func appendDistinct(dst, src []math.Point3, tol float64) []math.Point3 {
	for _, p := range src {
		if matchPoint(dst, p, tol) < 0 {
			dst = append(dst, p)
		}
	}
	return dst
}
