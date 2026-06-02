// SPDX-License-Identifier: GPL-2.0-only

package command

// Command is one reversible, named model mutation — the unit of undo. Apply and
// Revert are exact inverses: Revert after Apply must restore the prior state.
// Commands are self-contained (they capture their own target), so a [Batch] can
// group edits across documents.
type Command interface {
	// Label is the undo-menu text (was Transaction.DisplayName).
	Label() string
	// Apply performs the mutation.
	Apply() error
	// Revert undoes exactly what Apply did.
	Revert() error
}

// Func is a [Command] built from apply/revert closures. It exists so generic
// mutations (and tests) have a command before the typed domain commands
// (AddFeature, SetParameter, …) arrive with their milestones. A closure that
// captures the prior value at Apply time (not at construction) makes redo correct
// after intervening edits.
type Func struct {
	label  string
	apply  func() error
	revert func() error
}

// NewFunc builds a closure command. apply and revert must be inverses.
func NewFunc(label string, apply, revert func() error) *Func {
	return &Func{label: label, apply: apply, revert: revert}
}

// Label returns the undo text.
func (c *Func) Label() string { return c.label }

// Apply runs the apply closure.
func (c *Func) Apply() error { return c.apply() }

// Revert runs the revert closure.
func (c *Func) Revert() error { return c.revert() }

// Batch is a composite command: several sub-commands that undo as one step. It is
// how a transaction is committed (parametric-cad §8) and how nested/global
// transactions and merges are represented. Apply runs sub-commands in order;
// Revert runs them in reverse, so the group is a clean inverse of itself.
type Batch struct {
	label string
	cmds  []Command
}

// NewBatch groups cmds under a single undo label.
func NewBatch(label string, cmds ...Command) *Batch {
	return &Batch{label: label, cmds: cmds}
}

// Label returns the batch's undo text.
func (b *Batch) Label() string { return b.label }

// Commands returns the sub-commands in order.
func (b *Batch) Commands() []Command { return b.cmds }

// Len returns the number of sub-commands.
func (b *Batch) Len() int { return len(b.cmds) }

// Apply runs every sub-command in order. If one fails, the already-applied prefix
// is reverted in reverse so a failed batch leaves no partial mutation.
func (b *Batch) Apply() error {
	for i, c := range b.cmds {
		if err := c.Apply(); err != nil {
			b.revertPrefix(i)
			return err
		}
	}
	return nil
}

// Revert undoes the sub-commands in reverse order.
func (b *Batch) Revert() error {
	return b.revertFrom(len(b.cmds) - 1)
}

// revertPrefix reverts sub-commands [0,n) in reverse, used to roll back a partial
// Apply failure.
func (b *Batch) revertPrefix(n int) {
	_ = b.revertFrom(n - 1)
}

func (b *Batch) revertFrom(last int) error {
	for i := last; i >= 0; i-- {
		if err := b.cmds[i].Revert(); err != nil {
			return err
		}
	}
	return nil
}
