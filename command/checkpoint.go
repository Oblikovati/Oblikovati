// SPDX-License-Identifier: GPL-2.0-only

package command

import "fmt"

// CheckPoint is a lightweight marker for a model state: the history depth at the
// moment it was set, plus a label. Returning to it (via [History.GoToCheckPoint])
// undoes or redoes until the history is back at that depth — no geometry snapshot,
// just "remember the history length" (architecture core/06).
type CheckPoint struct {
	label string
	depth int
}

// Label returns the checkpoint's label.
func (c CheckPoint) Label() string { return c.label }

// Depth returns the committed-history length the checkpoint captured.
func (c CheckPoint) Depth() int { return c.depth }

// SetCheckPoint records the current history depth under label and remembers it for
// enumeration. It does not change any state.
func (h *History) SetCheckPoint(label string) CheckPoint {
	cp := CheckPoint{label: label, depth: len(h.done)}
	h.checkpoints = append(h.checkpoints, cp)
	return cp
}

// GoToCheckPoint moves the model back (or forward) to the checkpoint's depth by
// undoing or redoing committed steps, firing a single coalesced change. It errors
// if a transaction is open or the depth is unreachable (e.g. the steps needed to
// redo back up were discarded by an intervening edit).
func (h *History) GoToCheckPoint(cp CheckPoint) error {
	if h.open != nil {
		return errTransactionOpen
	}
	for len(h.done) > cp.depth {
		if err := h.undoOne(); err != nil {
			return err
		}
	}
	for len(h.done) < cp.depth {
		if err := h.redoOne(); err != nil {
			return fmt.Errorf("command: checkpoint depth %d unreachable: %w", cp.depth, err)
		}
	}
	h.notify()
	return nil
}

// CheckPoints returns the checkpoints set so far, in order.
func (h *History) CheckPoints() []CheckPoint {
	out := make([]CheckPoint, len(h.checkpoints))
	copy(out, h.checkpoints)
	return out
}

// ReleaseCheckPoint forgets a previously-set checkpoint, reporting whether it was
// found. The model state is unaffected.
func (h *History) ReleaseCheckPoint(cp CheckPoint) bool {
	for i, existing := range h.checkpoints {
		if existing == cp {
			h.checkpoints = append(h.checkpoints[:i], h.checkpoints[i+1:]...)
			return true
		}
	}
	return false
}
