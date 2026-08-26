// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Oriented minimum range box (M07-F07, Oblikovati/Oblikovati#630). The classic
// hull-flush approximation: for every convex-hull face normal, take that
// normal as one box axis, run 2D rotating calipers on the projection for the
// other two, and keep the smallest volume. Exact when the minimum box is flush
// with a hull face (always true for polyhedra, per O'Rourke); curved bodies
// are sampled at edge density, so the box is sample-accurate.

// The oriented box value type is the ONE in-module [math.OrientedBox] (audit B9, #1620:
// this file used to declare a second OrientedBox — corner + three edge vectors — a plain
// duplicate; it now builds the result in that corner+edges form and hands it to
// math.NewOrientedBoxFromEdges, so there is a single oriented-box type).

// OrientedMinimumRangeBox computes the minimal oriented box over the body's
// sampled geometry (vertices + 32 samples per curved edge).
//
// Example: obb := ops.OrientedMinimumRangeBox(body)
func OrientedMinimumRangeBox(b *topo.Body) (math.OrientedBox, error) {
	pts := bodySamplePoints(b)
	hull, err := ConvexHull(pts, "obb")
	if err != nil {
		return math.OrientedBox{}, err
	}
	// Only hull vertices matter for extents — they dominate every interior
	// sample, and there are far fewer of them than raw edge samples.
	hullPts := make([]math.Point3, 0, len(hull.Vertices()))
	for _, v := range hull.Vertices() {
		hullPts = append(hullPts, v.Point())
	}
	best, bestVol := math.OrientedBox{}, stdmath.Inf(1)
	for _, f := range hull.Faces() {
		box, ok := boxFlushWithFace(f, hullPts)
		if ok && box.Volume() < bestVol {
			best, bestVol = box, box.Volume()
		}
	}
	return best, nil
}

// bodySamplePoints collects vertices plus dense edge samples (curved
// silhouettes bulge past their vertices).
func bodySamplePoints(b *topo.Body) []math.Point3 {
	var pts []math.Point3
	for _, v := range b.Vertices() {
		pts = append(pts, v.Point())
	}
	for _, e := range b.Edges() {
		c := e.Geometry()
		lo, hi := c.Domain()
		for i := 0; i <= 32; i++ {
			pts = append(pts, c.PointAt(lo+(hi-lo)*float64(i)/32))
		}
	}
	for _, w := range b.Wires() {
		for _, u := range w.Uses() {
			c := u.Edge.Geometry()
			lo, hi := c.Domain()
			for i := 0; i <= 32; i++ {
				pts = append(pts, c.PointAt(lo+(hi-lo)*float64(i)/32))
			}
		}
	}
	return pts
}

// boxFlushWithFace builds the best box whose base plane is flush with the hull
// face: axis w = face normal, in-plane axes from 2D rotating calipers.
func boxFlushWithFace(f *topo.Face, pts []math.Point3) (math.OrientedBox, bool) {
	w := f.Geometry().NormalAt(0, 0)
	l := float64(w.Length())
	if l == 0 {
		return math.OrientedBox{}, false
	}
	w = w.Scale(math.Scalar(1 / l))
	u := perpUnit(w)
	v := w.Cross(u)
	proj := make([]math.Point2, len(pts))
	wLo, wHi := stdmath.Inf(1), stdmath.Inf(-1)
	for i, p := range pts {
		d := p.AsVector()
		proj[i] = math.P2(d.Dot(u), d.Dot(v))
		h := float64(d.Dot(w))
		wLo, wHi = min(wLo, h), max(wHi, h)
	}
	rect, ok := minAreaRectangle(proj)
	if !ok {
		return math.OrientedBox{}, false
	}
	return liftRectangle(rect, u, v, w, wLo, wHi), true
}

// perpUnit returns a unit vector perpendicular to w.
func perpUnit(w math.Vector3) math.Vector3 {
	ref := math.V3(1, 0, 0)
	if stdmath.Abs(float64(w.X)) > 0.9 {
		ref = math.V3(0, 1, 0)
	}
	u := w.Cross(ref)
	return u.Scale(math.Scalar(1 / float64(u.Length())))
}

// rect2 is a rotated 2D rectangle: corner + two perpendicular edge vectors.
type rect2 struct {
	corner math.Point2
	e1, e2 math.Vector2
}

// liftRectangle assembles the 3D box from the in-plane rectangle and the normal extent,
// as a corner and three orthogonal edge vectors handed to [math.NewOrientedBoxFromEdges]
// (the single oriented-box value type).
func liftRectangle(r rect2, u, v, w math.Vector3, wLo, wHi float64) math.OrientedBox {
	corner := math.P3(0, 0, 0).
		TranslateBy(u.Scale(r.corner.X)).
		TranslateBy(v.Scale(r.corner.Y)).
		TranslateBy(w.Scale(math.Scalar(wLo)))
	lift := func(e math.Vector2) math.Vector3 {
		return u.Scale(e.X).Add(v.Scale(e.Y))
	}
	return math.NewOrientedBoxFromEdges(corner, lift(r.e1), lift(r.e2), w.Scale(math.Scalar(wHi-wLo)))
}

// minAreaRectangle runs rotating calipers over the 2D convex hull: the minimum
// rectangle is flush with a hull edge.
func minAreaRectangle(pts []math.Point2) (rect2, bool) {
	hull := convexHull2D(pts)
	if len(hull) < 3 {
		return degenerateRect(hull)
	}
	best, bestArea := rect2{}, stdmath.Inf(1)
	for i := range hull {
		j := (i + 1) % len(hull)
		dir := unit2(hull[i].VectorTo(hull[j]))
		if float64(dir.Length()) == 0 {
			continue
		}
		r := rectAlong(hull, dir)
		if a := float64(r.e1.Length()) * float64(r.e2.Length()); a < bestArea {
			best, bestArea = r, a
		}
	}
	return best, bestArea < stdmath.Inf(1)
}

// rectAlong bounds the hull in the (dir, perp) frame.
func rectAlong(hull []math.Point2, dir math.Vector2) rect2 {
	perp := perpLeft(dir)
	xLo, xHi := stdmath.Inf(1), stdmath.Inf(-1)
	yLo, yHi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, p := range hull {
		x, y := float64(p.AsVector().Dot(dir)), float64(p.AsVector().Dot(perp))
		xLo, xHi = min(xLo, x), max(xHi, x)
		yLo, yHi = min(yLo, y), max(yHi, y)
	}
	corner := dir.Scale(math.Scalar(xLo)).Add(perp.Scale(math.Scalar(yLo))).AsPoint()
	return rect2{
		corner: corner,
		e1:     dir.Scale(math.Scalar(xHi - xLo)),
		e2:     perp.Scale(math.Scalar(yHi - yLo)),
	}
}

// degenerateRect covers hulls of fewer than 3 points (a segment or point).
func degenerateRect(hull []math.Point2) (rect2, bool) {
	switch len(hull) {
	case 2:
		return rect2{corner: hull[0], e1: hull[0].VectorTo(hull[1]), e2: math.V2(0, 0)}, true
	case 1:
		return rect2{corner: hull[0]}, true
	default:
		return rect2{}, false
	}
}

// convexHull2D is Andrew's monotone chain.
func convexHull2D(pts []math.Point2) []math.Point2 {
	sorted := append([]math.Point2(nil), pts...)
	sortPoints2(sorted)
	sorted = dedupePoints2(sorted)
	if len(sorted) <= 2 {
		return sorted
	}
	lower := halfHull(sorted)
	upper := halfHull(reversed2(sorted))
	return append(lower[:len(lower)-1], upper[:len(upper)-1]...)
}

func halfHull(pts []math.Point2) []math.Point2 {
	var h []math.Point2
	for _, p := range pts {
		for len(h) >= 2 && float64(h[len(h)-2].VectorTo(h[len(h)-1]).Cross(h[len(h)-2].VectorTo(p))) <= 0 {
			h = h[:len(h)-1]
		}
		h = append(h, p)
	}
	return h
}

func sortPoints2(pts []math.Point2) {
	sort.Slice(pts, func(i, j int) bool { return lessPoint2(pts[i], pts[j]) })
}

func lessPoint2(a, b math.Point2) bool {
	if a.X != b.X {
		return a.X < b.X
	}
	return a.Y < b.Y
}

func dedupePoints2(pts []math.Point2) []math.Point2 {
	out := pts[:0]
	for i, p := range pts {
		if i == 0 || float64(p.DistanceTo(out[len(out)-1])) > 1e-12 { // tol:numeric — float-noise-level consecutive-point dedup
			out = append(out, p)
		}
	}
	return out
}

func reversed2(pts []math.Point2) []math.Point2 {
	out := make([]math.Point2, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}
