// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/math"
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

// PointCloudPointHandle is a snapped scan point: the owning cloud and the picked point's
// model-space location. A point-collecting tool reads Position to model against as-built scan
// data (M17-F06, #645). It is what the ray picker returns when the cursor snaps to a cloud point.
type PointCloudPointHandle struct {
	Cloud *pointcloud.PointCloud
	Point math.Point3
}

// SelectionKind implements Selectable.
func (PointCloudPointHandle) SelectionKind() SelectionKind { return SelectPointCloudPoint }

// Position returns the snapped point's model-space location.
func (h PointCloudPointHandle) Position() math.Point3 { return h.Point }

// PointCloudCropHandle is the selectable browser handle for one crop: the owning cloud (for the
// crop collection) and the crop itself.
type PointCloudCropHandle struct {
	Cloud *pointcloud.PointCloud
	Crop  *pointcloud.PointCloudCrop
}

// SelectionKind implements Selectable.
func (PointCloudCropHandle) SelectionKind() SelectionKind { return SelectPointCloud }

// addPointCloudBranch lists the part's attached scans under a Point Clouds folder, a hidden cloud
// tagged so its state reads in the tree, each cloud nesting its crop volumes. Omitted when the
// part has no clouds.
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
		node := folder.selectableBranch(label, "pointCloud", PointCloudHandle{Clouds: clouds, Cloud: pc})
		addCropNodes(node, pc)
	}
}

// addCropNodes nests a cloud's crop volumes under its browser node, each tagged active/inactive.
func addCropNodes(cloudNode *BrowserNode, pc *pointcloud.PointCloud) {
	crops := pc.Crops()
	for i := 0; i < crops.Count(); i++ {
		c := crops.Item(i)
		label := c.Name()
		if !c.Active() {
			label += "  (inactive)"
		}
		cloudNode.selectableChild(label, "pointCloudCrop", PointCloudCropHandle{Cloud: pc, Crop: c})
	}
}
