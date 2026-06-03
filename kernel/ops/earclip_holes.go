// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/math/predicate"
)

// mergeHoles bridges each hole loop into the outer loop, producing a single simple polygon
// (the 2D copy for ear-clipping plus its 3D pre-image in lockstep) whose triangulation
// equals the triangulation of the holed face. It is the textbook bridge construction
// (Eberly, "Triangulation by Ear Clipping" §4): for each hole, pick its rightmost vertex,
// find a mutually visible outer vertex, and splice the (CW) hole into the (CCW) outer along
// a doubled bridge edge. Holes are processed right-to-left so an already-bridged hole is
// just part of the growing outer for the next one.
func mergeHoles(outer2D []math.Point2, outer3D []math.Point3, holes2D [][]math.Point2, holes3D [][]math.Point3) ([]math.Point2, []math.Point3) {
	o2, o3 := orient(outer2D, outer3D, true) // outer CCW
	order := holesByRightmost(holes2D)
	for _, hi := range order {
		h2, h3 := orient(holes2D[hi], holes3D[hi], false) // hole CW
		ph := rightmost(h2)
		mo := findVisibleVertex(o2, h2[ph])
		o2, o3 = spliceBridge(o2, o3, mo, h2, h3, ph)
	}
	return o2, o3
}

// holesByRightmost orders hole indices by their rightmost vertex's X (descending), so the
// hole nearest the outer's right boundary bridges first.
func holesByRightmost(holes [][]math.Point2) []int {
	order := make([]int, len(holes))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return holes[order[a]][rightmost(holes[order[a]])].X > holes[order[b]][rightmost(holes[order[b]])].X
	})
	return order
}

// rightmost returns the index of the loop's vertex with maximum X (ties broken by Y).
func rightmost(loop []math.Point2) int {
	best := 0
	for i, p := range loop {
		if p.X > loop[best].X || (p.X == loop[best].X && p.Y > loop[best].Y) {
			best = i
		}
	}
	return best
}

// orient returns the loop (2D and its 3D pre-image) reordered to the requested winding
// (ccw=true → counter-clockwise).
func orient(loop2D []math.Point2, loop3D []math.Point3, ccw bool) ([]math.Point2, []math.Point3) {
	if (signedArea(loop2D) > 0) == ccw {
		return loop2D, loop3D
	}
	r2 := make([]math.Point2, len(loop2D))
	r3 := make([]math.Point3, len(loop3D))
	for i := range loop2D {
		r2[len(loop2D)-1-i] = loop2D[i]
		r3[len(loop3D)-1-i] = loop3D[i]
	}
	return r2, r3
}

// spliceBridge inserts a hole into the outer loop after outer vertex mo: the doubled bridge
// runs outer[mo] → hole[ph] → (whole hole) → hole[ph] → outer[mo], turning the annulus into
// one simple polygon.
func spliceBridge(o2 []math.Point2, o3 []math.Point3, mo int, h2 []math.Point2, h3 []math.Point3, ph int) ([]math.Point2, []math.Point3) {
	n2 := make([]math.Point2, 0, len(o2)+len(h2)+2)
	n3 := make([]math.Point3, 0, len(o3)+len(h3)+2)
	n2 = append(n2, o2[:mo+1]...)
	n3 = append(n3, o3[:mo+1]...)
	for k := 0; k <= len(h2); k++ { // ph..around..ph (inclusive, so the hole closes)
		j := (ph + k) % len(h2)
		n2 = append(n2, h2[j])
		n3 = append(n3, h3[j])
	}
	n2 = append(n2, o2[mo])
	n3 = append(n3, o3[mo])
	n2 = append(n2, o2[mo+1:]...)
	n3 = append(n3, o3[mo+1:]...)
	return n2, n3
}

// findVisibleVertex returns the index of an outer-loop vertex mutually visible from p (a
// hole's rightmost vertex): cast a ray from p toward +X, take the hit edge's higher-X
// endpoint M, and if any reflex outer vertex lies inside triangle (p, hit, M) blocking the
// view, fall back to the blocker whose direction from p is closest to the +X ray.
func findVisibleVertex(outer []math.Point2, p math.Point2) int {
	mIdx, hit := rayHitEdge(outer, p)
	if mIdx < 0 {
		return 0
	}
	m := outer[mIdx]
	best, bestAng, bestD := mIdx, stdmath.Inf(1), stdmath.Inf(1)
	for i := range outer {
		if i == mIdx || !isReflex(outer, i) {
			continue
		}
		v := outer[i]
		if !inTriAny(v, p, hit, m) {
			continue
		}
		ang := stdmath.Abs(stdmath.Atan2(v.Y-p.Y, v.X-p.X))
		if d := p.VectorTo(v).LengthSquared(); ang < bestAng || (ang == bestAng && d < bestD) {
			best, bestAng, bestD = i, ang, d
		}
	}
	return best
}

// rayHitEdge casts a ray from p toward +X and returns the index of the hit edge's higher-X
// endpoint and the intersection point (mIdx = −1 if the ray hits nothing).
func rayHitEdge(outer []math.Point2, p math.Point2) (int, math.Point2) {
	n := len(outer)
	mIdx, bestD, hit := -1, stdmath.Inf(1), math.Point2{}
	for i := 0; i < n; i++ {
		a, b := outer[i], outer[(i+1)%n]
		if (a.Y > p.Y) == (b.Y > p.Y) {
			continue // edge does not straddle the horizontal ray line
		}
		t := (p.Y - a.Y) / (b.Y - a.Y)
		x := a.X + t*(b.X-a.X)
		if x < p.X || x-p.X >= bestD {
			continue
		}
		bestD, hit = x-p.X, math.P2(x, p.Y)
		if a.X > b.X {
			mIdx = i
		} else {
			mIdx = (i + 1) % n
		}
	}
	return mIdx, hit
}

// isReflex reports whether vertex i of a CCW loop is reflex (interior angle > 180°).
func isReflex(loop []math.Point2, i int) bool {
	n := len(loop)
	return predicate.Orient2D(loop[(i+n-1)%n], loop[i], loop[(i+1)%n]) < 0
}

// inTriAny reports whether p lies in triangle abc regardless of the triangle's winding.
func inTriAny(p, a, b, c math.Point2) bool {
	d1 := predicate.Orient2D(a, b, p)
	d2 := predicate.Orient2D(b, c, p)
	d3 := predicate.Orient2D(c, a, p)
	return !((d1 < 0 || d2 < 0 || d3 < 0) && (d1 > 0 || d2 > 0 || d3 > 0))
}
