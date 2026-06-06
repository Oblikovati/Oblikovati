// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati/model/feature"

// WorkPlaneTool is the guided-pick interaction behind every Work Features ribbon button:
// activated with nothing pre-selected, it restricts the selection filter to the kinds the
// constructor needs, prompts the user, collects picks (in the 3D view or browser) into the
// selection, and auto-commits the moment enough are gathered — mirroring Create 2D Sketch.
// It reuses the Session's Create*WorkPlane methods (which read the selection) for the
// commit, so the interactive and pre-selected paths build the datum the same way.
type WorkPlaneTool struct {
	name   string
	prompt string
	filter *SelectionFilter
	ready  func(*Session) bool                        // enough picked to build?
	create func(*Session) (*feature.WorkPlane, error) // build from the current selection
	prev   *SelectionFilter
	sess   *Session
}

// Name implements [Tool].
func (t *WorkPlaneTool) Name() string { return t.name }

// Start restricts selection to the tool's kinds and clears any prior picks so the tool
// gathers a fresh set.
func (t *WorkPlaneTool) Start(s *Session) {
	t.sess = s
	t.prev = s.Selection().Filter()
	s.Selection().Clear()
	s.Selection().SetFilter(t.filter)
}

// Pick adds the clicked entity to the selection (the constructor reads it on commit).
func (t *WorkPlaneTool) Pick(s *Session, sel Selectable) { s.Selection().Add(sel) }

// CanCommit is true once the selection satisfies the constructor.
func (t *WorkPlaneTool) CanCommit() bool { return t.sess != nil && t.ready(t.sess) }

// AutoCommitOnPick finishes the tool as soon as the last needed pick lands.
func (t *WorkPlaneTool) AutoCommitOnPick() bool { return true }

// Commit restores the prior filter and builds the datum from the gathered picks.
func (t *WorkPlaneTool) Commit(s *Session) error {
	s.Selection().SetFilter(t.prev)
	_, err := t.create(s)
	return err
}

// Cancel restores the prior filter with no change.
func (t *WorkPlaneTool) Cancel(s *Session) { s.Selection().SetFilter(t.prev) }

// Prompt is the status-bar guidance shown while the tool gathers its picks.
func (t *WorkPlaneTool) Prompt(*Session) string { return t.prompt }

func newMidplaneWorkPlaneTool() *WorkPlaneTool {
	return &WorkPlaneTool{
		name: "Midplane", prompt: "Select two planes to bisect",
		filter: NewSelectionFilter(SelectWorkPlane),
		ready:  canMidplaneWorkPlane, create: (*Session).CreateMidplaneWorkPlane,
	}
}

func newThreePointWorkPlaneTool() *WorkPlaneTool {
	return &WorkPlaneTool{
		name: "Three Points", prompt: "Select three points or model vertices",
		filter: NewSelectionFilter(SelectWorkPoint, SelectVertex),
		ready:  canThreePointWorkPlane, create: (*Session).CreateThreePointWorkPlane,
	}
}

func newTangentWorkPlaneTool() *WorkPlaneTool {
	return &WorkPlaneTool{
		name: "Tangent to Face", prompt: "Select a plane, then a cylindrical/spherical face",
		filter: NewSelectionFilter(SelectWorkPlane, SelectFace),
		ready:  canTangentWorkPlane, create: (*Session).CreateTangentWorkPlane,
	}
}

func newNormalToAxisWorkPlaneTool() *WorkPlaneTool {
	return &WorkPlaneTool{
		name: "Normal to Axis", prompt: "Select an axis, then a point on it",
		filter: NewSelectionFilter(SelectWorkAxis, SelectWorkPoint, SelectVertex),
		ready:  canNormalToAxisWorkPlane, create: (*Session).CreateNormalToAxisWorkPlane,
	}
}

// startWorkPlane is a Work Features command action: build the datum immediately when the
// current selection already satisfies the constructor, otherwise start the guided tool so
// the click always does something (the fix for an inert, pre-selection-gated button).
func startWorkPlane(makeTool func() *WorkPlaneTool) func(*Session) error {
	return func(s *Session) error {
		t := makeTool()
		if t.ready(s) {
			_, err := t.create(s)
			return err
		}
		s.StartTool(t)
		return nil
	}
}

// canStartWorkPlane enables the Work Features buttons whenever a part is active and no
// sketch is open — the buttons are always live (like Create 2D Sketch) and guide the pick
// when nothing is pre-selected, rather than greying out.
func canStartWorkPlane(s *Session) bool {
	if s.InSketch() {
		return false
	}
	_, err := activePart(s)
	return err == nil
}
