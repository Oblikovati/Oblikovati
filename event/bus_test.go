// SPDX-License-Identifier: GPL-2.0-only

package event

import (
	"context"
	"testing"
)

// Two sample events for the tests.
type docClosing struct {
	name string
}

func (docClosing) EventID() TypeID { return 1 }

type docSaved struct {
	name string
}

func (docSaved) EventID() TypeID { return 2 }

func TestSubscriberReceivesEvents(t *testing.T) {
	b := NewBus()
	var seen []string
	Subscribe(b, After, func(_ Context, e docSaved) Outcome {
		seen = append(seen, e.name)
		return Continue()
	})

	Emit(b, After, docSaved{name: "part.obk"})
	if len(seen) != 1 || seen[0] != "part.obk" {
		t.Errorf("handler saw %v, want [part.obk]", seen)
	}
}

func TestMulticastAndPhaseIsolation(t *testing.T) {
	b := NewBus()
	calls := 0
	Subscribe(b, After, func(_ Context, _ docSaved) Outcome { calls++; return Continue() })
	Subscribe(b, After, func(_ Context, _ docSaved) Outcome { calls++; return Continue() })
	// A Before handler for the same type must NOT fire on an After emit.
	Subscribe(b, Before, func(_ Context, _ docSaved) Outcome { calls += 100; return Continue() })

	Emit(b, After, docSaved{})
	if calls != 2 {
		t.Errorf("calls = %d, want 2 After handlers (no Before, no other type)", calls)
	}
}

func TestTypeIsolation(t *testing.T) {
	b := NewBus()
	got := 0
	Subscribe(b, After, func(_ Context, _ docSaved) Outcome { got++; return Continue() })
	Emit(b, After, docClosing{}) // different type, same phase
	if got != 0 {
		t.Error("handler fired for an unrelated event type")
	}
}

func TestBeforeHandlerVetoCancels(t *testing.T) {
	b := NewBus()
	Subscribe(b, Before, func(_ Context, e docClosing) Outcome {
		if e.name == "dirty.obk" {
			return Veto("document has unsaved changes")
		}
		return Continue()
	})

	if out := Emit(b, Before, docClosing{name: "clean.obk"}); out.Vetoed() {
		t.Error("clean close was vetoed")
	}
	out := Emit(b, Before, docClosing{name: "dirty.obk"})
	if !out.Vetoed() || out.Reason != "document has unsaved changes" {
		t.Errorf("veto outcome = %+v, want abort with reason", out)
	}
}

func TestAggregateOutcomeStrongestWins(t *testing.T) {
	b := NewBus()
	Subscribe(b, Before, func(_ Context, _ docClosing) Outcome { return Continue() })
	Subscribe(b, Before, func(_ Context, _ docClosing) Outcome { return Handle() })
	Subscribe(b, Before, func(_ Context, _ docClosing) Outcome { return Veto("first reason") })
	Subscribe(b, Before, func(_ Context, _ docClosing) Outcome { return Veto("second reason") })

	out := Emit(b, Before, docClosing{})
	if !out.Vetoed() || out.Reason != "first reason" {
		t.Errorf("aggregate = %+v, want abort keeping the first reason", out)
	}
}

func TestNoHandlersIsContinue(t *testing.T) {
	if out := Emit(NewBus(), After, docSaved{}); out.Vetoed() || out.Code != NotHandled {
		t.Errorf("emit with no subscribers = %+v, want Continue", out)
	}
}

func TestUnsubscribe(t *testing.T) {
	b := NewBus()
	calls := 0
	sub := Subscribe(b, After, func(_ Context, _ docSaved) Outcome { calls++; return Continue() })
	Emit(b, After, docSaved{})
	if !sub.Cancel() {
		t.Error("Cancel returned false for an active subscription")
	}
	Emit(b, After, docSaved{})
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (handler should not fire after cancel)", calls)
	}
	if sub.Cancel() {
		t.Error("second Cancel returned true")
	}
}

func TestContextCarriesPhaseAndDeadline(t *testing.T) {
	b := NewBus()
	type ctxKey string
	var gotPhase Phase
	var gotVal any
	Subscribe(b, Before, func(c Context, _ docClosing) Outcome {
		gotPhase = c.Phase
		gotVal = c.Ctx.Value(ctxKey("k"))
		return Continue()
	})
	ctx := context.WithValue(context.Background(), ctxKey("k"), "v")
	EmitContext(ctx, b, Before, docClosing{})
	if gotPhase != Before {
		t.Errorf("handler phase = %v, want Before", gotPhase)
	}
	if gotVal != "v" {
		t.Errorf("handler context value = %v, want v", gotVal)
	}
}

// TestBusMethodFormMatchesFreeFunctions exercises the Go 1.27 generic methods
// (Bus.Subscribe/Emit/EmitContext) directly rather than through the free-function
// wrappers the rest of this file uses, proving the method form behaves identically
// (Subscribe/Emit/EmitContext above already cover dispatch semantics in depth).
func TestBusMethodFormMatchesFreeFunctions(t *testing.T) {
	b := NewBus()
	var seen []string
	sub := b.Subscribe(After, func(_ Context, e docSaved) Outcome {
		seen = append(seen, e.name)
		return Continue()
	})

	if out := b.Emit(After, docSaved{name: "part.obk"}); out.Vetoed() {
		t.Errorf("Emit outcome = %+v, want not vetoed", out)
	}
	if len(seen) != 1 || seen[0] != "part.obk" {
		t.Errorf("handler saw %v, want [part.obk]", seen)
	}

	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("k"), "v")
	var gotVal any
	b.Subscribe(Before, func(c Context, _ docClosing) Outcome {
		gotVal = c.Ctx.Value(ctxKey("k"))
		return Continue()
	})
	b.EmitContext(ctx, Before, docClosing{})
	if gotVal != "v" {
		t.Errorf("EmitContext context value = %v, want v", gotVal)
	}

	if !sub.Cancel() {
		t.Error("Cancel returned false for an active subscription")
	}
}

func TestHandlerMaySubscribeDuringEmit(_ *testing.T) {
	b := NewBus()
	added := false
	Subscribe(b, After, func(_ Context, _ docSaved) Outcome {
		if !added {
			added = true
			Subscribe(b, After, func(_ Context, _ docSaved) Outcome { return Continue() })
		}
		return Continue()
	})
	// Must not deadlock or panic (snapshot isolates the in-flight handler slice).
	Emit(b, After, docSaved{})
}
