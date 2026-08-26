// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"
	"slices"

	"oblikovati.org/model/sketch"
)

// Bend3DTool is the interactive 3D-sketch Bend command (issue #143 — Inventor's
// SketchArcs3D.AddAsBend): set the radius, then pick two connected 3D lines; the
// corner is replaced by a tangent arc and the maintaining bend constraint. The tool
// auto-applies on the second pick, so a pipe-run's corners can be bent click-by-click
// after adjusting the radius once.
type Bend3DTool struct {
	dialogTool
	lines  []*sketch.Line3D
	radius float64 // database units (cm)
}

// defaultBendRadius3D is the bend radius a fresh tool offers (0.5 cm = 5 mm).
const defaultBendRadius3D = 0.5

// NewBend3DTool returns a bend tool at the default radius.
func NewBend3DTool() *Bend3DTool { return &Bend3DTool{radius: defaultBendRadius3D} }

// Name implements [Tool].
func (t *Bend3DTool) Name() string { return "Bend" }

// Start/Cancel implement [Tool].
func (t *Bend3DTool) Cancel(*Session) { t.lines = nil }

// Params exposes the bend radius (the head shows it in the tool-params dialog).
func (t *Bend3DTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{{"Radius", func() float64 { return t.radius }, func(r float64) { t.radius = r }}}}
}

// Accepts reports line picks as valid (the hover-highlight cue).
func (t *Bend3DTool) Accepts(e sketch.Entity) bool {
	_, ok := e.(*sketch.Line3D)
	return ok
}

// Picked returns the gathered lines (the picked-highlight cue).
func (t *Bend3DTool) Picked() []sketch.Entity {
	out := make([]sketch.Entity, len(t.lines))
	for i, l := range t.lines {
		out[i] = l
	}
	return out
}

// Pick records a clicked 3D line (ignoring other kinds and repeats).
func (t *Bend3DTool) Pick(_ *Session, sel Selectable) {
	h, ok := sel.(SketchEntityHandle)
	if !ok {
		return
	}
	t.PickSnap(h.Entity, SnapResult{})
}

// PickSnap implements [SketchEntityTool]; the snap carries nothing a bend needs.
func (t *Bend3DTool) PickSnap(ent sketch.Entity, _ SnapResult) {
	l, ok := ent.(*sketch.Line3D)
	if !ok || t.contains(l) {
		return
	}
	t.lines = append(t.lines, l)
}

func (t *Bend3DTool) contains(l *sketch.Line3D) bool {
	return slices.Contains(t.lines, l)
}

// CanCommit is true once two lines are picked.
func (t *Bend3DTool) CanCommit() bool { return len(t.lines) >= 2 }

// AutoCommitOnPick applies the bend as soon as the second line is picked.
func (t *Bend3DTool) AutoCommitOnPick() bool { return true }

// Prompt guides the user (the status-bar hint).
func (t *Bend3DTool) Prompt(*Session) string {
	return "Set the radius, then select two connected lines to bend"
}

// Commit inserts the bend into the active 3D sketch.
func (t *Bend3DTool) Commit(s *Session) error {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return errors.New("bend: not editing a 3D sketch")
	}
	if !t.CanCommit() {
		return errors.New("bend: select two connected lines")
	}
	if _, err := sk.AddBend3D(t.lines[0], t.lines[1], t.radius); err != nil {
		return fmt.Errorf("bend: %w", err)
	}
	s.Select(nil)
	return nil
}
