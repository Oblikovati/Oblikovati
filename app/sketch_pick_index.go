// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// sketchSeg is one faceted sub-segment of a sketch entity, precomputed for picking:
// its model-space endpoints (so a pick needs no per-entity plane transform) and the
// entity it selects. A curved entity (arc/circle/ellipse/elliptical arc/spline)
// contributes several segments that all point back to the same entity, so the whole
// curve is pickable along its true sweep, not just its chord.
type sketchSeg struct {
	a, b   math.Point3
	entity sketch.Entity
}

// sketchPickIndex is a uniform 2D grid (in sketch space) over a sketch's faceted
// entity segments so hover/ray picking tests only the handful near the cursor instead
// of all of them — the difference between sluggish and instant on a dense imported
// drawing (hundreds of thousands of segments). It is camera-independent
// (model-space endpoints + sketch-space bins), so it is cached per sketch and
// rebuilt only when the entity count or the sketch plane changes.
type sketchPickIndex struct {
	count int
	plane sketch.Plane
	segs  []sketchSeg

	originX, originY float64
	cell             float64
	cols, rows       int
	bins             [][]int32 // cols*rows buckets of indices into segs
}

// sketchPickIndexes caches one index per sketch. The render/pick loop is
// single-threaded, so a package-level map is safe; entries are small relative to
// the geometry they index and are rebuilt on change.
var sketchPickIndexes = map[*sketch.Sketch]*sketchPickIndex{}

// pickIndexFor returns the cached pick index for sk, building it when absent or
// stale (entity count or plane changed).
func pickIndexFor(sk *sketch.Sketch) *sketchPickIndex {
	n := sk.EntityCount()
	if idx := sketchPickIndexes[sk]; idx != nil && idx.count == n && idx.plane == sk.Plane() {
		return idx
	}
	idx := buildPickIndex(sk, n)
	sketchPickIndexes[sk] = idx
	return idx
}

// buildPickIndex facets every entity into model-space segments and bins them into a
// roughly sqrt(N) × sqrt(N) grid over the sketch's 2D extent (N = segment count).
func buildPickIndex(sk *sketch.Sketch, count int) *sketchPickIndex {
	plane := sk.Plane()
	idx := &sketchPickIndex{count: count, plane: plane, segs: make([]sketchSeg, 0, count)}
	minX, minY := stdmath.Inf(1), stdmath.Inf(1)
	maxX, maxY := stdmath.Inf(-1), stdmath.Inf(-1)
	type box struct{ aX, aY, bX, bY float64 }
	var boxes []box
	addSeg := func(a2, b2 math.Point2, e sketch.Entity) {
		idx.segs = append(idx.segs, sketchSeg{a: plane.ToModel(a2), b: plane.ToModel(b2), entity: e})
		boxes = append(boxes, box{a2.X, a2.Y, b2.X, b2.Y})
		minX, minY = stdmath.Min(minX, stdmath.Min(a2.X, b2.X)), stdmath.Min(minY, stdmath.Min(a2.Y, b2.Y))
		maxX, maxY = stdmath.Max(maxX, stdmath.Max(a2.X, b2.X)), stdmath.Max(maxY, stdmath.Max(a2.Y, b2.Y))
	}
	for _, e := range sk.Entities() {
		pts, closed := sketch.EntityPolyline(e)
		for k := 0; k+1 < len(pts); k++ {
			addSeg(pts[k], pts[k+1], e)
		}
		if closed && len(pts) >= 2 {
			addSeg(pts[len(pts)-1], pts[0], e)
		}
	}
	n := len(idx.segs)
	if n == 0 || maxX <= minX && maxY <= minY {
		return idx // degenerate (no segments, or all coincident): queries full-scan segs
	}
	dim := int(stdmath.Sqrt(float64(n)))
	if dim < 1 {
		dim = 1
	} else if dim > 512 {
		dim = 512
	}
	idx.originX, idx.originY = minX, minY
	idx.cols, idx.rows = dim, dim
	idx.cell = stdmath.Max(stdmath.Max(maxX-minX, maxY-minY)/float64(dim), 1e-9)
	idx.bins = make([][]int32, dim*dim)
	for i, bx := range boxes {
		for cj := idx.row(stdmath.Min(bx.aY, bx.bY)); cj <= idx.row(stdmath.Max(bx.aY, bx.bY)); cj++ {
			for ci := idx.col(stdmath.Min(bx.aX, bx.bX)); ci <= idx.col(stdmath.Max(bx.aX, bx.bX)); ci++ {
				b := cj*idx.cols + ci
				idx.bins[b] = append(idx.bins[b], int32(i))
			}
		}
	}
	return idx
}

func (idx *sketchPickIndex) col(x float64) int {
	return clampCell(int((x-idx.originX)/idx.cell), idx.cols)
}
func (idx *sketchPickIndex) row(y float64) int {
	return clampCell(int((y-idx.originY)/idx.cell), idx.rows)
}

func clampCell(c, n int) int {
	if c < 0 {
		return 0
	}
	if c >= n {
		return n - 1
	}
	return c
}

// forCandidates calls visit for each segment that could be within tol of the ray:
// the segments binned near where the ray pierces the sketch plane. A ray nearly
// parallel to the plane (or a degenerate index) falls back to every segment, so a
// pick is never missed — the index only culls, the exact test still runs in the
// caller.
func (idx *sketchPickIndex) forCandidates(origin math.Point3, dir math.Vector3, tol float64, visit func(sketchSeg)) {
	if idx.cols == 0 {
		for _, s := range idx.segs {
			visit(s)
		}
		return
	}
	n := idx.plane.Normal()
	nv := math.V3(n.X(), n.Y(), n.Z())
	denom := dir.Dot(nv)
	if stdmath.Abs(denom) < 1e-9 {
		for _, s := range idx.segs {
			visit(s)
		}
		return
	}
	t := origin.VectorTo(idx.plane.Origin()).Dot(nv) / denom
	hit := math.P3(origin.X+dir.X*t, origin.Y+dir.Y*t, origin.Z+dir.Z*t)
	h := idx.plane.ToSketch(hit)
	for cj := idx.row(h.Y - tol); cj <= idx.row(h.Y+tol); cj++ {
		for ci := idx.col(h.X - tol); ci <= idx.col(h.X+tol); ci++ {
			for _, i := range idx.bins[cj*idx.cols+ci] {
				visit(idx.segs[i])
			}
		}
	}
}
