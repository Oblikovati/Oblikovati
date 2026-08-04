// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
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
	s.EnsureActiveEditBaseline() // capture the pre-sketch state so "Create Sketch" is its own step (#1270)
	sk := host.Sketches().Add(plane)
	sk.SetParameters(host.Parameters()) // dimension expressions resolve in the host's table
	s.autoProjectOrigin(host, sk)
	s.EnterSketch(sk)
	// Creating the sketch is its own undo step, so the baseline for the in-sketch operations that
	// follow is the empty sketch — undoing them reverts each op without removing the sketch (#1270).
	s.RecordActiveEdit("Create Sketch")
	return sk, nil
}

// autoProjectOrigin projects the origin centre into a freshly created sketch, for a host that
// owns one. An assembly does not, so it is a no-op there.
func (s *Session) autoProjectOrigin(host sketchHost, sk *sketch.Sketch) {
	if part, ok := host.(*compdef.PartComponentDefinition); ok {
		s.AutoProjectOriginInto(part, sk)
	}
}

// AutoProjectOriginInto projects the part's origin centre point into a freshly created sketch, so
// the sketch carries the origin as a constrainable reference at (0,0) — the Inventor default the
// bug report (#1262) asked for. The projection is associative like any other (it re-derives
// through recompute via the [feature.OriginCenter] reference).
//
//	s.AutoProjectOriginInto(part, sk)
//
// It honours Application Options ▸ Sketch ▸ Autoproject part origin, which is on by default, and
// is exported so the wire path (sketch.create) applies the same rule as the interactive one: a
// sketch an add-in creates gets the same anchor as one drawn by hand (#2016).
func (s *Session) AutoProjectOriginInto(part *compdef.PartComponentDefinition, sk *sketch.Sketch) {
	if !s.appOptions.Sketch.AutoProjectOrigin {
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

// reattachActiveSketchAfterRestore re-resolves the active sketch after an undo/redo restored the
// part — which rebuilds the sketch objects, leaving s.activeSketch dangling. It re-finds the
// sketch by its stable id, re-enters edit on the fresh object, and clears the now-stale selection
// so editing continues seamlessly (#1270). A sketch undone past its own creation drops out of the
// sketch environment. Only the active document's edit is reattached (a background-document jump
// leaves it untouched).
func (s *Session) reattachActiveSketchAfterRestore(d *doc.Document) {
	if d != s.ActiveDocument() {
		return
	}
	if s.activeSketch != nil {
		s.reattach2DSketch(d)
	}
	if s.activeSketch3D != nil {
		s.reattach3DSketch(d)
	}
}

func (s *Session) reattach2DSketch(d *doc.Document) {
	host, ok := d.Content().(interface {
		Sketches() *sketch.Sketches
	})
	if !ok {
		return
	}
	sk, ok := host.Sketches().ByID(s.activeSketch.ID())
	if !ok {
		s.ExitSketch() // the sketch was undone away
		return
	}
	s.activeSketch = sk
	sk.Edit()
	s.selection.Clear()
}

func (s *Session) reattach3DSketch(d *doc.Document) {
	part, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return
	}
	sk, ok := part.Sketches3D().ByID(s.activeSketch3D.ID())
	if !ok {
		s.activeSketch3D = nil
		return
	}
	s.activeSketch3D = sk
	sk.Edit()
	s.selection.Clear()
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
