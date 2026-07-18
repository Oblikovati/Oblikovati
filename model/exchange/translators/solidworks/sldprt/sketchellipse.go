// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"math"
	"sort"
)

// Ellipse is a full ellipse: its centre, the unit major-axis direction, and the two semi-axis
// lengths (metres). MajorRadius >= MinorRadius by construction.
type Ellipse struct {
	Center         Point
	MajorX, MajorY float64 // unit major-axis direction
	MajorRadius    float64
	MinorRadius    float64
}

// ellipseAxisEps is the tolerance (metres) for reflecting an axis endpoint about the centre and for
// the axis-perpendicularity check — well below any real sketch feature, above f64 arithmetic noise.
const ellipseAxisEps = 1e-9

// ellipseFromPoints recovers an ellipse from its cached points. SolidWorks caches an ellipse as its
// centre plus the four axis endpoints (±major, ±minor) and a rim parameter point. The centre is the
// point about which two perpendicular endpoint pairs are symmetric; the longer pair gives the major
// axis. Returns ok=false when the points do not form that cross, so a non-ellipse region is rejected.
func ellipseFromPoints(pts []Point) (Ellipse, bool) {
	for _, c := range pts {
		reps := symmetricPairs(c, pts)
		if len(reps) < 2 {
			continue
		}
		maj, mnr := reps[0], reps[1] // reps are sorted longest-first
		if !perpendicularAbout(c, maj, mnr) {
			continue
		}
		majR := dist(maj, c)
		return Ellipse{
			Center:      c,
			MajorX:      (maj.X - c.X) / majR,
			MajorY:      (maj.Y - c.Y) / majR,
			MajorRadius: majR,
			MinorRadius: dist(mnr, c),
		}, true
	}
	return Ellipse{}, false
}

// symmetricPairs returns one representative endpoint per pair of distinct points symmetric about c
// (a point P whose reflection 2c−P is also present), sorted by distance from c descending.
func symmetricPairs(c Point, pts []Point) []Point {
	var reps []Point
	seen := map[Point]bool{}
	for _, p := range pts {
		if p == c || seen[p] {
			continue
		}
		refl := Point{X: 2*c.X - p.X, Y: 2*c.Y - p.Y}
		if refl == p {
			continue // p == c handled above; a zero-length axis is not a pair
		}
		if containsPoint(pts, refl) {
			reps = append(reps, p)
			seen[p], seen[refl] = true, true
		}
	}
	sort.Slice(reps, func(i, j int) bool { return dist(reps[i], c) > dist(reps[j], c) })
	return reps
}

// containsPoint reports whether pts holds a point within ellipseAxisEps of q.
func containsPoint(pts []Point, q Point) bool {
	for _, p := range pts {
		if math.Abs(p.X-q.X) <= ellipseAxisEps && math.Abs(p.Y-q.Y) <= ellipseAxisEps {
			return true
		}
	}
	return false
}

// perpendicularAbout reports whether the vectors c→a and c→b are perpendicular (the ellipse axes).
func perpendicularAbout(c, a, b Point) bool {
	return math.Abs((a.X-c.X)*(b.X-c.X)+(a.Y-c.Y)*(b.Y-c.Y)) <= ellipseAxisEps
}
