// SPDX-License-Identifier: GPL-2.0-only

package command

import "errors"

// errTransactionOpen guards operations that may not run while a transaction is
// being recorded (a bare undo/redo would corrupt the open unit).
var errTransactionOpen = errors.New("command: a transaction is open")

// History is the undo/redo store: a stack of committed commands plus the redo
// stack of undone ones. Edits enter through [History.Do] (or a [Transaction]); a
// single coalesced [History.OnChange] callback fires per committed step so the
// recompute/notification layer (M07+) updates once, not per sub-edit.
type History struct {
	done        []Command
	undone      []Command
	checkpoints []CheckPoint
	onChange    func()
	suppressed  bool
	open        *Transaction // the innermost open transaction, or nil
}

// NewHistory returns an empty history.
func NewHistory() *History {
	return &History{}
}

// OnChange registers the coalesced change hook (recompute / notification). It is
// invoked once after each committed Do, Undo, Redo, or checkpoint move — unless
// notifications are suppressed.
func (h *History) OnChange(fn func()) { h.onChange = fn }

// Do applies a command and records it as a new undo step, clearing the redo stack.
// If a transaction is open, the command is recorded into it instead (so any edit
// performed mid-transaction joins that undo unit, regardless of who issued it).
func (h *History) Do(c Command) error {
	if h.open != nil {
		return h.open.record(c)
	}
	if err := c.Apply(); err != nil {
		return err
	}
	h.done = append(h.done, c)
	h.undone = nil
	h.notify()
	return nil
}

// Record appends an already-applied command as a new undo step *without* applying
// it, then clears the redo stream and fires one coalesced change. Use it (rather
// than [History.Do]) for snapshot events whose mutation has already happened in the
// model — the app edits the model, captures the before/after recipe, and records
// the resulting [RecipeEvent]. If a transaction is open, the command joins it.
func (h *History) Record(c Command) {
	if h.open != nil {
		h.open.cmds = append(h.open.cmds, c)
		return
	}
	h.done = append(h.done, c)
	h.undone = nil
	h.notify()
}

// Len returns the number of committed undo steps.
func (h *History) Len() int { return len(h.done) }

// Labels returns the labels of the committed steps, oldest first — the undo
// history as shown in a menu (replaces TransactionsEnumerator).
func (h *History) Labels() []string {
	return labelsOf(h.done)
}

// Clear empties both stacks (e.g. on document close). A transaction must not be open.
func (h *History) Clear() error {
	if h.open != nil {
		return errTransactionOpen
	}
	h.done, h.undone = nil, nil
	return nil
}

// SuppressNotifications turns the coalesced change hook off or on. Turning it back
// on fires one update, so a long batch performed under suppression results in a
// single recompute/notification (PBI-047).
func (h *History) SuppressNotifications(on bool) {
	h.suppressed = on
	if !on {
		h.notify()
	}
}

// notify fires the change hook unless suppressed or no hook is set.
func (h *History) notify() {
	if h.onChange != nil && !h.suppressed {
		h.onChange()
	}
}

// labelsOf extracts the labels of a command slice.
func labelsOf(cmds []Command) []string {
	out := make([]string, len(cmds))
	for i, c := range cmds {
		out[i] = c.Label()
	}
	return out
}
