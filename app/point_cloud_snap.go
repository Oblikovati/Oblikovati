// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// Snapped scan-point consumer (M17-F06, #645): a picked point on an attached cloud
// (PointCloudPointHandle) carries its model-space position, so a datum point can be anchored
// directly on the as-built scan data — the bridge from a reference cloud to model geometry.

// SelectedCloudPoint returns the model-space location of the snapped scan point selected in the
// viewport, if one is selected.
func (s *Session) SelectedCloudPoint() (math.Point3, bool) {
	if h, ok := s.SelectedCloudPointHandle(); ok {
		return h.Position(), true
	}
	return math.Point3{}, false
}

// SelectedCloudPointHandle returns the full snapped scan-point selection (the cloud and the picked
// point), so a consumer that needs the source cloud — like an associative work point — can reach it.
func (s *Session) SelectedCloudPointHandle() (PointCloudPointHandle, bool) {
	for _, it := range s.selection.Items() {
		if h, ok := it.(PointCloudPointHandle); ok {
			return h, true
		}
	}
	return PointCloudPointHandle{}, false
}

// CreateWorkPointAtSelectedCloudPoint adds a datum point at the snapped scan point selected in the
// viewport, then recomputes — the Point Cloud panel's Work Point command. The point keeps a live
// link to its cloud (provenance): it follows the cloud's placement and the link round-trips in the
// document. It errors when there is no active part or no scan point is selected.
func (s *Session) CreateWorkPointAtSelectedCloudPoint() (*feature.WorkPoint, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	h, ok := s.SelectedCloudPointHandle()
	if !ok {
		return nil, errors.New("app: select a point on a scan (snap to a cloud point) to place a work point")
	}
	wp := part.WorkPoints().AddByCloudPoint(newCloudPointSource(h.Cloud, h.Point))
	part.Recompute()
	s.recordEdit(part, labelWorkPoint)
	return wp, nil
}

// canWorkPointAtCloudPoint enables Work Point from a scan: a snapped scan point is selected and we
// are not in a sketch.
func canWorkPointAtCloudPoint(s *Session) bool {
	_, ok := s.SelectedCloudPoint()
	return ok && !s.InSketch()
}

// CreateSketchPointAtSelectedCloudPoint adds a sketch point at the snapped scan point selected in
// the viewport, projected onto the active sketch plane — the in-sketch counterpart of the Work
// Point command, so a 2D profile can be drawn against scanned features. It errors when there is no
// active 2D sketch or no scan point is selected.
func (s *Session) CreateSketchPointAtSelectedCloudPoint() (*sketch.Point, error) {
	sk := s.ActiveSketch()
	if sk == nil {
		return nil, errors.New("app: enter a 2D sketch to place a sketch point on a scan point")
	}
	at, ok := s.SelectedCloudPoint()
	if !ok {
		return nil, errors.New("app: select a point on a scan (snap to a cloud point) for a sketch point")
	}
	return sk.Points().Add(sk.ToSketch(at)), nil
}

// canSketchPointAtCloudPoint enables Sketch Point from a scan: in a 2D sketch with a snapped scan
// point selected.
func canSketchPointAtCloudPoint(s *Session) bool {
	_, ok := s.SelectedCloudPoint()
	return ok && s.InSketch()
}
