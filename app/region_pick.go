// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/topo"
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
// the one object for both.
func (p *RayPicker) PickRegion(x0, y0, x1, y1 float64, crossing bool, filter *SelectionFilter) []Selectable {
	if !filter.Accepts(SelectBody) || p.bodies == nil {
		return nil
	}
	sel := orderedRect(x0, y0, x1, y1)
	var hits []Selectable
	for _, b := range p.bodies() {
		box, ok := p.projectBodyRect(b)
		if !ok {
			continue
		}
		if (crossing && box.overlaps(sel)) || (!crossing && sel.contains(box)) {
			hits = append(hits, BodyHandle{Body: b})
		}
	}
	return hits
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

// projectBodyRect projects the eight corners of a body's range box to screen and returns
// their bounding rectangle. ok is false when any corner is at/behind the camera (the body
// is partly off-screen), in which case the caller skips it.
func (p *RayPicker) projectBodyRect(b *topo.Body) (screenRect, bool) {
	corners := b.RangeBox().Corners()
	first := true
	var r screenRect
	for _, c := range corners {
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
