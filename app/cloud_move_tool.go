// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/math"
	"oblikovati.org/model/pointcloud"
)

// Interactive point-cloud move (M17-F06, #645): while the Move tool is active, a left-drag in the
// viewport translates the selected cloud in the plane facing the camera, recomputing the part on
// each step so the datums built on the cloud (fit planes, anchored work/sketch points) follow it
// live. The drag is screen-parallel — the cloud stays under the cursor regardless of the view.

// CloudMoveTool arms interactive dragging of the selected point cloud. It carries no clicks of its
// own; its presence as the active tool lets the viewport's drag handler move the cloud.
type CloudMoveTool struct{}

// NewCloudMoveTool returns the point-cloud move tool.
func NewCloudMoveTool() *CloudMoveTool { return &CloudMoveTool{} }

func (*CloudMoveTool) Name() string              { return "Move Point Cloud" }
func (*CloudMoveTool) Start(*Session)            {}
func (*CloudMoveTool) Pick(*Session, Selectable) {}
func (*CloudMoveTool) CanCommit() bool           { return false } // drag-driven, not click/OK-driven
func (*CloudMoveTool) Commit(*Session) error     { return nil }
func (*CloudMoveTool) Cancel(*Session)           {}

// cloudMoveDrag is the in-progress interactive translation of a cloud.
type cloudMoveDrag struct {
	cloud  *pointcloud.PointCloud
	start  math.Matrix4 // the cloud's transform at drag start
	origin math.Point3  // the drag plane's point (the cloud's centre at start)
	normal math.Vector3 // the drag plane's normal (the camera forward at start)
	from   math.Point3  // the world point under the cursor at drag start
	active bool
}

// StartMoveSelectedCloud starts the Move tool on the browser-selected cloud — the Point Cloud
// panel's Move command. It errors when no cloud is selected.
func (s *Session) StartMoveSelectedCloud() error {
	if _, ok := s.SelectedPointCloud(); !ok {
		return errors.New("app: select a point cloud to move")
	}
	s.StartTool(NewCloudMoveTool())
	return nil
}

// canMoveSelectedCloud enables Move: a point cloud is selected and we are not in a sketch.
func canMoveSelectedCloud(s *Session) bool {
	_, ok := s.SelectedPointCloud()
	return ok && !s.InSketch()
}

// CloudMoveActive reports whether the Move tool is the active tool (so the viewport routes a
// left-drag to moving the cloud rather than selecting).
func (s *Session) CloudMoveActive() bool {
	_, ok := s.activeToolOK[*CloudMoveTool]()
	return ok
}

// CloudDragActive reports whether a cloud drag is in progress.
func (s *Session) CloudDragActive() bool { return s.cloudMove.active }

// BeginCloudDrag starts dragging the selected cloud from the cursor pixel. It returns false (no
// drag) when the Move tool is not active, no cloud is selected, or the cursor ray misses the drag
// plane.
func (s *Session) BeginCloudDrag(px, py float64) bool {
	if !s.CloudMoveActive() {
		return false
	}
	pc, ok := s.SelectedPointCloud()
	if !ok {
		return false
	}
	cam := s.Camera()
	o, d := cam.RayThrough(px, py)
	center, normal := pc.RangeBox().Center(), cam.Forward()
	from, ok := rayPlane(o, d, center, normal)
	if !ok {
		return false
	}
	s.cloudMove = cloudMoveDrag{cloud: pc, start: pc.Transform(), origin: center, normal: normal, from: from, active: true}
	return true
}

// UpdateCloudDrag moves the dragged cloud so the grabbed point tracks the cursor, then recomputes
// the part so the cloud-derived datums follow.
func (s *Session) UpdateCloudDrag(px, py float64) {
	if !s.cloudMove.active {
		return
	}
	cam := s.Camera()
	o, d := cam.RayThrough(px, py)
	to, ok := rayPlane(o, d, s.cloudMove.origin, s.cloudMove.normal)
	if !ok {
		return
	}
	delta := s.cloudMove.from.VectorTo(to)
	s.cloudMove.cloud.SetTransform(translateMatrix(s.cloudMove.start, delta))
	s.RecomputeAfterPointCloudMove()
}

// CommitCloudDrag ends the drag, recording the move as one undoable edit.
func (s *Session) CommitCloudDrag() {
	if s.cloudMove.active {
		if part, err := activePart(s); err == nil {
			s.recordEdit(part, "Move Point Cloud")
		}
	}
	s.cloudMove = cloudMoveDrag{}
}

// translateMatrix adds a world translation to a placement's translation column.
func translateMatrix(m math.Matrix4, v math.Vector3) math.Matrix4 {
	c := m.Cells()
	c[3] += v.X
	c[7] += v.Y
	c[11] += v.Z
	return math.Matrix4FromCells(c)
}
