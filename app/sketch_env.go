// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// The sketch environment — Inventor's "Create 2D Sketch" → edit → "Finish Sketch"
// flow. Creating a sketch adds it to the active part and opens it for editing; while a
// sketch is active the Sketch ribbon tools are enabled and clicks place geometry in its
// plane (see screenToSketch). Finishing recomputes the part so downstream features see
// the new/edited sketch. The geometry lives in model/sketch; the Session only tracks
// which sketch is being edited (activeSketch).

// CreateSketch adds a new sketch on the given plane to the active part and enters the
// sketch environment to edit it. It errors when there is no active part or a sketch is
// already being edited (Inventor forbids nesting sketch edits).
func (s *Session) CreateSketch(plane sketch.Plane) (*sketch.Sketch, error) {
	if s.activeSketch != nil {
		return nil, errors.New("app: already editing a sketch (finish it first)")
	}
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	sk := part.Sketches().Add(plane)
	s.EnterSketch(sk)
	return sk, nil
}

// CreateSketchOnOrigin creates a sketch on one of the part's origin planes (XY/XZ/YZ),
// the common default when no face is selected.
func (s *Session) CreateSketchOnOrigin(p OriginPlane) (*sketch.Sketch, error) {
	return s.CreateSketch(p.plane())
}

// CreateSketchOnSelectedPlane starts a sketch on the selected work plane (picked in the
// 3D view or the browser), falling back to the XY origin plane when nothing usable is
// selected — the action behind the Create 2D Sketch ribbon command.
func (s *Session) CreateSketchOnSelectedPlane() (*sketch.Sketch, error) {
	if wp := s.SelectedWorkPlane(); wp != nil {
		return s.CreateSketch(wp.Plane())
	}
	return s.CreateSketchOnOrigin(OriginXY)
}

// SelectedWorkPlane returns the first selected work plane, or nil.
func (s *Session) SelectedWorkPlane() *feature.WorkPlane {
	for _, it := range s.selection.Items() {
		if h, ok := it.(WorkPlaneHandle); ok {
			return h.Plane
		}
	}
	return nil
}

// OriginPlane names one of the three origin work planes a sketch can start on.
type OriginPlane uint8

const (
	// OriginXY/OriginXZ/OriginYZ are the standard origin planes.
	OriginXY OriginPlane = iota
	OriginXZ
	OriginYZ
)

// plane maps an origin-plane selector to its sketch plane.
func (p OriginPlane) plane() sketch.Plane {
	switch p {
	case OriginXZ:
		return sketch.XZPlane()
	case OriginYZ:
		return sketch.YZPlane()
	default:
		return sketch.XYPlane()
	}
}

// FinishSketch leaves the sketch environment and recomputes the part so features that
// consume the sketch update. It errors when no sketch is being edited.
func (s *Session) FinishSketch() error {
	if s.activeSketch == nil {
		return errors.New("app: not editing a sketch")
	}
	if s.tool != nil {
		s.CancelTool() // an in-progress geometry tool is abandoned on finish
	}
	s.ExitSketch()
	if part, err := activePart(s); err == nil {
		part.Recompute()
		s.recordEdit(part, "Sketch")
	}
	return nil
}

// InSketch reports whether a sketch is currently being edited — the predicate that
// enables the Sketch ribbon tools and the Finish Sketch command.
func (s *Session) InSketch() bool { return s.activeSketch != nil }

// CanCreateSketch reports whether a new sketch may be started (a part is active and no
// sketch edit is open) — the enable predicate for the Create 2D Sketch command.
func (s *Session) CanCreateSketch() bool {
	if s.activeSketch != nil {
		return false
	}
	_, err := activePart(s)
	return err == nil
}
