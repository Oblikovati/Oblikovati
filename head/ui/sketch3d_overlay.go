//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"strings"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// sketch3DBoundsCache memoises the model-space extent of the finished 3D-sketch line work, so
// the viewport's adaptive far plane can enclose a non-planar DWG import (large coordinates,
// all line primitives) without rescanning every segment each frame. Keyed on the 3D-sketch
// geometry so it survives camera moves and rebuilds only on import/edit.
var sketch3DBoundsCache struct {
	key                  string
	boundsMin, boundsMax [3]float32
	hasBounds            bool
}

// cachedSketch3DBounds returns the extent of the active part's visible 3D sketches, computed at
// the last geometry change. ok is false when there is no 3D-sketch line work. The far plane
// unions this with the body/2D-sketch bounds so a 3D import is not clipped on zoom-out.
func cachedSketch3DBounds(s *app.Session) (min, max [3]float32, ok bool) {
	key, keyed := sketch3DBoundsKey(s)
	if !keyed {
		return drawItemsBounds(sketch3DOverlays(s, 1))
	}
	if key != sketch3DBoundsCache.key {
		sketch3DBoundsCache.key = key
		sketch3DBoundsCache.boundsMin, sketch3DBoundsCache.boundsMax, sketch3DBoundsCache.hasBounds =
			drawItemsBounds(sketch3DOverlays(s, 1))
	}
	return sketch3DBoundsCache.boundsMin, sketch3DBoundsCache.boundsMax, sketch3DBoundsCache.hasBounds
}

// sketch3DBoundsKey fingerprints the 3D-sketch geometry (model version + each sketch's seq /
// visibility / entity count), which changes on import and on add/remove/edit.
func sketch3DBoundsKey(s *app.Session) (string, bool) {
	version, ok := activeModelGeometryVersion(s)
	if !ok {
		return "", false
	}
	part := activePart(s)
	if part == nil {
		return "", false
	}
	var b strings.Builder
	b.WriteString(version)
	for i := 0; i < part.Sketches3D().Count(); i++ {
		sk := part.Sketches3D().Item(i)
		fmt.Fprintf(&b, "|%d:%t:%d", sk.Seq(), sk.Visible(), sk.EntityCount())
	}
	return b.String(), true
}

// sketch3DOverlays renders the active part's 3D sketches in the viewport — curves as
// sampled polylines (the same sampling the ray picker tests against), standalone
// points as screen-constant crosses — so 3D-sketch geometry is visible and pickable
// for the constraint tools (issue #142). Entities picked by the active pick-driven
// tool draw highlighted, mirroring the 2D sketch overlay's cue.
func sketch3DOverlays(s *app.Session, hPoint float64) []renderer.DrawItem {
	part := activePart(s)
	if part == nil {
		return nil
	}
	picked := pickedSketchEntities(s)
	normal, sel := &segAccum{}, &segAccum{}
	for i := 0; i < part.Sketches3D().Count(); i++ {
		sk := part.Sketches3D().Item(i)
		if !sk.Visible() {
			continue
		}
		accumSketch3D(sk, picked, normal, sel, hPoint)
	}
	var items []renderer.DrawItem
	items = appendGrid(items, normal, sketchColor)
	items = appendGrid(items, sel, sketchSelectedColor)
	return items
}

// pickedSketchEntities returns the entities the active pick-driven tool has gathered
// (so they highlight), or nil when no such tool is active.
func pickedSketchEntities(s *app.Session) map[sketch.Entity]bool {
	at := s.ActiveTool()
	if at == nil {
		return nil
	}
	et, ok := at.Tool().(app.SketchEntityTool)
	if !ok {
		return nil
	}
	picked := et.Picked()
	if len(picked) == 0 {
		return nil
	}
	set := make(map[sketch.Entity]bool, len(picked))
	for _, e := range picked {
		set[e] = true
	}
	return set
}

// accumSketch3D accumulates one 3D sketch's entities as model-space segments: curves
// by their sampled polyline, single points as a three-axis cross of half-size h.
func accumSketch3D(sk *sketch.Sketch3D, picked map[sketch.Entity]bool, normal, sel *segAccum, h float64) {
	for _, e := range sk.Entities() {
		acc := normal
		if picked[e] {
			acc = sel
		}
		pts := sketch.SamplePolyline3D(e, sketchSegments)
		if len(pts) == 1 {
			accumPointCross3D(acc, pts[0], h)
			continue
		}
		for i := 0; i+1 < len(pts); i++ {
			acc.addSegment(pts[i], pts[i+1])
		}
	}
}

// accumPointCross3D draws a standalone 3D point as a small axis-aligned cross.
func accumPointCross3D(acc *segAccum, p math.Point3, h float64) {
	d := math.Scalar(h)
	acc.addSegment(math.P3(p.X-d, p.Y, p.Z), math.P3(p.X+d, p.Y, p.Z))
	acc.addSegment(math.P3(p.X, p.Y-d, p.Z), math.P3(p.X, p.Y+d, p.Z))
	acc.addSegment(math.P3(p.X, p.Y, p.Z-d), math.P3(p.X, p.Y, p.Z+d))
}
