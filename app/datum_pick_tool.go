// SPDX-License-Identifier: GPL-2.0-only

package app

// DatumPickTool is the guided-pick interaction behind every Work Features ribbon entry:
// activated with nothing pre-selected, it restricts the selection filter to the kinds the
// constructor needs, prompts the user, collects picks (in the 3D view or browser) into the
// selection, and auto-commits the moment enough are gathered — mirroring Create 2D Sketch.
// It reuses the Session's Create*Work* methods (which read the selection) for the commit, so
// the interactive and pre-selected paths build the datum the same way.
//
// It builds a plane, an axis or a point: the flavour is entirely in the filter/ready/create
// triple, which is why one tool can back all 22 constructors the Work Features panel offers
// (#2043, #2044) instead of one tool type per datum kind.
type DatumPickTool struct {
	name   string
	prompt string
	filter *SelectionFilter
	ready  func(*Session) bool  // enough picked to build?
	create func(*Session) error // build from the current selection
	prev   *SelectionFilter
	sess   *Session
}

// Name implements [Tool].
func (t *DatumPickTool) Name() string { return t.name }

// Start restricts selection to the tool's kinds and clears any prior picks so the tool
// gathers a fresh set.
func (t *DatumPickTool) Start(s *Session) {
	t.sess = s
	t.prev = s.Selection().Filter()
	s.Selection().Clear()
	s.Selection().SetFilter(t.filter)
}

// Pick adds the clicked entity to the selection (the constructor reads it on commit).
func (t *DatumPickTool) Pick(s *Session, sel Selectable) { s.Selection().Add(sel) }

// CanCommit is true once the selection satisfies the constructor.
func (t *DatumPickTool) CanCommit() bool { return t.sess != nil && t.ready(t.sess) }

// AutoCommitOnPick finishes the tool as soon as the last needed pick lands.
func (t *DatumPickTool) AutoCommitOnPick() bool { return true }

// Commit restores the prior filter and builds the datum from the gathered picks.
func (t *DatumPickTool) Commit(s *Session) error {
	s.Selection().SetFilter(t.prev)
	return t.create(s)
}

// Cancel restores the prior filter with no change.
func (t *DatumPickTool) Cancel(s *Session) { s.Selection().SetFilter(t.prev) }

// Prompt is the status-bar guidance shown while the tool gathers its picks.
func (t *DatumPickTool) Prompt(*Session) string { return t.prompt }

// startDatum is a Work Features command action: build the datum immediately when the current
// selection already satisfies the constructor, otherwise start the guided tool so the click
// always does something (the fix for an inert, pre-selection-gated button).
func startDatum(makeTool func() *DatumPickTool) func(*Session) error {
	return func(s *Session) error {
		t := makeTool()
		if t.ready(s) {
			return t.create(s)
		}
		s.StartTool(t)
		return nil
	}
}

// canStartWorkFeature enables the Work Features entries whenever a part is active and no sketch
// is open — they are always live (like Create 2D Sketch) and guide the pick when nothing is
// pre-selected, rather than greying out.
func canStartWorkFeature(s *Session) bool {
	if s.InSketch() {
		return false
	}
	_, err := activePart(s)
	return err == nil
}

// discardResult adapts a Create*Work* method that returns the created datum to the error-only
// shape DatumPickTool.create takes.
func discardResult[T any](create func(*Session) (T, error)) func(*Session) error {
	return func(s *Session) error {
		_, err := create(s)
		return err
	}
}
