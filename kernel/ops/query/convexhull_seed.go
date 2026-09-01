// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"
	"strconv"

	"oblikovati.org/kernel/mesh"
	"oblikovati.org/math"
)

// initialTetra picks four affinely-independent seed points: two extremes along the widest
// spread, the point farthest from their line, then the point farthest from that plane. It
// returns ok=false when the cloud is collinear or coplanar (no non-degenerate tetrahedron).
func initialTetra(pts []math.Point3) ([4]int, bool) {
	i0, i1 := farthestPair(pts)
	if i0 == i1 {
		return [4]int{}, false
	}
	i2 := farthestFromLine(pts, i0, i1)
	if i2 < 0 {
		return [4]int{}, false
	}
	i3 := farthestFromPlane(pts, i0, i1, i2)
	if i3 < 0 {
		return [4]int{}, false
	}
	return [4]int{i0, i1, i2, i3}, true
}

// farthestPair returns the indices of the two points farthest apart along each axis extreme —
// a cheap, robust diameter approximation good enough to seed the hull.
func farthestPair(pts []math.Point3) (int, int) {
	minX, maxX := 0, 0
	for i, p := range pts {
		if p.X < pts[minX].X {
			minX = i
		}
		if p.X > pts[maxX].X {
			maxX = i
		}
	}
	a, b, best := minX, maxX, pts[minX].DistanceTo(pts[maxX])
	for i := range pts { // refine: farthest point from minX overall
		if d := pts[minX].DistanceTo(pts[i]); d > best {
			a, b, best = minX, i, d
		}
	}
	return a, b
}

// farthestFromLine returns the point with the largest perpendicular distance to line i0i1
// (−1 if every point is on the line).
func farthestFromLine(pts []math.Point3, i0, i1 int) int {
	d := pts[i0].VectorTo(pts[i1])
	dl := d.Length()
	if dl == 0 {
		return -1
	}
	best, bestDist := -1, 1e-12*dl
	for i := range pts {
		area := pts[i0].VectorTo(pts[i]).Cross(d).Length() // 2·triangle area = base·height
		if area/dl > bestDist {
			best, bestDist = i, area/dl
		}
	}
	return best
}

// farthestFromPlane returns the point with the largest absolute distance to plane i0i1i2
// (−1 if every point is coplanar).
func farthestFromPlane(pts []math.Point3, i0, i1, i2 int) int {
	n := pts[i0].VectorTo(pts[i1]).Cross(pts[i0].VectorTo(pts[i2]))
	nl := n.Length()
	if nl == 0 {
		return -1
	}
	best, bestDist := -1, 1e-12*nl
	for i := range pts {
		if d := stdmath.Abs(pts[i0].VectorTo(pts[i]).Dot(n)); d > bestDist {
			// Guard with the exact predicate so a near-coplanar seed is rejected.
			if orient3p(pts[i0], pts[i1], pts[i2], pts[i]) != 0 {
				best, bestDist = i, d
			}
		}
	}
	return best
}

// dedupPoints removes points that coincide to the weld grid, preserving first-seen order so
// the hull is deterministic.
func dedupPoints(points []math.Point3, grid float64) []math.Point3 {
	seen := map[[3]int64]bool{}
	out := make([]math.Point3, 0, len(points))
	for _, p := range points {
		k := [3]int64{mesh.Quantize(p.X, grid), mesh.Quantize(p.Y, grid), mesh.Quantize(p.Z, grid)}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, p)
	}
	return out
}

// boundsDiagonal returns the diagonal length of the points' axis-aligned bounding box — the
// scale used to set a relative visibility tolerance.
func boundsDiagonal(pts []math.Point3) float64 {
	lo, hi := pts[0], pts[0]
	for _, p := range pts {
		lo = math.P3(stdmath.Min(lo.X, p.X), stdmath.Min(lo.Y, p.Y), stdmath.Min(lo.Z, p.Z))
		hi = math.P3(stdmath.Max(hi.X, p.X), stdmath.Max(hi.Y, p.Y), stdmath.Max(hi.Z, p.Z))
	}
	d := lo.DistanceTo(hi)
	if d == 0 {
		return 1
	}
	return d
}

func itoa(n int) string { return strconv.Itoa(n) }
