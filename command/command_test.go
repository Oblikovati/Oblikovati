// SPDX-License-Identifier: GPL-2.0-only

package command

import "testing"

func TestFuncAndBatchAccessors(t *testing.T) {
	a := NewFunc("a", func() error { return nil }, func() error { return nil })
	if a.Label() != "a" {
		t.Errorf("Func.Label = %q", a.Label())
	}
	b := NewBatch("group", a, a)
	if b.Label() != "group" || b.Len() != 2 || len(b.Commands()) != 2 {
		t.Errorf("Batch accessors wrong: label=%q len=%d", b.Label(), b.Len())
	}
}

func TestBatchApplyThenRevertIsInverse(t *testing.T) {
	var log []string
	mk := func(name string) Command {
		return NewFunc(name,
			func() error { log = append(log, "+"+name); return nil },
			func() error { log = append(log, "-"+name); return nil })
	}
	b := NewBatch("g", mk("1"), mk("2"), mk("3"))
	if err := b.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := b.Revert(); err != nil {
		t.Fatalf("Revert: %v", err)
	}
	// Apply forward 1,2,3 then Revert reverse 3,2,1.
	want := []string{"+1", "+2", "+3", "-3", "-2", "-1"}
	for i, w := range want {
		if log[i] != w {
			t.Fatalf("order = %v, want %v", log, want)
		}
	}
}

func TestTransactionLabel(t *testing.T) {
	h := NewHistory()
	tx := h.Begin("My Edit")
	if tx.Label() != "My Edit" {
		t.Errorf("Transaction.Label = %q", tx.Label())
	}
	_ = tx.Commit()
}

func TestClear(t *testing.T) {
	d := newPart(t)
	h := NewHistory()
	_ = h.Do(Rename(d, "x"))
	if err := h.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if h.Len() != 0 {
		t.Errorf("Len after Clear = %d, want 0", h.Len())
	}
	// Clear must refuse while a transaction is open.
	h.Begin("open")
	if err := h.Clear(); err == nil {
		t.Error("Clear succeeded with an open transaction")
	}
}
