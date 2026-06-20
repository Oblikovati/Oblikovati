// SPDX-License-Identifier: GPL-2.0-only

package app

import (
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

// cloudDrawItem builds one visible cloud's marker batch; nil for a hidden or empty cloud.
func cloudDrawItem(pc *pointcloud.PointCloud, markerSize float64) *renderer.DrawItem {
	if !pc.Visible() {
		return nil
	}
	return renderer.PointMarkers(pc.DisplayedPoints(), markerSize, renderer.PointCloudColor, 0)
}
