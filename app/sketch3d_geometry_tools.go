// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"github.com/Oblikovati/oblikovati/math"
)

// Line3DTool is the 3D-sketch line interaction: the user places points in model space and
// each consecutive pair becomes a segment, building a connected polyline rail (Inventor's
// 3D Line). Points arrive via AddPoint (the head feeds model-space picks; e2e tests call
// it directly). Committing adds the segments to the active 3D sketch.
type Line3DTool struct {
	points []math.Point3
}

// NewLine3DTool returns a 3D line tool.
func NewLine3DTool() *Line3DTool { return &Line3DTool{} }

// Name implements [Tool].
func (t *Line3DTool) Name() string { return "3D Line" }

// Start implements [Tool]; the 3D line tool needs no selection filter.
func (t *Line3DTool) Start(*Session) {}

// Pick implements [Tool]; model-space picks arrive via AddPoint, not selectables.
func (t *Line3DTool) Pick(*Session, Selectable) {}

// AddPoint records a model-space vertex of the polyline.
func (t *Line3DTool) AddPoint(p math.Point3) { t.points = append(t.points, p) }

// CanCommit is true once at least two points define a segment.
func (t *Line3DTool) CanCommit() bool { return len(t.points) >= 2 }

// Commit adds the polyline's segments to the active 3D sketch as connected lines (later
// segments reuse the previous endpoint so the rail is a single chain).
func (t *Line3DTool) Commit(s *Session) error {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return errors.New("3D line: not editing a 3D sketch")
	}
	if !t.CanCommit() {
		return errors.New("3D line: need at least two points")
	}
	for i := 1; i < len(t.points); i++ {
		sk.AddLine3D(t.points[i-1], t.points[i])
	}
	return nil
}

// Cancel implements [Tool] (no state to restore).
func (t *Line3DTool) Cancel(*Session) {}

// Point3DTool places a single standalone point in the active 3D sketch.
type Point3DTool struct {
	point *math.Point3
}

// NewPoint3DTool returns a 3D point tool.
func NewPoint3DTool() *Point3DTool { return &Point3DTool{} }

// Name implements [Tool].
func (t *Point3DTool) Name() string { return "3D Point" }

// Start/Pick implement [Tool]; the point arrives via SetPoint.
func (t *Point3DTool) Start(*Session)            {}
func (t *Point3DTool) Pick(*Session, Selectable) {}

// SetPoint records the point's model-space position.
func (t *Point3DTool) SetPoint(p math.Point3) { t.point = &p }

// CanCommit is true once a position is set.
func (t *Point3DTool) CanCommit() bool { return t.point != nil }

// Commit adds the point to the active 3D sketch.
func (t *Point3DTool) Commit(s *Session) error {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return errors.New("3D point: not editing a 3D sketch")
	}
	if t.point == nil {
		return errors.New("3D point: no position set")
	}
	sk.AddPoint3D(*t.point)
	return nil
}

// Cancel implements [Tool].
func (t *Point3DTool) Cancel(*Session) {}

// compile-time assertions that the 3D sketch tools satisfy the Tool interface.
var (
	_ Tool = (*Line3DTool)(nil)
	_ Tool = (*Point3DTool)(nil)
)
