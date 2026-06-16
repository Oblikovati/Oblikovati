// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"sort"

	"oblikovati.org/event"
	"oblikovati.org/kernel/ops"
)

// Select Other — Inventor's S10: when several objects stack up under the cursor, cycle through
// the occluded candidates (front to back) and commit the one you want, instead of always getting
// the front-most. The picker resolves every body the ray crosses (PickAll); the session owns the
// cycle state and applies the current candidate to the selection so each step is visible.

// MultiPicker is a Picker that can also return every selectable under a pixel, depth-sorted —
// the input to Select Other. The RayPicker implements it; a plain Picker (e.g. a test stub) does
// not, so BeginSelectOther is a no-op there.
type MultiPicker interface {
	PickAll(x, y float64, filter *SelectionFilter) []Selectable
}

// RayPicker also answers the depth-sorted multi-pick that Select Other cycles through.
var _ MultiPicker = (*RayPicker)(nil)

// PickAll returns one candidate per body the ray crosses, depth-sorted front to back — the
// occluded-geometry list Select Other cycles through. Each body resolves to the same selectable
// Pick would give it (its component occurrence under a permissive filter, otherwise its face or
// body), so cycling steps through distinct objects.
func (p *RayPicker) PickAll(x, y float64, filter *SelectionFilter) []Selectable {
	origin, dir := p.camera.RayThrough(x, y)
	var cands []pickCandidate
	for _, b := range p.bodies() {
		f, t, ok := ops.RayCastFaces(b, origin, dir, ops.DefaultQuality())
		if !ok {
			continue
		}
		if occ, ok := p.occurrenceForBody(b, filter); ok {
			cands = append(cands, pickCandidate{t, OccurrenceHandle{Occurrence: occ}})
		} else if sel, ok := facePick(f, b, filter); ok {
			cands = append(cands, pickCandidate{t, sel})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].t < cands[j].t })
	out := make([]Selectable, 0, len(cands))
	for _, c := range cands {
		out = append(out, c.sel)
	}
	return out
}

// selectOther is the in-progress Select Other cycle: the depth-sorted candidates and the index of
// the one currently applied to the selection.
type selectOther struct {
	cands  []Selectable
	index  int
	active bool
}

// SelectOtherActive reports whether a Select Other cycle is in progress.
func (s *Session) SelectOtherActive() bool { return s.selectOther.active }

// SelectOtherStatus returns the 1-based position and the total candidate count, for the cycle
// widget's "i / N" label.
func (s *Session) SelectOtherStatus() (pos, count int) {
	return s.selectOther.index + 1, len(s.selectOther.cands)
}

// BeginSelectOther starts a Select Other cycle at (x,y): it resolves every object under the cursor
// and selects the front-most. It is a no-op (returning false, so the caller keeps the plain pick)
// when the picker cannot enumerate candidates or fewer than two stack up — Select Other is only
// meaningful when something is actually occluded.
func (s *Session) BeginSelectOther(x, y float64) bool {
	mp, ok := s.picker.(MultiPicker)
	if !ok {
		return false
	}
	cands := mp.PickAll(x, y, s.selection.Filter())
	if len(cands) < 2 {
		return false
	}
	s.selectOther = selectOther{cands: cands, index: 0, active: true}
	s.applySelectOther()
	return true
}

// CycleSelectOther advances the cycle by delta (+1 next / −1 previous), wrapping, and applies the
// new candidate to the selection so it highlights.
func (s *Session) CycleSelectOther(delta int) {
	if !s.selectOther.active {
		return
	}
	n := len(s.selectOther.cands)
	s.selectOther.index = ((s.selectOther.index+delta)%n + n) % n
	s.applySelectOther()
}

// applySelectOther replaces the selection with the current candidate and announces the change.
func (s *Session) applySelectOther() {
	s.selection.Clear()
	if s.selection.Add(s.selectOther.cands[s.selectOther.index]) {
		event.Emit(s.bus, event.After, SelectionChanged{Count: s.selection.Count()})
	}
}

// CommitSelectOther ends the cycle, keeping the current candidate selected.
func (s *Session) CommitSelectOther() { s.selectOther = selectOther{} }

// CancelSelectOther ends the cycle (e.g. on Escape); the current candidate stays selected, as in
// Inventor where Esc leaves the last-highlighted object picked.
func (s *Session) CancelSelectOther() { s.selectOther = selectOther{} }
