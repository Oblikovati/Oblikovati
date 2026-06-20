// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/renderer"
)

// RayPicker answers both the point pick (Picker) and the box-select region pick.
var _ RegionPicker = (*RayPicker)(nil)

// region-pick near/far for the projection — any valid pair works, since Project's screen
// x,y depend only on the FOV/aspect, not on the depth range (renderer.Project).
const (
	regionNear = 0.1
	regionFar  = 5000.0
)

// PickRegion resolves a box-select rectangle to whole visible bodies — the first-cut
// RegionPicker (whole-body selection, the common case). It projects each body's range box
// to screen and tests the projected rectangle against the selection box: a window select
// (crossing=false) keeps bodies whose projection is fully enclosed; a crossing select keeps
// bodies whose projection merely overlaps. Per-face/edge region picking and sketch-entity
// box-select are tracked follow-ups (#909).
//
// RayPicker satisfies both Picker (point pick) and RegionPicker (this), so the head installs
// the one object for both. The granularity follows the active selection filter, like Inventor's
// selection priority: a body-permissive filter (the default) selects whole bodies, a face-only
// filter selects faces, an edge-only filter selects edges.
func (p *RayPicker) PickRegion(x0, y0, x1, y1 float64, crossing bool, filter *SelectionFilter) []Selectable {
	if p.bodies == nil {
		return nil
	}
	sel := orderedRect(x0, y0, x1, y1)
	switch {
	case filter.Accepts(SelectBody):
		return p.regionBodies(sel, crossing)
	case filter.Accepts(SelectFace):
		return p.regionFaces(sel, crossing)
	case filter.Accepts(SelectEdge):
		return p.regionEdges(sel, crossing)
	default:
		return nil
	}
}

// regionBodies selects whole bodies whose projected range box satisfies the rectangle.
func (p *RayPicker) regionBodies(sel screenRect, crossing bool) []Selectable {
	var hits []Selectable
	for _, b := range p.bodies() {
		if box, ok := p.projectBodyRect(b); ok && rectCovers(sel, box, crossing) {
			hits = append(hits, BodyHandle{Body: b})
		}
	}
	return hits
}

// regionFaces selects faces whose projected vertex bounds satisfy the rectangle.
func (p *RayPicker) regionFaces(sel screenRect, crossing bool) []Selectable {
	var hits []Selectable
	for _, b := range p.bodies() {
		for _, f := range b.Faces() {
			if box, ok := p.projectVertexRect(f.Vertices()); ok && rectCovers(sel, box, crossing) {
				hits = append(hits, FaceHandle{Face: f, Body: b})
			}
		}
	}
	return hits
}

// regionEdges selects edges whose projected outline satisfies the rectangle. The outline is the
// edge's full adaptive sampling (ops.TessellateEdge — the same discretization the renderer draws),
// not just the two endpoints, so a curved edge whose mid-span enters the box but whose endpoints
// sit outside is caught by a crossing select and correctly excluded by a window select (#936).
// A straight edge samples to its two endpoints, preserving the prior behaviour.
func (p *RayPicker) regionEdges(sel screenRect, crossing bool) []Selectable {
	q := ops.DefaultQuality()
	var hits []Selectable
	for _, b := range p.bodies() {
		for _, e := range b.Edges() {
			pts := p.projectModelPoints(ops.TessellateEdge(e, q))
			if len(pts) >= 2 && regionCoversOutline(pts, sel, crossing) {
				hits = append(hits, EdgeHandle{Edge: e})
			}
		}
	}
	return hits
}

// projectModelPoints maps world points to viewport pixels, dropping any at/behind the camera —
// the projected outline a region edge test consumes.
func (p *RayPicker) projectModelPoints(model []math.Point3) []screenPt {
	out := make([]screenPt, 0, len(model))
	for _, m := range model {
		if sx, sy, ok := renderer.Project(p.camera, regionNear, regionFar, m); ok {
			out = append(out, screenPt{sx, sy})
		}
	}
	return out
}

// rectCovers reports whether sel covers box per the select mode: crossing = overlap, window =
// full enclosure.
func rectCovers(sel, box screenRect, crossing bool) bool {
	if crossing {
		return sel.overlaps(box)
	}
	return sel.contains(box)
}

// screenRect is a viewport-pixel axis-aligned rectangle (minX≤maxX, minY≤maxY).
type screenRect struct{ minX, minY, maxX, maxY float64 }

// orderedRect builds a normalized rectangle from two opposite corners.
func orderedRect(x0, y0, x1, y1 float64) screenRect {
	return screenRect{minF(x0, x1), minF(y0, y1), maxF(x0, x1), maxF(y0, y1)}
}

// contains reports whether r fully encloses inner (window-select test).
func (r screenRect) contains(inner screenRect) bool {
	return inner.minX >= r.minX && inner.maxX <= r.maxX &&
		inner.minY >= r.minY && inner.maxY <= r.maxY
}

// overlaps reports whether r and other intersect at all (crossing-select test).
func (r screenRect) overlaps(other screenRect) bool {
	return r.minX <= other.maxX && r.maxX >= other.minX &&
		r.minY <= other.maxY && r.maxY >= other.minY
}

// containsPoint reports whether (x,y) lies inside r (inclusive).
func (r screenRect) containsPoint(x, y float64) bool {
	return x >= r.minX && x <= r.maxX && y >= r.minY && y <= r.maxY
}

// projectBodyRect projects the eight corners of a body's range box to a screen rectangle.
func (p *RayPicker) projectBodyRect(b *topo.Body) (screenRect, bool) {
	corners := b.RangeBox().Corners()
	return p.projectPointsRect(corners[:])
}

// projectVertexRect projects topology vertices to their screen bounding rectangle.
func (p *RayPicker) projectVertexRect(vs []*topo.Vertex) (screenRect, bool) {
	pts := make([]math.Point3, len(vs))
	for i, v := range vs {
		pts[i] = v.Point()
	}
	return p.projectPointsRect(pts)
}

// projectPointsRect projects world points to their screen bounding rectangle. ok is false when
// any point is at/behind the camera (partly off-screen), in which case the caller skips it.
func (p *RayPicker) projectPointsRect(pts []math.Point3) (screenRect, bool) {
	first := true
	var r screenRect
	for _, c := range pts {
		sx, sy, ok := renderer.Project(p.camera, regionNear, regionFar, c)
		if !ok {
			return screenRect{}, false
		}
		if first {
			r = screenRect{sx, sy, sx, sy}
			first = false
			continue
		}
		r.minX, r.minY = minF(r.minX, sx), minF(r.minY, sy)
		r.maxX, r.maxY = maxF(r.maxX, sx), maxF(r.maxY, sy)
	}
	return r, !first
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
