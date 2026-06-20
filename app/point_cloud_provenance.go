// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/fit"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/pointcloud"
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

// relinkPointCloudFits re-attaches live cloud sources to a freshly opened part's point-cloud-fit
// work planes (matching by cloud id), then recomputes so they re-fit to the restored clouds. It is
// a no-op for a non-part document.
func (s *Session) relinkPointCloudFits(part *compdef.PartComponentDefinition) {
	relinked := part.WorkPlanes().RelinkCloudFits(func(id string) (feature.PlaneFitSource, bool) {
		if pc, ok := part.PointClouds().ByName(id); ok {
			return cloudPlaneFitSource{pc}, true
		}
		return nil, false
	})
	if relinked > 0 {
		part.Recompute()
	}
}
