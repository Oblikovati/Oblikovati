// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/math"
	"oblikovati.org/model/pointcloud"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// Point-cloud display (M17-F06, #645): the head appends these draw items to the viewport list
// after the body geometry, so attached scans render alongside the model. The per-cloud assembly
// lives here in the headless session (testable on the DrawItem snapshot), not in the cgo head.

// PointCloudItems returns the renderer draw items for the active part's visible attached clouds —
// each cloud's budgeted, model-space points as a batch of 3-axis crosses of markerSize (the head
// passes a screen-derived world size so markers stay a fixed pixel size at any zoom). Empty when
// the active document is not a part or has no visible clouds.
func (s *Session) PointCloudItems(cam scene.Camera, markerSize float64) []renderer.DrawItem {
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	clouds := part.PointClouds()
	var items []renderer.DrawItem
	for i := 0; i < clouds.Count(); i++ {
		if item := cloudDrawItem(clouds.Item(i), cam, markerSize); item != nil {
			items = append(items, *item)
		}
	}
	return items
}

// PickablePointClouds returns the active part's visible attached clouds — the snap targets the ray
// picker tests for cloud-point snapping (M17-F06, #645). Empty when the active document is not a
// part. The picker itself skips hidden clouds, but filtering here keeps the provider lean.
func (s *Session) PickablePointClouds() []*pointcloud.PointCloud {
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	clouds := part.PointClouds()
	out := make([]*pointcloud.PointCloud, 0, clouds.Count())
	for i := 0; i < clouds.Count(); i++ {
		if pc := clouds.Item(i); pc.Visible() {
			out = append(out, pc)
		}
	}
	return out
}

// CloudPointHighlightColor is the marker color for a selected (snapped) scan point.
var cloudPointHighlightColor = [4]float32{1.0, 0.62, 0.1, 1}

// SelectedCloudPointHighlight returns an on-top highlight marker for the currently selected scan
// point (a larger orange cross at its location), or ok=false when the selection is not a cloud
// point — the visible feedback that a click snapped to a scan point (M17-F06, #645).
func (s *Session) SelectedCloudPointHighlight(markerSize float64) (renderer.DrawItem, bool) {
	h, ok := s.Selection().First().(PointCloudPointHandle)
	if !ok {
		return renderer.DrawItem{}, false
	}
	item := renderer.PointMarkers([]math.Point3{h.Point}, markerSize*2.5, cloudPointHighlightColor, 0)
	if item == nil {
		return renderer.DrawItem{}, false
	}
	item.OnTop = true // draw over the model so the snap reads clearly
	return *item, true
}

// cloudDrawItem builds one visible cloud's marker batch; nil for a hidden or empty cloud. The
// displayed points (cached, model space) are thinned by screen coverage (LOD) and clipped to the
// camera frustum before the markers are built, so a large or partly-off-screen scan only spends
// marker geometry on what is actually on screen.
func cloudDrawItem(pc *pointcloud.PointCloud, cam scene.Camera, markerSize float64) *renderer.DrawItem {
	if !pc.Visible() {
		return nil
	}
	return renderer.PointMarkers(visibleDisplayPoints(pc, cam), markerSize, renderer.PointCloudColor, 0)
}

// pointCloudLODDensity is the most markers drawn per square pixel the cloud covers on screen — the
// LOD target. Below it a cloud that is small on screen is thinned so distant scans cost little, and
// a cloud filling the viewport still draws up to its display budget.
const pointCloudLODDensity = 0.5

// pointCloudNear/Far bound the projection used for clipping; the screen x,y depend only on the
// FOV/aspect, not the depth range (renderer.Project).
const (
	pointCloudNear = 0.1
	pointCloudFar  = 5000.0
)

// screenBox is a viewport-pixel rectangle.
type screenBox struct{ minX, minY, maxX, maxY float64 }

// visibleDisplayPoints returns the points to actually draw for a cloud: its cached displayed set,
// thinned to the screen-coverage LOD, then clipped to the frustum when the cloud is not fully on
// screen (when it is, every displayed point is visible, so the per-point clip is skipped).
func visibleDisplayPoints(pc *pointcloud.PointCloud, cam scene.Camera) []math.Point3 {
	pts := pc.DisplayedPoints()
	screen, fullyOnScreen := projectBoxToScreen(pc.RangeBox(), cam)
	pts = lodThin(pts, screen)
	if fullyOnScreen {
		return pts
	}
	return frustumClip(pts, cam)
}

// lodThin strides pts down so it draws no more than pointCloudLODDensity markers per pixel of the
// cloud's on-screen area — fewer points when the cloud is small on screen, the full set when large.
func lodThin(pts []math.Point3, screen screenBox) []math.Point3 {
	area := (screen.maxX - screen.minX) * (screen.maxY - screen.minY)
	budget := int(area * pointCloudLODDensity)
	if budget <= 0 || budget >= len(pts) {
		return pts
	}
	stride := len(pts) / budget
	out := make([]math.Point3, 0, budget)
	for i := 0; i < len(pts); i += stride {
		out = append(out, pts[i])
	}
	return out
}

// frustumClip keeps only the points whose projection lands within the viewport (plus a small
// margin), dropping the off-screen ones so they cost no marker geometry.
func frustumClip(pts []math.Point3, cam scene.Camera) []math.Point3 {
	const margin = 8.0
	w, h := float64(cam.Width)+margin, float64(cam.Height)+margin
	pr := renderer.NewProjector(cam, pointCloudNear, pointCloudFar) // build the matrix once, not per point
	out := pts[:0:0]                                                // a fresh backing array; never alias the cached slice
	for _, p := range pts {
		if x, y, ok := pr.Project(p); ok && x >= -margin && x <= w && y >= -margin && y <= h {
			out = append(out, p)
		}
	}
	return out
}

// projectBoxToScreen projects a model-space box's eight corners to their screen bounding rectangle
// and reports whether every corner is on screen (so the box, and thus all its points, are visible).
func projectBoxToScreen(box math.Box, cam scene.Camera) (screenBox, bool) {
	if box.IsEmpty() {
		return screenBox{}, false
	}
	r := screenBox{minX: 1e18, minY: 1e18, maxX: -1e18, maxY: -1e18}
	fully := true
	pr := renderer.NewProjector(cam, pointCloudNear, pointCloudFar)
	for _, c := range box.Corners() {
		x, y, ok := pr.Project(c)
		if !ok || x < 0 || x > float64(cam.Width) || y < 0 || y > float64(cam.Height) {
			fully = false
		}
		if ok {
			r.minX, r.minY = min(r.minX, x), min(r.minY, y)
			r.maxX, r.maxY = max(r.maxX, x), max(r.maxY, y)
		}
	}
	return r, fully
}
