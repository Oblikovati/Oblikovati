// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/fit"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/pointcloud"
	"oblikovati.org/model/sketch"
)

// Datum-cloud provenance (M17-F06, #645): a work plane fit to a point cloud keeps a live link to
// that cloud (a PlaneFitSource) so it re-fits when the cloud moves, and persists the cloud's id so
// the link is re-attached after a document is reopened (the live source object is not serialized,
// only its id).

// cloudPlaneFitSource adapts a point cloud to feature.PlaneFitSource: it best-fits the cloud's
// currently displayed points (those passing its active crops) on demand, identified by cloud name.
type cloudPlaneFitSource struct{ cloud *pointcloud.PointCloud }

func (s cloudPlaneFitSource) SourceID() string { return s.cloud.Name() }

// FitFrame best-fits a plane to the cloud's cropped points, returning its origin and in-plane axes;
// ok is false when the points do not determine a plane (too few, or collinear).
func (s cloudPlaneFitSource) FitFrame() (math.Point3, math.UnitVector3, math.UnitVector3, bool) {
	plane, err := fit.Plane(s.cloud.CroppedModelPoints())
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, math.UnitVector3{}, false
	}
	return plane.Origin, plane.UAxis, plane.VAxis, true
}

// cloudPointSource adapts a point cloud to feature.PointFromCloudSource: it anchors a scan point in
// cloud space (local) and re-derives its model-space position as the cloud moves, identified by
// cloud name.
type cloudPointSource struct {
	cloud *pointcloud.PointCloud
	local math.Point3 // cloud-space anchor; the model position re-derives from the cloud's placement
}

func (s cloudPointSource) SourceID() string { return s.cloud.Name() }

// Position re-derives the anchor's current model-space location through the cloud's placement.
func (s cloudPointSource) Position() (math.Point3, bool) { return s.cloud.ToModelSpace(s.local), true }

// newCloudPointSource anchors a model-space scan point in cloud space so the work point follows the
// cloud's placement. A non-invertible placement (degenerate, scale 0) leaves the anchor at the raw
// point — it stays valid, it just will not track.
func newCloudPointSource(cloud *pointcloud.PointCloud, modelPoint math.Point3) cloudPointSource {
	local, ok := cloud.FromModelSpace(modelPoint)
	if !ok {
		local = modelPoint
	}
	return cloudPointSource{cloud: cloud, local: local}
}

// cloudSketchPointSource adapts a point cloud to sketch.CloudPointAnchor: it anchors a scan point in
// cloud space and re-derives its model-space position as the cloud moves, for the sketch to project.
type cloudSketchPointSource struct {
	cloud *pointcloud.PointCloud
	local math.Point3
}

func (s cloudSketchPointSource) SourceID() string         { return s.cloud.Name() }
func (s cloudSketchPointSource) LocalAnchor() math.Point3 { return s.local }
func (s cloudSketchPointSource) ModelPosition() (math.Point3, bool) {
	return s.cloud.ToModelSpace(s.local), true
}

// newCloudSketchPointSource anchors a model-space scan point in cloud space for a sketch point.
func newCloudSketchPointSource(cloud *pointcloud.PointCloud, modelPoint math.Point3) cloudSketchPointSource {
	local, ok := cloud.FromModelSpace(modelPoint)
	if !ok {
		local = modelPoint
	}
	return cloudSketchPointSource{cloud: cloud, local: local}
}

// relinkPointCloudProvenance re-attaches live cloud sources to a freshly opened part's
// point-cloud-derived datums — the fit planes, the anchored work points, and the scan-anchored
// sketch points — matching by cloud id, then recomputes so they re-derive against the restored
// clouds. It is a no-op for a non-part document or a part with no such datums.
func (s *Session) relinkPointCloudProvenance(part *compdef.PartComponentDefinition) {
	relinked := part.WorkPlanes().RelinkCloudFits(func(id string) (feature.PlaneFitSource, bool) {
		if pc, ok := part.PointClouds().ByName(id); ok {
			return cloudPlaneFitSource{pc}, true
		}
		return nil, false
	})
	relinked += part.WorkPoints().RelinkCloudPoints(func(id string, frozen math.Point3) (feature.PointFromCloudSource, bool) {
		if pc, ok := part.PointClouds().ByName(id); ok {
			return newCloudPointSource(pc, frozen), true // reconstruct the cloud-local anchor from the frozen point
		}
		return nil, false
	})
	relinked += s.relinkSketchCloudAnchors(part)
	if relinked > 0 {
		part.Recompute()
	}
}

// relinkSketchCloudAnchors re-attaches cloud sources to every sketch's scan-anchored points.
func (s *Session) relinkSketchCloudAnchors(part *compdef.PartComponentDefinition) int {
	n := 0
	sketches := part.Sketches()
	for i := 0; i < sketches.Count(); i++ {
		n += sketches.Item(i).RelinkCloudAnchors(func(id string, local math.Point3) (sketch.CloudPointAnchor, bool) {
			if pc, ok := part.PointClouds().ByName(id); ok {
				return cloudSketchPointSource{cloud: pc, local: local}, true
			}
			return nil, false
		})
	}
	return n
}
