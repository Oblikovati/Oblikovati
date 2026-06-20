// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/pointcloud"
)

// Point-cloud browser presence (M17-F06, #645): the active part's attached scans appear under a
// Point Clouds folder, each node selectable so the right-click menu can toggle its visibility or
// delete it (see pointCloudMenu).

// PointCloudHandle is the selectable browser handle for one attached cloud: the owning collection
// (for delete) and the cloud itself.
type PointCloudHandle struct {
	Clouds *pointcloud.PointClouds
	Cloud  *pointcloud.PointCloud
}

// SelectionKind implements Selectable.
func (PointCloudHandle) SelectionKind() SelectionKind { return SelectPointCloud }

// addPointCloudBranch lists the part's attached scans under a Point Clouds folder, a hidden cloud
// tagged so its state reads in the tree. Omitted when the part has no clouds.
func addPointCloudBranch(root *BrowserNode, part *compdef.PartComponentDefinition) {
	clouds := part.PointClouds()
	if clouds.Count() == 0 {
		return
	}
	folder := root.child("Point Clouds", "pointClouds")
	for i := 0; i < clouds.Count(); i++ {
		pc := clouds.Item(i)
		label := pc.Name()
		if !pc.Visible() {
			label += "  (hidden)"
		}
		folder.selectableChild(label, "pointCloud", PointCloudHandle{Clouds: clouds, Cloud: pc})
	}
}
