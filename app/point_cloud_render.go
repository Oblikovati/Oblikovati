// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/math"
	"oblikovati.org/model/pointcloud"
	"oblikovati.org/renderer"
)

// Point-cloud display (M17-F06, #645): the head appends these draw items to the viewport list
// after the body geometry, so attached scans render alongside the model. The per-cloud assembly
// lives here in the headless session (testable on the DrawItem snapshot), not in the cgo head.

// PointCloudItems returns the renderer draw items for the active part's visible attached clouds —
// each cloud's budgeted, model-space points as a batch of 3-axis crosses of markerSize (the head
// passes a screen-derived world size so markers stay a fixed pixel size at any zoom). Empty when
// the active document is not a part or has no visible clouds.
func (s *Session) PointCloudItems(markerSize float64) []renderer.DrawItem {
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	clouds := part.PointClouds()
	var items []renderer.DrawItem
	for i := 0; i < clouds.Count(); i++ {
		if item := cloudDrawItem(clouds.Item(i), markerSize); item != nil {
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

// cloudDrawItem builds one visible cloud's marker batch; nil for a hidden or empty cloud.
func cloudDrawItem(pc *pointcloud.PointCloud, markerSize float64) *renderer.DrawItem {
	if !pc.Visible() {
		return nil
	}
	return renderer.PointMarkers(pc.DisplayedPoints(), markerSize, renderer.PointCloudColor, 0)
}
