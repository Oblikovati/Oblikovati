// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/math"
	"oblikovati.org/model/pointcloud"
	"oblikovati.org/renderer"
)

// Interactive crop-box draw tool (M17-F06, #645): the user boxes a region of an attached cloud in
// the viewport with two corner clicks, and the cloud is cropped to the model-space bounds of the
// scan points inside that screen rectangle — so a working region can be isolated by eye, instead
// of only through a typed/wire model-space box (pointClouds.addCrop).

// CropBoxTool collects two viewport-pixel corners and, on the second click, crops its cloud to the
// scan points enclosed by the rectangle.
type CropBoxTool struct {
	cloud   *pointcloud.PointCloud
	corners []math.Point2 // viewport-pixel corners (not sketch-plane points)
}

// NewCropBoxTool returns a crop-box tool targeting cloud.
func NewCropBoxTool(cloud *pointcloud.PointCloud) *CropBoxTool { return &CropBoxTool{cloud: cloud} }

func (t *CropBoxTool) Name() string              { return "Crop Point Cloud" }
func (t *CropBoxTool) Start(*Session)            {}
func (t *CropBoxTool) Pick(*Session, Selectable) {}
func (t *CropBoxTool) CanCommit() bool           { return len(t.corners) == 2 }
func (t *CropBoxTool) AutoCommits() bool         { return true }
func (t *CropBoxTool) Cancel(*Session)           { t.corners = nil }

// ClickAt records a raw viewport-pixel corner (the rectangle is a screen region, not a plane).
func (t *CropBoxTool) ClickAt(_ *Session, px, py float64) {
	t.corners = append(t.corners, math.P2(math.Scalar(px), math.Scalar(py)))
}

// Commit crops the cloud to the scan points whose projection lands inside the boxed rectangle.
func (t *CropBoxTool) Commit(s *Session) error {
	if t.cloud == nil {
		return errors.New("crop: no target point cloud")
	}
	if len(t.corners) != 2 {
		return errors.New("crop: click two opposite corners")
	}
	box, ok := s.cloudPointsBoxInRect(t.cloud, t.corners[0], t.corners[1])
	if !ok {
		return errors.New("crop: no scan points inside the box")
	}
	t.cloud.AddCrop(box)
	if part, err := activePart(s); err == nil {
		part.Recompute()
		s.recordEdit(part, labelCropPointCloud)
	}
	return nil
}

// StartCropSelectedCloud starts the crop-box tool on the browser-selected cloud — the Point Cloud
// panel's Crop Box command. It errors when no cloud is selected.
func (s *Session) StartCropSelectedCloud() error {
	cloud, ok := s.SelectedPointCloud()
	if !ok {
		return errors.New("app: select a point cloud to crop with a box")
	}
	s.StartTool(NewCropBoxTool(cloud))
	return nil
}

// canCropSelectedCloud enables Crop Box: a point cloud is selected and we are not in a sketch.
func canCropSelectedCloud(s *Session) bool {
	_, ok := s.SelectedPointCloud()
	return ok && !s.InSketch()
}

// cloudPointsBoxInRect returns the model-space bounding box of the cloud's scan points whose
// projection lands inside the viewport rectangle spanned by corners a and b. ok is false when no
// point projects inside (an empty box would crop everything away).
func (s *Session) cloudPointsBoxInRect(pc *pointcloud.PointCloud, a, b math.Point2) (math.Box, bool) {
	rect := orderedRect(float64(a.X), float64(a.Y), float64(b.X), float64(b.Y))
	cam := s.Camera()
	box := math.EmptyBox()
	found := false
	for _, cp := range pc.CloudPoints() {
		m := pc.ToModelSpace(cp)
		if sx, sy, ok := renderer.Project(cam, regionNear, regionFar, m); ok && rect.containsPoint(sx, sy) {
			box = box.ExtendPoint(m)
			found = true
		}
	}
	return box, found
}

var _ Tool = (*CropBoxTool)(nil)
var _ PlaneClickTool = (*CropBoxTool)(nil)
var _ AutoCommitTool = (*CropBoxTool)(nil)
