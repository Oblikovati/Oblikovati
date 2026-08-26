// SPDX-License-Identifier: GPL-2.0-only

package command

import (
	"errors"
	"slices"
)

// errTransactionClosed reports use of a transaction that has already committed or
// aborted.
var errTransactionClosed = errors.New("command: transaction is not open")

// State is a transaction's lifecycle state (the TransactionStateEnum).
type State uint8

const (
	// Open: recording mutations; not yet committed or aborted.
	Open State = iota
	// Committed: folded into its parent or pushed to the history as one undo step.
	Committed
	// Aborted: its recorded mutations were reverted and discarded.
	Aborted
)

// Transaction records a sequence of mutations as one named undo unit. It is opened
// with [History.Begin], built with [Transaction.Do], and closed with
// [Transaction.Commit] (keep) or [Transaction.Abort] (roll back). Beginning a
// transaction while one is open creates a nested child that folds into its parent
// on commit (replaces COM nested/global transactions).
type Transaction struct {
	h         *History
	label     string
	cmds      []Command
	parent    *Transaction
	state     State
	mergePrev bool
}

// Begin opens a transaction. If one is already open, the new one nests inside it.
func (h *History) Begin(label string) *Transaction {
	t := &Transaction{h: h, label: label, parent: h.open, state: Open}
	h.open = t
	return t
}

// Do applies a command and records it into this transaction.
func (t *Transaction) Do(c Command) error { return t.record(c) }

// record applies c and appends it to the transaction's command list.
func (t *Transaction) record(c Command) error {
	if t.state != Open {
		return errTransactionClosed
	}
	if err := c.Apply(); err != nil {
		return err
	}
	t.cmds = append(t.cmds, c)
	return nil
}

// Label returns the transaction's undo label.
func (t *Transaction) Label() string { return t.label }

// State returns the transaction's lifecycle state.
func (t *Transaction) State() State { return t.state }

// MergeWithPrevious marks this top-level transaction to combine with the previous
// committed step, so the two undo as one (PBI-047). Ignored for nested transactions.
func (t *Transaction) MergeWithPrevious() { t.mergePrev = true }

// Commit closes the transaction. A nested transaction folds its work into its
// parent (no history change yet); a top-level one pushes a single [Batch] onto the
// history (or merges with the previous step) and fires one coalesced notification.
func (t *Transaction) Commit() error {
	if t.state != Open {
		return errTransactionClosed
	}
	t.state = Committed
	t.h.open = t.parent
	if t.parent != nil {
		t.parent.cmds = append(t.parent.cmds, t.batch())
		return nil
	}
	t.commitToHistory()
	return nil
}

// commitToHistory pushes the transaction's batch as one undo step, merging with the
// previous step if requested, and clears the redo stack.
func (t *Transaction) commitToHistory() {
	batch := t.batch()
	h := t.h
	if t.mergePrev && len(h.done) > 0 {
		prev := h.done[len(h.done)-1]
		h.done[len(h.done)-1] = NewBatch(prev.Label(), prev, batch)
	} else {
		h.done = append(h.done, batch)
	}
	h.undone = nil
	h.notify()
}

// Abort reverts every recorded mutation in reverse order, restoring the
// pre-transaction state exactly, then discards the transaction.
func (t *Transaction) Abort() error {
	if t.state != Open {
		return errTransactionClosed
	}
	t.state = Aborted
	t.h.open = t.parent
	for _, v := range slices.Backward(t.cmds) {
		if err := v.Revert(); err != nil {
			return err
		}
	}
	return nil
}

// batch wraps the recorded commands in a single labeled command, preserving the
// transaction's display name as the undo label even for a single mutation.
func (t *Transaction) batch() Command {
	return NewBatch(t.label, t.cmds...)
}
