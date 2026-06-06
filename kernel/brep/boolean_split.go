// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// splitFace splits a face by its imprint segments via the 2D arrangement and returns the
// material sub-faces (regions inside the original face), each carried back to 3D with an
// interior point for classification. A face with no imprints yields itself unchanged.
func splitFace(f planarFace, imprints [][2]math.Point3) []subFace {
	segs := faceBoundarySegments(f)
	for _, s := range imprints {
		segs = append(segs, [2]math.Point2{to2D(f.plane, s[0]), to2D(f.plane, s[1])})
	}
	var out []subFace
	for _, r := range Arrange(segs) {
		ip, ok := interiorPoint2D(r)
		if !ok || !pointInFace2D(ip, f) {
			continue // a region outside the face's material (e.g. inside one of its holes)
		}
		sf := subFace{normal: f.normal, point: to3D(f.plane, ip), outer: ring3D(f.plane, r.Outer)}
		for _, h := range r.Holes {
			sf.holes = append(sf.holes, ring3D(f.plane, h))
		}
		out = append(out, sf)
	}
	return out
}

// faceBoundarySegments returns a face's loop edges as 2D segments in its plane.
func faceBoundarySegments(f planarFace) [][2]math.Point2 {
	var segs [][2]math.Point2
	for _, ring := range f.loops {
		n := len(ring)
		for i := 0; i < n; i++ {
			segs = append(segs, [2]math.Point2{to2D(f.plane, ring[i]), to2D(f.plane, ring[(i+1)%n])})
		}
	}
	return segs
}

// ring3D lifts a 2D loop back to model space on the given plane.
func ring3D(pl geom.Plane, loop []math.Point2) []math.Point3 {
	out := make([]math.Point3, len(loop))
	for i, q := range loop {
		out[i] = to3D(pl, q)
	}
	return out
}

// interiorPoint2D returns a point STRICTLY inside the region (inside the outer loop,
// outside holes). It probes just inside each outer edge (the midpoint nudged inward by a
// small fraction of the edge length): such points hug the boundary, so unlike an ear
// centroid they never fall in a central hole (a square frame) and never land on an edge
// (which would make the inside/outside classification ambiguous). An ear-centroid pass is
// the fallback for slivers where no edge probe lands clear.
func interiorPoint2D(r Face2D) (math.Point2, bool) {
	poly := r.Outer
	n := len(poly)
	if n < 3 {
		return math.Point2{}, false
	}
	for i := 0; i < n; i++ {
		a, b := poly[i], poly[(i+1)%n]
		e := a.VectorTo(b)
		mid := math.P2((a.X+b.X)/2, (a.Y+b.Y)/2)
		left := math.V2(-e.Y, e.X) // interior side of a CCW loop (magnitude = |edge|)
		for _, f := range []float64{1e-3, 1e-2, 0.05, 0.2} {
			p := mid.TranslateBy(left.Scale(f))
			if pointInPolygon2D(p, poly) && !inHoles2D(p, r.Holes) {
				return p, true
			}
		}
	}
	for i := 0; i < n; i++ { // fallback: ear centroids
		prev, cur, next := poly[(i-1+n)%n], poly[i], poly[(i+1)%n]
		if turn2D(prev, cur, next) <= arrTol || !earEmpty(poly, i) {
			continue
		}
		if c := centroid2D([]math.Point2{prev, cur, next}); !inHoles2D(c, r.Holes) {
			return c, true
		}
	}
	return math.Point2{}, false
}

// turn2D returns the signed turn (cross product) at b going a→b→c (>0 = left/convex CCW).
func turn2D(a, b, c math.Point2) float64 {
	return b.VectorTo(c).Cross(a.VectorTo(b)) * -1
}

// earEmpty reports whether the triangle at vertex i (prev,cur,next) contains no other
// vertex of the polygon — the ear test.
func earEmpty(poly []math.Point2, i int) bool {
	n := len(poly)
	a, b, c := poly[(i-1+n)%n], poly[i], poly[(i+1)%n]
	for j := 0; j < n; j++ {
		if j == i || j == (i-1+n)%n || j == (i+1)%n {
			continue
		}
		if pointInTriangle2D(poly[j], a, b, c) {
			return false
		}
	}
	return true
}

// pointInTriangle2D reports whether p is inside triangle abc (inclusive of edges).
func pointInTriangle2D(p, a, b, c math.Point2) bool {
	d1 := turn2D(a, b, p)
	d2 := turn2D(b, c, p)
	d3 := turn2D(c, a, p)
	hasNeg := d1 < -arrTol || d2 < -arrTol || d3 < -arrTol
	hasPos := d1 > arrTol || d2 > arrTol || d3 > arrTol
	return !(hasNeg && hasPos)
}

// inHoles2D reports whether p lies in any hole loop.
func inHoles2D(p math.Point2, holes [][]math.Point2) bool {
	for _, h := range holes {
		if pointInPolygon2D(p, h) {
			return true
		}
	}
	return false
}

// centroid2D returns the average of a point set.
func centroid2D(pts []math.Point2) math.Point2 {
	var sx, sy float64
	for _, p := range pts {
		sx, sy = sx+p.X, sy+p.Y
	}
	n := float64(len(pts))
	return math.P2(sx/n, sy/n)
}
