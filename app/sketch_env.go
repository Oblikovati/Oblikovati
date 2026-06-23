// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// The sketch environment — Inventor's "Create 2D Sketch" → edit → "Finish Sketch"
// flow. Creating a sketch adds it to the active part and opens it for editing; while a
// sketch is active the Sketch ribbon tools are enabled and clicks place geometry in its
// plane (see screenToSketch). Finishing recomputes the part so downstream features see
// the new/edited sketch. The geometry lives in model/sketch; the Session only tracks
// which sketch is being edited (activeSketch).

// sketchHost is the active document's content a sketch is authored on — a part or an assembly.
// Both own a sketch collection and the parameter DAG dimensions share, and both recompute, so the
// sketch environment opens, finishes, and recomputes against either without knowing which (#766,
// mirroring the content-agnostic router seam).
type sketchHost interface {
	Sketches() *sketch.Sketches
	Parameters() *param.Parameters
	Recompute()
}

// activeSketchHost resolves the active document's content as a sketch host, erroring when there is
// no active document or its content hosts no sketches.
func activeSketchHost(s *Session) (sketchHost, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, errors.New("app: no active document")
	}
	host, ok := d.Content().(sketchHost)
	if !ok {
		return nil, errors.New("app: the active document hosts no sketches")
	}
	return host, nil
}

// CreateSketch adds a new sketch on the given plane to the active document (part or assembly) and
// enters the sketch environment to edit it. The sketch shares the host's parameter DAG so its
// dimension expressions resolve against the host's parameters. It errors when there is no sketch
// host active or a sketch is already being edited (Inventor forbids nesting sketch edits).
func (s *Session) CreateSketch(plane sketch.Plane) (*sketch.Sketch, error) {
	if s.activeSketch != nil {
		return nil, errors.New("app: already editing a sketch (finish it first)")
	}
	host, err := activeSketchHost(s)
	if err != nil {
		return nil, err
	}
	sk := host.Sketches().Add(plane)
	sk.SetParameters(host.Parameters()) // dimension expressions resolve in the host's table
	autoProjectOrigin(host, sk)
	s.EnterSketch(sk)
	return sk, nil
}

// autoProjectOrigin projects the part's origin centre point into a freshly created sketch, so
// every new sketch carries the projected origin as a constrainable reference at (0,0) — the
// Inventor default the bug report (#1262) asked for. It is a no-op for hosts without an origin
// centre point (an assembly), and the projection is associative like any other (it re-derives
// through recompute via the [feature.OriginCenter] reference).
func autoProjectOrigin(host sketchHost, sk *sketch.Sketch) {
	part, ok := host.(*compdef.PartComponentDefinition)
	if !ok {
		return
	}
	sk.ProjectPoint(compdef.NewWorkPointRefSource(part, feature.OriginCenter))
}

// CreateSketchOnOrigin creates a sketch on one of the part's origin planes (XY/XZ/YZ),
// the common default when no face is selected.
func (s *Session) CreateSketchOnOrigin(p OriginPlane) (*sketch.Sketch, error) {
	return s.CreateSketch(p.plane())
}

// CreateSketchOnSelectedPlane starts a sketch on the selected sketch host (a work plane
// OR a planar face picked in the 3D view or the browser), falling back to the XY origin
// plane when nothing usable is selected — the action behind the Create 2D Sketch ribbon
// command.
func (s *Session) CreateSketchOnSelectedPlane() (*sketch.Sketch, error) {
	if plane, ok := s.SelectedSketchHostPlane(); ok {
		return s.CreateSketch(plane)
	}
	return s.CreateSketchOnOrigin(OriginXY)
}

// SelectedSketchHostPlane returns the sketch plane of the first selected valid host — a
// work plane or a planar face — so a pre-selected face (not just a work plane) sketches
// immediately. ok is false when nothing in the selection can host a sketch.
func (s *Session) SelectedSketchHostPlane() (sketch.Plane, bool) {
	for _, it := range s.selection.Items() {
		if h, ok := it.(WorkPlaneHandle); ok {
			return h.Plane.Plane(), true
		}
		if h, ok := it.(FaceHandle); ok {
			if pl, ok := sketchPlaneFromFace(h); ok {
				return pl, true
			}
		}
	}
	return sketch.Plane{}, false
}

// sketchPlaneFromFace derives a sketch plane from a picked planar face (its underlying
// geometry plane), or ok=false for a non-planar face — the seam that lets a planar face
// act as a sketch host wherever a work plane can. The face's geometry is the same plane
// pickedPlaneRef uses for plane-driven features (feature_edit.go).
func sketchPlaneFromFace(fh FaceHandle) (sketch.Plane, bool) {
	pl, ok := fh.Face.Geometry().(geom.Plane)
	if !ok {
		return sketch.Plane{}, false
	}
	sp, err := sketch.NewPlane(pl.Origin, pl.UAxis, pl.VAxis)
	if err != nil {
		return sketch.Plane{}, false
	}
	return sp, true
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
	if host, err := activeSketchHost(s); err == nil {
		host.Recompute()
		if rs, ok := host.(recipeStore); ok {
			s.recordEdit(rs, "Sketch")
		}
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
	_, err := activeSketchHost(s)
	return err == nil
}
