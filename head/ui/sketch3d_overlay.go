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

// sketch3DOverlayCache memoises the finished 3D-sketch CURVE wireframe (the camera-independent
// bulk) together with the standalone-point positions and the model-space extent. A dense
// non-planar DWG import becomes a Sketch3D of hundreds of thousands of segments; re-sampling
// every curve every frame (SamplePolyline3D is the costly step) dropped the viewport to a
// crawl and meant a captured frame could catch a mid-build overlay. The curve geometry is
// camera-independent (model space), so it holds across camera moves and picking and rebuilds
// only when the geometry changes (see sketch3DOverlayKey). Standalone-point crosses are screen-
// constant (size depends on zoom) and pick highlights change per interaction, so those small
// pieces are built fresh each frame on top of the cached curves — without re-sampling the bulk.
// The render loop is single-threaded, so a package-level cache is safe.
var sketch3DOverlayCache struct {
	key                  string
	curves               []renderer.DrawItem // all visible curve segments, normal colour
	points               []math.Point3       // standalone-point positions (cross drawn per frame)
	boundsMin, boundsMax [3]float32
	hasBounds            bool
}

// cachedSketch3DCurves returns the cached normal-colour curve overlay, rebuilding it only when
// the 3D-sketch geometry changes. The returned slice is a shallow copy so the caller's appends
// never mutate the cached list. With no part to key on it falls back to an uncached build.
func cachedSketch3DCurves(s *app.Session) []renderer.DrawItem {
	key, ok := sketch3DOverlayKey(s)
	if !ok {
		curves, _ := buildSketch3DCurves(s)
		return curves
	}
	if key != sketch3DOverlayCache.key {
		sketch3DOverlayCache.key = key
		sketch3DOverlayCache.curves, sketch3DOverlayCache.points = buildSketch3DCurves(s)
		sketch3DOverlayCache.boundsMin, sketch3DOverlayCache.boundsMax, sketch3DOverlayCache.hasBounds =
			sketch3DCacheBounds()
	}
	return append([]renderer.DrawItem(nil), sketch3DOverlayCache.curves...)
}

// sketch3DCacheBounds is the extent of the just-rebuilt cache: the curve vertices widened by
// any standalone points (a lone point off on its own still extends the far plane / Fit box).
func sketch3DCacheBounds() (min, max [3]float32, ok bool) {
	min, max, ok = drawItemsBounds(sketch3DOverlayCache.curves)
	for _, p := range sketch3DOverlayCache.points {
		if !ok {
			min, max, ok = [3]float32{float32(p.X), float32(p.Y), float32(p.Z)}, [3]float32{float32(p.X), float32(p.Y), float32(p.Z)}, true
			continue
		}
		min[0], max[0] = minF32(min[0], float32(p.X)), maxF32(max[0], float32(p.X))
		min[1], max[1] = minF32(min[1], float32(p.Y)), maxF32(max[1], float32(p.Y))
		min[2], max[2] = minF32(min[2], float32(p.Z)), maxF32(max[2], float32(p.Z))
	}
	return min, max, ok
}

// cachedSketch3DBounds returns the extent of the active part's visible 3D sketches, computed at
// the last cache rebuild, so the viewport far plane encloses a non-planar DWG import (large
// coordinates, all line primitives) without rescanning every segment each frame. ok is false
// when there is no 3D-sketch line work. Reading the cache first builds it if the key changed.
func cachedSketch3DBounds(s *app.Session) (min, max [3]float32, ok bool) {
	if _, keyed := sketch3DOverlayKey(s); !keyed {
		return drawItemsBounds(buildSketch3DCurvesOnly(s))
	}
	cachedSketch3DCurves(s) // rebuilds the cache (including bounds) when the key changed
	return sketch3DOverlayCache.boundsMin, sketch3DOverlayCache.boundsMax, sketch3DOverlayCache.hasBounds
}

// sketch3DOverlayKey fingerprints the 3D-sketch geometry (model version + each sketch's seq /
// visibility / entity count), which changes on import and on add/remove/edit — but NOT on a
// camera move or a pick, so the cached curve bulk survives both.
func sketch3DOverlayKey(s *app.Session) (string, bool) {
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

// sketch3DOverlays renders the active part's 3D sketches in the viewport — curves as sampled
// polylines (the same sampling the ray picker tests against), standalone points as screen-
// constant crosses — so 3D-sketch geometry is visible and pickable for the constraint tools
// (issue #142). The curve bulk comes from the geometry-keyed cache; the small per-frame tail
// is the standalone-point crosses (sized by hPoint) and the highlight for entities the active
// pick-driven tool has gathered, drawn over the cached normal curves.
func sketch3DOverlays(s *app.Session, hPoint float64) []renderer.DrawItem {
	items := cachedSketch3DCurves(s)
	return append(items, sketch3DLiveOverlay(s, hPoint)...)
}

// sketch3DLiveOverlay builds the per-frame, non-cacheable part of the 3D overlay: standalone-
// point crosses at the current screen-constant size, plus the pick highlight. It draws the
// cached points (positions are camera-independent) and re-samples only the small picked set,
// so it never re-samples the curve bulk.
func sketch3DLiveOverlay(s *app.Session, h float64) []renderer.DrawItem {
	normal, sel := &segAccum{}, &segAccum{}
	for _, p := range cachePointsFor(s) {
		accumPointCross3D(normal, p, h)
	}
	highlightPickedSketch3D(s, sel, h)
	var items []renderer.DrawItem
	items = appendGrid(items, normal, sketchColor)
	items = appendGrid(items, sel, sketchSelectedColor)
	return items
}

// cachePointsFor returns the standalone-point positions for the active part, from the cache
// when it is keyed (the common path) or a fresh scan when it is not.
func cachePointsFor(s *app.Session) []math.Point3 {
	if _, ok := sketch3DOverlayKey(s); ok {
		return sketch3DOverlayCache.points
	}
	_, points := buildSketch3DCurves(s)
	return points
}

// highlightPickedSketch3D adds the active pick-driven tool's gathered entities to sel — curves
// re-sampled (the set is small) and standalone points as crosses — so they highlight over the
// cached normal curves. A no-op when no such tool is active or nothing is picked.
func highlightPickedSketch3D(s *app.Session, sel *segAccum, h float64) {
	for e := range pickedSketchEntities(s) {
		pts := sketch.SamplePolyline3D(e, sketchSegments)
		if len(pts) == 1 {
			accumPointCross3D(sel, pts[0], h)
			continue
		}
		for i := 0; i+1 < len(pts); i++ {
			sel.addSegment(pts[i], pts[i+1])
		}
	}
}

// buildSketch3DCurves samples every visible 3D sketch once, returning the normal-colour curve
// overlay items and the standalone-point positions. This is the cached (geometry-keyed) build.
func buildSketch3DCurves(s *app.Session) (curves []renderer.DrawItem, points []math.Point3) {
	part := activePart(s)
	if part == nil {
		return nil, nil
	}
	acc := &segAccum{}
	for i := 0; i < part.Sketches3D().Count(); i++ {
		sk := part.Sketches3D().Item(i)
		if !sk.Visible() {
			continue
		}
		points = accumSketch3DCurves(sk, acc, points)
	}
	return appendGrid(nil, acc, sketchColor), points
}

// buildSketch3DCurvesOnly is buildSketch3DCurves discarding the points — the un-keyed bounds
// fallback only needs the curve extent.
func buildSketch3DCurvesOnly(s *app.Session) []renderer.DrawItem {
	curves, _ := buildSketch3DCurves(s)
	return curves
}

// accumSketch3DCurves accumulates one 3D sketch's curve segments into acc and appends its
// standalone-point positions to points (single-sample entities are lone points, not curves).
func accumSketch3DCurves(sk *sketch.Sketch3D, acc *segAccum, points []math.Point3) []math.Point3 {
	for _, e := range sk.Entities() {
		pts := sketch.SamplePolyline3D(e, sketchSegments)
		if len(pts) == 1 {
			points = append(points, pts[0])
			continue
		}
		for i := 0; i+1 < len(pts); i++ {
			acc.addSegment(pts[i], pts[i+1])
		}
	}
	return points
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

// accumPointCross3D draws a standalone 3D point as a small axis-aligned cross.
func accumPointCross3D(acc *segAccum, p math.Point3, h float64) {
	d := math.Scalar(h)
	acc.addSegment(math.P3(p.X-d, p.Y, p.Z), math.P3(p.X+d, p.Y, p.Z))
	acc.addSegment(math.P3(p.X, p.Y-d, p.Z), math.P3(p.X, p.Y+d, p.Z))
	acc.addSegment(math.P3(p.X, p.Y, p.Z-d), math.P3(p.X, p.Y, p.Z+d))
}
