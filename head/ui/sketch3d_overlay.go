//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"strings"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/linetype"
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
	curves               []renderer.DrawItem // all visible curve segments, styled per entity
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

// sketch3DCacheBounds is the model-space extent of the just-rebuilt cache. It is computed
// ROBUSTLY (math.RobustPointBox): far-flung off-sheet strays — a georeferenced DWG's stray
// entities sit hundreds of thousands of units from the drawing — are excluded, so the viewport
// far plane is sized to the drawing the camera frames rather than to a stray. A huge far plane
// degenerated the inverse view-projection the skybox reconstructs view rays from, blanking the
// HDR sky to black (the strays are still drawn; they are simply clipped once far enough behind
// the framed drawing). Matches the framing in app.unionSketchBounds so the two stay consistent.
func sketch3DCacheBounds() (min, max [3]float32, ok bool) {
	total := len(sketch3DOverlayCache.points)
	for _, it := range sketch3DOverlayCache.curves {
		total += len(it.Positions)
	}
	pts := make([]math.Point3, 0, total)
	for _, it := range sketch3DOverlayCache.curves {
		pts = append(pts, it.Positions...)
	}
	pts = append(pts, sketch3DOverlayCache.points...)
	b := math.RobustPointBox(pts)
	if b.IsEmpty() {
		return min, max, false
	}
	return [3]float32{float32(b.Min.X), float32(b.Min.Y), float32(b.Min.Z)},
		[3]float32{float32(b.Max.X), float32(b.Max.Y), float32(b.Max.Z)}, true
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
// visibility / entity count / format revision), which changes on import and on add/remove/edit —
// but NOT on a camera move or a pick, so the cached curve bulk survives both. The Show Format
// toggle and the format revision are in the key because recolouring an entity changes neither
// the geometry version nor the entity count, so the cache would otherwise show stale styling
// (the planar overlay keys on the same two, #2015/#2039).
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
	fmt.Fprintf(&b, "|fmt=%t", s.ShowFormat())
	for i := 0; i < part.Sketches3D().Count(); i++ {
		sk := part.Sketches3D().Item(i)
		fmt.Fprintf(&b, "|%d:%t:%d:%d", sk.Seq(), sk.Visible(), sk.EntityCount(), sk.FormatRevision())
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
	items = appendGrid(items, normal, chromeTheme.sketchColor)
	items = appendGrid(items, sel, chromeTheme.sketchSelectedColor)
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

// highlightPickedSketch3D adds the entities to draw in the selected colour to sel — curves
// re-sampled (the set is small) and standalone points as crosses — so they highlight over the
// cached normal curves. That is the active pick-driven tool's gathered entities, plus the
// ambient selection: with no tool running, a picked 3D curve had no highlight at all, so
// selecting geometry for the Format panel gave no sign anything was selected (#2039).
func highlightPickedSketch3D(s *app.Session, sel *segAccum, h float64) {
	for e := range highlightedSketch3DEntities(s) {
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

// buildSketch3DCurves samples every visible 3D sketch once, returning the styled curve overlay
// items and the standalone-point positions. This is the cached (geometry-keyed) build.
func buildSketch3DCurves(s *app.Session) (curves []renderer.DrawItem, points []math.Point3) {
	part := activePart(s)
	if part == nil {
		return nil, nil
	}
	lane := newStrokeLane()
	for i := 0; i < part.Sketches3D().Count(); i++ {
		sk := part.Sketches3D().Item(i)
		if !sk.Visible() {
			continue
		}
		points = accumSketch3DCurves(sk, &lane, s.ShowFormat(), points)
	}
	return lane.appendTo(nil), points
}

// buildSketch3DCurvesOnly is buildSketch3DCurves discarding the points — the un-keyed bounds
// fallback only needs the curve extent.
func buildSketch3DCurvesOnly(s *app.Session) []renderer.DrawItem {
	curves, _ := buildSketch3DCurves(s)
	return curves
}

// accumSketch3DCurves accumulates one 3D sketch's curve segments into the stroke lane and
// appends its standalone-point positions to points (single-sample entities are lone points, not
// curves). Each curve is bucketed by its resolved colour and stroke width and dashed by its line
// type, so a construction curve reads as construction and a Format-panel override reaches the
// screen — until #2039 every 3D curve drew identically in the plain sketch colour.
//
// suppress is the Show Format toggle: it draws every entity with default attributes.
func accumSketch3DCurves(sk *sketch.Sketch3D, lane *strokeLane, suppress bool, points []math.Point3) []math.Point3 {
	for _, e := range sk.Entities() {
		pts := sketch.SamplePolyline3D(e, sketchSegments)
		if len(pts) == 1 {
			points = append(points, pts[0])
			continue
		}
		style := app.SketchEntityStyle(sk, e, suppress)
		acc := lane.forStroke(strokeKey{sketchEntityColor(style), sketchStrokeWidth(style)})
		accumPatterned3D(acc, pts, style.Pattern)
	}
	return points
}

// accumPatterned3D adds a sampled model-space curve either solid or split into its line-type
// dashes; a nil or degenerate pattern falls back to the solid polyline.
func accumPatterned3D(acc *segAccum, pts []math.Point3, pattern []float64) {
	if segs := linetype.DashPolyline3D(pts, false, pattern); segs != nil {
		for _, s := range segs {
			acc.addSegment(s[0], s[1])
		}
		return
	}
	for i := 0; i+1 < len(pts); i++ {
		acc.addSegment(pts[i], pts[i+1])
	}
}

// highlightedSketch3DEntities is every sketch entity that should draw selected: the active
// pick-driven tool's gathered set, or — with no such tool — the ambient selection.
func highlightedSketch3DEntities(s *app.Session) map[sketch.Entity]bool {
	if picked := pickedSketchEntities(s); len(picked) > 0 {
		return picked
	}
	set := map[sketch.Entity]bool{}
	for _, it := range s.Selection().Items() {
		if h, ok := it.(app.SketchEntityHandle); ok {
			set[h.Entity] = true
		}
	}
	return set
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
