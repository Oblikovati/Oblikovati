// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/math"
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

// interiorPoint2D returns a point strictly inside the region (in the outer loop, outside
// holes). It tries the vertex centroid, then triangle-fan centroids — robust for the
// convex and mildly-non-convex regions the boolean produces.
func interiorPoint2D(r Face2D) (math.Point2, bool) {
	inHoles := func(p math.Point2) bool {
		for _, h := range r.Holes {
			if pointInPolygon2D(p, h) {
				return true
			}
		}
		return false
	}
	ok := func(p math.Point2) bool { return pointInPolygon2D(p, r.Outer) && !inHoles(p) }
	if c := centroid2D(r.Outer); ok(c) {
		return c, true
	}
	for i := 1; i+1 < len(r.Outer); i++ {
		c := centroid2D([]math.Point2{r.Outer[0], r.Outer[i], r.Outer[i+1]})
		if ok(c) {
			return c, true
		}
	}
	return math.Point2{}, false
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
