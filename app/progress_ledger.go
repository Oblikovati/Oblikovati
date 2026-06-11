// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/event"
)

// ProgressCancelled fires (After) when the user cancels a progress bar; the events
// relay forwards it as a progress.cancelled push event (M05-F09).
type ProgressCancelled struct{ ID int }

// EventID implements event.Event.
func (ProgressCancelled) EventID() event.TypeID { return tidProgressCancelled }

// ProgressView is one live bar for the status bar's rendering: the innermost (most
// recently begun) is the one shown.
type ProgressView struct {
	ID        int
	Steps     int
	Step      int
	Message   string
	Cancelled bool
}

// ProgressLedger tracks the live progress bars — nesting-safe: each Begin gets its
// own id and bars end independently (a sub-operation's bar inside an outer one).
type ProgressLedger struct {
	nextID int
	order  []int
	bars   map[int]*ProgressView
}

// NewProgressLedger returns an empty ledger.
func NewProgressLedger() *ProgressLedger {
	return &ProgressLedger{nextID: 1, bars: map[int]*ProgressView{}}
}

// Begin starts a bar of steps steps (≥1) and returns its id.
func (l *ProgressLedger) Begin(steps int, message string) (int, error) {
	if steps < 1 {
		return 0, fmt.Errorf("app: progress needs at least 1 step, got %d", steps)
	}
	id := l.nextID
	l.nextID++
	l.bars[id] = &ProgressView{ID: id, Steps: steps, Message: message}
	l.order = append(l.order, id)
	return id, nil
}

// Update advances a bar (clamped to its step count), optionally replacing its
// message, and reports whether the user cancelled it.
func (l *ProgressLedger) Update(id, step int, message string) (cancelled bool, err error) {
	bar, ok := l.bars[id]
	if !ok {
		return false, fmt.Errorf("app: no progress bar %d", id)
	}
	if step > bar.Steps {
		step = bar.Steps
	}
	bar.Step = step
	if message != "" {
		bar.Message = message
	}
	return bar.Cancelled, nil
}

// End removes a bar.
func (l *ProgressLedger) End(id int) error {
	if _, ok := l.bars[id]; !ok {
		return fmt.Errorf("app: no progress bar %d", id)
	}
	delete(l.bars, id)
	for i, x := range l.order {
		if x == id {
			l.order = append(l.order[:i], l.order[i+1:]...)
			break
		}
	}
	return nil
}

// Innermost returns the most recently begun live bar (what the status bar shows).
func (l *ProgressLedger) Innermost() (ProgressView, bool) {
	if len(l.order) == 0 {
		return ProgressView{}, false
	}
	return *l.bars[l.order[len(l.order)-1]], true
}

// Progress returns the session's progress ledger.
func (s *Session) Progress() *ProgressLedger { return s.progress }

// CancelProgress marks a bar cancelled (the status bar's cancel control) and emits
// the event the owning add-in observes.
func (s *Session) CancelProgress(id int) error {
	bar, ok := s.progress.bars[id]
	if !ok {
		return fmt.Errorf("app: no progress bar %d", id)
	}
	if bar.Cancelled {
		return nil
	}
	bar.Cancelled = true
	event.Emit(s.bus, event.After, ProgressCancelled{ID: id})
	return nil
}
