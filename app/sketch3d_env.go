// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/sketch"
)

// The 3D-sketch environment — Inventor's "3D Sketch" → edit → "Finish Sketch" flow. A 3D
// sketch has no host plane: its geometry lives directly in model space, so creating one
// just adds it to the active part and opens it for editing (no plane pick, no camera
// swing). While a 3D sketch is active the 3D-sketch tools place geometry in model space;
// finishing recomputes the part. The Session only tracks which 3D sketch is being edited
// (activeSketch3D); the geometry lives in model/sketch (M22-F12).

// CreateSketch3D adds a new 3D sketch to the active part and enters its environment. It
// errors when there is no active part or any sketch (2D or 3D) is already being edited
// (Inventor forbids nesting sketch edits).
func (s *Session) CreateSketch3D() (*sketch.Sketch3D, error) {
	if s.activeSketch != nil || s.activeSketch3D != nil {
		return nil, errors.New("app: already editing a sketch (finish it first)")
	}
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	s.EnsureActiveEditBaseline() // capture the pre-sketch state so "Create 3D Sketch" is its own step (#1270)
	sk := part.Sketches3D().Add()
	s.EnterSketch3D(sk)
	s.RecordActiveEdit("Create 3D Sketch")
	return sk, nil
}

// EnterSketch3D activates a 3D sketch for editing; ExitSketch3D leaves it. Unlike a 2D
// sketch there is no plane to face, so the camera is left as the user had it.
func (s *Session) EnterSketch3D(sk *sketch.Sketch3D) {
	s.activeSketch3D = sk
	s.emitSketchEdit(sk.Seq(), sk.Name(), true) // SketchEvents surface (#148)
	sk.Edit()
}

// ExitSketch3D leaves the 3D-sketch environment.
func (s *Session) ExitSketch3D() {
	if s.activeSketch3D != nil {
		s.emitSketchEdit(s.activeSketch3D.Seq(), s.activeSketch3D.Name(), false) // SketchEvents surface (#148)
		s.activeSketch3D.ExitEdit()
		s.activeSketch3D = nil
	}
}

// ActiveSketch3D returns the 3D sketch being edited, or nil.
func (s *Session) ActiveSketch3D() *sketch.Sketch3D { return s.activeSketch3D }

// InSketch3D reports whether a 3D sketch is currently being edited — the predicate that
// enables the 3D-sketch tools and the Finish Sketch command.
func (s *Session) InSketch3D() bool { return s.activeSketch3D != nil }

// FinishSketch3D leaves the 3D-sketch environment and recomputes the part. It errors when
// no 3D sketch is being edited.
func (s *Session) FinishSketch3D() error {
	if s.activeSketch3D == nil {
		return errors.New("app: not editing a 3D sketch")
	}
	if s.tool != nil {
		s.CancelTool()
	}
	s.ExitSketch3D()
	if part, err := activePart(s); err == nil {
		part.Recompute()
		s.recordEdit(part, "3D Sketch")
	}
	return nil
}

// CanCreateSketch3D reports whether a new 3D sketch may be started (a part is active and
// no sketch edit is open) — the enable predicate for the 3D Sketch command.
func (s *Session) CanCreateSketch3D() bool {
	if s.activeSketch != nil || s.activeSketch3D != nil {
		return false
	}
	_, err := activePart(s)
	return err == nil
}
