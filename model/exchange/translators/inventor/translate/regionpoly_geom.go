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
