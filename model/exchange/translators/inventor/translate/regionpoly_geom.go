// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
)

func pt2(p ipt.Point2D) math.Point2 { return math.P2(math.Scalar(p.X), math.Scalar(p.Y)) }

func last(p []math.Point2) math.Point2 { return p[len(p)-1] }

// sameXY reports whether two boundary points coincide. (The package's samePt compares a solved
// sketch point against a decoded one — a different question.)
func sameXY(a, b math.Point2) bool {
	return stdmath.Hypot(float64(a.X-b.X), float64(a.Y-b.Y)) <= polyTol
}

func reversePts(p []math.Point2) []math.Point2 {
	out := make([]math.Point2, len(p))
	for i, q := range p {
		out[len(p)-1-i] = q
	}
	return out
}

// angleOn is p's angle about c, in [0, 2pi).
func angleOn(cx, cy float64, p ipt.Point2D) float64 {
	a := stdmath.Atan2(p.Y-cy, p.X-cx)
	if a < 0 {
		a += 2 * stdmath.Pi
	}
	return a
}

// arcPolyline samples a trimmed arc from Start to End, COUNTER-CLOCKWISE. The direction is not a
// choice here: ipt.trimCircleEdge already ordered Start/End by the file's posDir, so walking CCW
// from Start reproduces exactly the span the face uses — including an obround's 180° halves, where
// the two candidates are the same length and only the direction tells them apart.
func arcPolyline(a ipt.Arc) []math.Point2 {
	s := angleOn(a.Center.X, a.Center.Y, a.Start)
	e := angleOn(a.Center.X, a.Center.Y, a.End)
	span := e - s
	for span <= 0 {
		span += 2 * stdmath.Pi
	}
	n := int(stdmath.Ceil(float64(arcSegments) * span / (2 * stdmath.Pi)))
	if n < 2 {
		n = 2
	}
	out := make([]math.Point2, 0, n+1)
	for i := 0; i <= n; i++ {
		t := s + span*float64(i)/float64(n)
		out = append(out, math.P2(
			math.Scalar(a.Center.X+a.Radius*stdmath.Cos(t)),
			math.Scalar(a.Center.Y+a.Radius*stdmath.Sin(t))))
	}
	return out
}

// ellipseParam returns the eccentric angle t of a point on the ellipse, i.e. the t for which the
// point is Center + MajorR*cos(t)*u + MinorR*sin(t)*v with u the major axis and v its CCW normal.
func ellipseParam(e ipt.EllipseArc, p ipt.Point2D) float64 {
	ux, uy := e.MajorAxis.X, e.MajorAxis.Y
	dx, dy := p.X-e.Center.X, p.Y-e.Center.Y
	x := dx*ux + dy*uy  // component along major axis
	y := -dx*uy + dy*ux // component along minor axis (v = (-uy, ux))
	return stdmath.Atan2(y/e.MinorR, x/e.MajorR)
}

// ellipsePoint maps an eccentric angle back to a boundary point.
func ellipsePoint(e ipt.EllipseArc, t float64) math.Point2 {
	ux, uy := e.MajorAxis.X, e.MajorAxis.Y
	c, s := stdmath.Cos(t), stdmath.Sin(t)
	x := e.Center.X + e.MajorR*c*ux - e.MinorR*s*uy
	y := e.Center.Y + e.MajorR*c*uy + e.MinorR*s*ux
	return math.P2(math.Scalar(x), math.Scalar(y))
}

// ellipseArcPolyline samples an ellipse boundary edge. When Start == End the loop names the whole
// ellipse (sampled full turn); otherwise it walks the eccentric-angle span CCW from Start to End —
// the same convention arcPolyline uses, since ipt.trimEllipseEdge ordered Start/End by posDir.
func ellipseArcPolyline(e ipt.EllipseArc) []math.Point2 {
	if e.Start == (ipt.Point2D{}) && e.End == (ipt.Point2D{}) {
		out := make([]math.Point2, arcSegments)
		for i := range out {
			out[i] = ellipsePoint(e, 2*stdmath.Pi*float64(i)/float64(arcSegments))
		}
		return out
	}
	s, en := ellipseParam(e, e.Start), ellipseParam(e, e.End)
	span := en - s
	for span <= 0 {
		span += 2 * stdmath.Pi
	}
	n := int(stdmath.Ceil(float64(arcSegments) * span / (2 * stdmath.Pi)))
	if n < 2 {
		n = 2
	}
	out := make([]math.Point2, 0, n+1)
	for i := 0; i <= n; i++ {
		out = append(out, ellipsePoint(e, s+span*float64(i)/float64(n)))
	}
	return out
}

// sampleWholeCircle samples a circle the loop names whole (a bore's hole loop).
func sampleWholeCircle(c ipt.Circle) []math.Point2 {
	out := make([]math.Point2, arcSegments)
	for i := range out {
		t := 2 * stdmath.Pi * float64(i) / float64(arcSegments)
		out[i] = math.P2(
			math.Scalar(c.Center.X+c.Radius*stdmath.Cos(t)),
			math.Scalar(c.Center.Y+c.Radius*stdmath.Sin(t)))
	}
	return out
}

// polygonArea is the absolute area a closed polygon encloses (shoelace).
func polygonArea(poly []math.Point2) float64 {
	a := 0.0
	for i := range poly {
		j := (i + 1) % len(poly)
		a += float64(poly[i].X)*float64(poly[j].Y) - float64(poly[j].X)*float64(poly[i].Y)
	}
	if a < 0 {
		a = -a
	}
	return a / 2
}

// smallestContainingArea returns the area of the smallest region outer loop that contains q, and
// whether any does. The smallest (tightest) loop is the one a cell inside q must fit — where outer
// loops overlap, a cell can belong to at most the smaller of them.
func smallestContainingArea(q math.Point2, outers [][]math.Point2) (float64, bool) {
	best, found := 0.0, false
	for _, o := range outers {
		if !pointInPoly(q, o) {
			continue
		}
		if a := polygonArea(o); !found || a < best {
			best, found = a, true
		}
	}
	return best, found
}

// pointInPoly is the even-odd containment test.
func pointInPoly(q math.Point2, poly []math.Point2) bool {
	in := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		yi, yj := float64(poly[i].Y), float64(poly[j].Y)
		if (yi > float64(q.Y)) == (yj > float64(q.Y)) {
			continue
		}
		x := float64(poly[i].X) + (float64(q.Y)-yi)/(yj-yi)*(float64(poly[j].X)-float64(poly[i].X))
		if float64(q.X) < x {
			in = !in
		}
	}
	return in
}

// interiorPointAvoiding returns a point inside poly that also satisfies keep — used to place a test
// point in a cell's material rather than in one of its own holes. A centroid is NOT safe (a concave
// cell's centroid can fall outside it, and an annulus's lands in its hole), so this walks for a
// vertex whose i/i+2 midpoint qualifies and only then falls back to the centroid.
func interiorPointAvoiding(poly []math.Point2, keep func(math.Point2) bool) (math.Point2, bool) {
	if len(poly) < 3 {
		return math.Point2{}, false
	}
	for i := range poly {
		a, b := poly[i], poly[(i+2)%len(poly)]
		mid := math.P2((a.X+b.X)/2, (a.Y+b.Y)/2)
		if pointInPoly(mid, poly) && keep(mid) {
			return mid, true
		}
	}
	var sx, sy float64
	for _, p := range poly {
		sx += float64(p.X)
		sy += float64(p.Y)
	}
	c := math.P2(math.Scalar(sx/float64(len(poly))), math.Scalar(sy/float64(len(poly))))
	return c, pointInPoly(c, poly) && keep(c)
}
