// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/fit"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/pointcloud"
)

// CreatePointCloudPlane fits a least-squares work plane to the named cloud's displayed points —
// those in model space passing the cloud's active crops — and adds it as a fixed datum plane, then
// recomputes. Cropping the cloud to a planar region first selects what the plane is fitted to. It
// errors when there is no active part, no such cloud, or the points do not determine a plane (fewer
// than three, or collinear). The fitted geom.Plane is returned alongside the datum for callers that
// report the origin/normal (#645).
func (s *Session) CreatePointCloudPlane(cloud string) (*feature.WorkPlane, geom.Plane, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, geom.Plane{}, err
	}
	pc, ok := part.PointClouds().ByName(cloud)
	if !ok {
		return nil, geom.Plane{}, fmt.Errorf("app: no point cloud named %q to fit a plane to", cloud)
	}
	plane, err := fit.Plane(pc.CroppedModelPoints()) // validate the fit and report origin/normal
	if err != nil {
		return nil, geom.Plane{}, fmt.Errorf("app: fit plane to point cloud %q: %w", cloud, err)
	}
	// The plane keeps a live link to the cloud (provenance): it re-fits when the cloud moves and
	// the link round-trips in the document (#645).
	wp := finishWorkPlane(part, part.WorkPlanes().AddByPointCloudFit(cloudPlaneFitSource{pc}))
	s.recordEdit(part, labelWorkPlane)
	return wp, plane, nil
}

// SelectedPointCloud returns the point cloud selected in the browser, if any.
func (s *Session) SelectedPointCloud() (*pointcloud.PointCloud, bool) {
	for _, it := range s.selection.Items() {
		if h, ok := it.(PointCloudHandle); ok {
			return h.Cloud, true
		}
	}
	return nil, false
}

// targetPointCloud resolves which cloud the Point Cloud panel's display controls (point size,
// density, display mode, intensity ramp) act on. When the active part holds exactly one scan that
// cloud is always the target, so a lone scan needs no manual selection; with several attached the
// user must select one, falling back to the browser selection (or none). Viewport tools (Move /
// Crop / Fit) keep using SelectedPointCloud — they act on an explicitly chosen cloud (#645).
func (s *Session) targetPointCloud() (*pointcloud.PointCloud, bool) {
	if part, err := activePart(s); err == nil {
		if clouds := part.PointClouds(); clouds.Count() == 1 {
			return clouds.Item(0), true
		}
	}
	return s.SelectedPointCloud()
}

// FitSelectedCloudPlane fits a work plane to the browser-selected cloud — the Point Cloud panel's
// Fit Work Plane command. It errors when no cloud is selected.
func (s *Session) FitSelectedCloudPlane() (*feature.WorkPlane, error) {
	pc, ok := s.SelectedPointCloud()
	if !ok {
		return nil, errors.New("app: select a point cloud to fit a work plane to")
	}
	wp, _, err := s.CreatePointCloudPlane(pc.Name())
	return wp, err
}

// canFitPointCloudPlane enables Fit Work Plane: a point cloud is selected and not in a sketch.
func canFitPointCloudPlane(s *Session) bool {
	_, ok := s.SelectedPointCloud()
	return ok && !s.InSketch()
}

// SetTargetPointCloudDisplayMode changes the target cloud's display mode. The target is the sole
// attached scan when there is only one, else the selected cloud (see targetPointCloud); it errors
// when several scans are attached and none is selected.
func (s *Session) SetTargetPointCloudDisplayMode(mode types.PointCloudDisplayMode) error {
	pc, ok := s.targetPointCloud()
	if !ok {
		return errors.New("app: select a point cloud to change its display mode")
	}
	if !pc.SetDisplayMode(mode) {
		return fmt.Errorf("app: invalid point cloud display mode %q", mode)
	}
	return nil
}

// canSetTargetPointCloudDisplayMode enables the ribbon dropdown when a target cloud resolves — a
// lone attached scan, or the selected one when several are attached.
func canSetTargetPointCloudDisplayMode(s *Session) bool {
	_, ok := s.targetPointCloud()
	return ok
}
