// SPDX-License-Identifier: GPL-2.0-only

package ui

import "testing"

// TestQATDispatchMapping covers the id→channel mapping and the callID value
// (a wrong New Part command id fails here — G design, r7).
func TestQATDispatchMapping(t *testing.T) {
	cases := []struct {
		id       string
		callID   string
		setOpen  bool
		setSave  bool
		callUndo bool
		callRedo bool
		disabled bool
	}{
		{"new-part", "GetStarted.NewPart", false, false, false, false, false},
		{"open", "", true, false, false, false, false},
		{"save", "", false, true, false, false, false},
		{"undo", "", false, false, true, false, false},
		{"redo", "", false, false, false, true, false},
		{"unknown", "", false, false, false, false, false},
		{"", "", false, false, false, false, false},
	}
	for _, c := range cases {
		eff := qatDispatch(c.id, false, true, true)
		if eff.callID != c.callID || eff.setOpen != c.setOpen || eff.setSave != c.setSave ||
			eff.callUndo != c.callUndo || eff.callRedo != c.callRedo || eff.disabled != c.disabled {
			t.Errorf("qatDispatch(%q) = %+v, want {callID:%q open:%v save:%v undo:%v redo:%v dis:%v}",
				c.id, eff, c.callID, c.setOpen, c.setSave, c.callUndo, c.callRedo, c.disabled)
		}
	}
}

// TestQATDispatchGuard covers the #1750 in-transaction guard: undo/redo clicks
// are no-ops while a transaction is open, regardless of stack state.
func TestQATDispatchGuard(t *testing.T) {
	// inTxn=true → callUndo/callRedo false, even when the stacks are non-empty.
	eff := qatDispatch("undo", true, true, true)
	if eff.callUndo || eff.disabled {
		t.Errorf("in-transaction undo should be enabled-but-noop, got %+v", eff)
	}
	eff = qatDispatch("redo", true, true, true)
	if eff.callRedo || eff.disabled {
		t.Errorf("in-transaction redo should be enabled-but-noop, got %+v", eff)
	}
	// Outside a transaction, non-empty stacks → real dispatch.
	eff = qatDispatch("undo", false, true, true)
	if !eff.callUndo || eff.disabled {
		t.Errorf("idle undo should dispatch, got %+v", eff)
	}
	eff = qatDispatch("redo", false, true, true)
	if !eff.callRedo || eff.disabled {
		t.Errorf("idle redo should dispatch, got %+v", eff)
	}
}

// TestQATDispatchDisabled covers the CanUndo/CanRedo grey-out (Edit-menu
// parity) and the invariant that Save is NEVER disabled.
func TestQATDispatchDisabled(t *testing.T) {
	if eff := qatDispatch("undo", false, false, false); !eff.disabled {
		t.Error("empty undo stack must grey the Undo button")
	}
	if eff := qatDispatch("redo", false, false, false); !eff.disabled {
		t.Error("empty redo stack must grey the Redo button")
	}
	if eff := qatDispatch("save", false, false, false); eff.disabled {
		t.Error("Save must never be disabled (File-menu parity)")
	}
	if eff := qatDispatch("save", true, false, false); eff.disabled {
		t.Error("Save must never be disabled, even mid-transaction")
	}
	// A disabled button must never carry a dispatch effect.
	for _, id := range []string{"undo", "redo"} {
		eff := qatDispatch(id, false, false, false)
		if eff.disabled && (eff.callUndo || eff.callRedo) {
			t.Errorf("%s: disabled button must not dispatch, got %+v", id, eff)
		}
	}
}

// TestQATButtonsFixedSet guards the default QAT contents (D2).
func TestQATButtonsFixedSet(t *testing.T) {
	if len(qatButtons) != 5 {
		t.Fatalf("QAT must have exactly 5 default buttons, got %d", len(qatButtons))
	}
	for i, b := range qatButtons {
		if b.id == "" || b.icon == "" || b.label == "" {
			t.Errorf("button %d has an empty field: %+v", i, b)
		}
		if qatActionForID(b.id) == qatNone {
			t.Errorf("button %q has no dispatch action", b.id)
		}
	}
}
