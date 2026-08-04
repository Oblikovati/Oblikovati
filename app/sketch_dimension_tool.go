// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// DimensionTool is Inventor's General Dimension: pick the geometry, then click to place the
// value label. The placement click is what finishes the tool.
//
// The generic pick-and-auto-commit flow the other constraint tools use cannot express this
// (#2022). It committed as soon as its pick set was "ready", and a single line reads as ready
// because a line supplies the two points a distance dimension needs — so the first line pick
// created a length dimension and deactivated the tool. A second line could never be picked,
// which made the two-line angle dimension unreachable from the UI even though the code to
// build one was there and correct. Waiting for an explicit placement click keeps a one-line
// pick set open long enough for the user to say what they meant.
//
// The placement point is also the dimension's text position, so a dimension lands where the
// user put it instead of at a derived default offset.
type DimensionTool struct {
	dialogTool
	picks     []constraintPick
	placement math.Point2
	placed    bool
}

// NewDimensionTool returns the general dimension tool (length/distance, radius, or angle).
//
// Example: s.StartTool(NewDimensionTool()) — then click a line and click again to place it.
func NewDimensionTool() *DimensionTool { return &DimensionTool{} }

func (t *DimensionTool) Name() string { return "Dimension" }

// Cancel discards the in-progress picks and placement.
func (t *DimensionTool) Cancel(*Session) { t.picks, t.placed = nil, false }

// Accepts/Picked implement [SketchEntityTool] so the head can highlight valid geometry on
// hover and show what has been picked so far.
func (t *DimensionTool) Accepts(e sketch.Entity) bool { return acceptDimensionable(e) }
func (t *DimensionTool) Picked() []sketch.Entity      { return entitiesOf(t.picks) }

// PickSnap records a valid, not-already-picked entity together with its snap.
func (t *DimensionTool) PickSnap(ent sketch.Entity, snap SnapResult) {
	if !acceptDimensionable(ent) || t.contains(ent) {
		return
	}
	t.picks = append(t.picks, constraintPick{entity: ent, snap: snap})
}

// Pick records a clicked entity with no snap context (the generic-tool path).
func (t *DimensionTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(SketchEntityHandle); ok {
		t.PickSnap(h.Entity, SnapResult{})
	}
}

// ClickAt routes a viewport click: geometry the pick set still wants joins it, anything else
// places the label once the set already describes a dimension. Implements [PlaneClickTool],
// which is why the click arrives here with coordinates rather than as a bare entity pick.
func (t *DimensionTool) ClickAt(s *Session, px, py float64) {
	if t.placed {
		return
	}
	if ent, ok := s.pickSketchEntity(px, py); ok && t.wants(ent) {
		snap, _ := s.SnapAt(px, py)
		t.PickSnap(ent, snap)
		return
	}
	if !dimensionComplete(t.Picked()) {
		return // not enough geometry yet — a stray click must not commit anything
	}
	if p, ok := s.sketchClickPoint(px, py); ok {
		t.placement, t.placed = p, true
	}
}

// wants reports whether a clicked entity should JOIN the pick set rather than place the label.
// Once the set is complete only an entity that changes which dimension is meant is taken —
// otherwise clicking near existing geometry to place a label would silently absorb it.
func (t *DimensionTool) wants(ent sketch.Entity) bool {
	if t.contains(ent) || !acceptDimensionable(ent) {
		return false
	}
	ents := t.Picked()
	if !dimensionComplete(ents) {
		return true
	}
	return dimensionExtends(ents, ent)
}

// CanCommit reports whether the picks describe a dimension (this enables OK/Enter, which
// commits at the derived default position when no placement click was made).
func (t *DimensionTool) CanCommit() bool { return dimensionComplete(t.Picked()) }

// AutoCommits finishes the tool on the placement click — and only then, so a complete but
// still-extendable pick set (one line, which a second line would turn into an angle) stays
// open. This is the gate that #2022 was missing.
func (t *DimensionTool) AutoCommits() bool { return t.placed }

// Commit creates the dimension implied by the picks at its placed text position, and holds it
// as the session's pending dimension so the UI prompts for a value (Inventor's edit-on-place).
func (t *DimensionTool) Commit(s *Session) error {
	if s.activeSketch == nil {
		return errors.New("dimension: no active sketch")
	}
	created, err := addPlacedDimensionFor(s.activeSketch.DimensionConstraints(), s.DocumentUnits(),
		t.Picked(), t.placement, t.placed)
	if err != nil {
		return err
	}
	if t.placed {
		created.SetTextPoint(t.placement)
	}
	s.pendingDim = created
	return s.afterConstraint()
}

// Prompt guides the pick step, then the placement (Inventor's status-bar prompt).
func (t *DimensionTool) Prompt(*Session) string {
	ents := t.Picked()
	if !dimensionComplete(ents) {
		if len(t.picks) == 0 {
			return "Select points, a line, a circle, or two lines"
		}
		return "Select a second point"
	}
	if dimensionAwaitsSecondLine(ents) {
		return "Click to place the dimension, or select a second line for an angle"
	}
	return "Click to place the dimension"
}

func (t *DimensionTool) contains(e sketch.Entity) bool {
	for _, p := range t.picks {
		if p.entity == e {
			return true
		}
	}
	return false
}

// dimensionComplete reports whether the picks already describe a dimension: a circle (radius),
// a line (length) or two lines (angle), or two points (distance). One point alone does not.
func dimensionComplete(ents []sketch.Entity) bool {
	if len(filterCircles(ents)) >= 1 || len(filterLines(ents)) >= 1 {
		return true
	}
	return len(filterPoints(ents)) >= 2
}

// dimensionExtends reports whether adding ent changes which dimension is meant. Only the
// second line does: it turns a line's length into the angle between the two.
func dimensionExtends(ents []sketch.Entity, ent sketch.Entity) bool {
	if _, ok := ent.(*sketch.Line); !ok {
		return false
	}
	return dimensionAwaitsSecondLine(ents)
}

// dimensionAwaitsSecondLine reports whether the picks are exactly one line — the only state
// where another pick would change the dimension's kind.
func dimensionAwaitsSecondLine(ents []sketch.Entity) bool {
	return len(ents) == 1 && len(filterLines(ents)) == 1
}
