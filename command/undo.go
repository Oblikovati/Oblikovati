// SPDX-License-Identifier: GPL-2.0-only

package command

import "errors"

// errNothingToUndo / errNothingToRedo report empty stacks.
var (
	errNothingToUndo = errors.New("command: nothing to undo")
	errNothingToRedo = errors.New("command: nothing to redo")
)

// CanUndo reports whether there is a committed step to undo.
func (h *History) CanUndo() bool { return len(h.done) > 0 }

// CanRedo reports whether there is an undone step to redo.
func (h *History) CanRedo() bool { return len(h.undone) > 0 }

// UndoLabels returns the labels of committed steps, oldest first (the undo
// enumerator). RedoLabels returns the undone steps, most-recently-undone first.
func (h *History) UndoLabels() []string { return labelsOf(h.done) }

// RedoLabels returns the labels of steps available to redo, in redo order.
func (h *History) RedoLabels() []string {
	out := make([]string, len(h.undone))
	for i, c := range h.undone {
		out[len(h.undone)-1-i] = c.Label()
	}
	return out
}

// Undo reverts the most recent committed step and moves it to the redo stack,
// firing one coalesced change. It errors if a transaction is open or nothing is
// left to undo. (Maps to TransactionManager.UndoTransaction.)
func (h *History) Undo() error {
	if h.open != nil {
		return errTransactionOpen
	}
	if err := h.undoOne(); err != nil {
		return err
	}
	h.notify()
	return nil
}

// Redo re-applies the most recently undone step. (Maps to RedoTransaction.)
func (h *History) Redo() error {
	if h.open != nil {
		return errTransactionOpen
	}
	if err := h.redoOne(); err != nil {
		return err
	}
	h.notify()
	return nil
}

// undoOne reverts one step without notifying — the building block for Undo and
// GoToCheckPoint (which coalesces many steps into one notification).
func (h *History) undoOne() error {
	if !h.CanUndo() {
		return errNothingToUndo
	}
	c := h.done[len(h.done)-1]
	if err := c.Revert(); err != nil {
		return err
	}
	h.done = h.done[:len(h.done)-1]
	h.undone = append(h.undone, c)
	return nil
}

// redoOne re-applies one step without notifying.
func (h *History) redoOne() error {
	if !h.CanRedo() {
		return errNothingToRedo
	}
	c := h.undone[len(h.undone)-1]
	if err := c.Apply(); err != nil {
		return err
	}
	h.undone = h.undone[:len(h.undone)-1]
	h.done = append(h.done, c)
	return nil
}
