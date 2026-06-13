// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"errors"
	"testing"

	"oblikovati.org/event"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// placeUnit places a unit-box part in asm and returns the occurrence.
func placeUnit(asm *AssemblyComponentDefinition, name string) *occurrence.Occurrence {
	part := NewPartComponentDefinition()
	return asm.Place(name, part, math.Identity4())
}

// TestEventsRaiseOnPlaceMoveReplace covers the After-phase occurrence events reaching a
// bus subscriber for the three programmatic mutations that have no veto (M11-F07).
func TestEventsRaiseOnPlaceMoveReplace(t *testing.T) {
	asm := NewAssemblyComponentDefinition()
	var adds, transforms, replaces int
	var lastPrev math.Matrix4
	event.Subscribe(asm.Events().Bus(), event.After, func(_ event.Context, e OccurrenceAdd) event.Outcome {
		adds++
		return event.Continue()
	})
	event.Subscribe(asm.Events().Bus(), event.After, func(_ event.Context, e OccurrenceTransform) event.Outcome {
		transforms++
		lastPrev = e.Previous
		return event.Continue()
	})
	event.Subscribe(asm.Events().Bus(), event.After, func(_ event.Context, e OccurrenceReplace) event.Outcome {
		replaces++
		return event.Continue()
	})

	o := placeUnit(asm, "a:1")
	o.SetTransform(math.Translation4(math.V3(4, 0, 0)))
	asm.Occurrences().Replace(o, NewPartComponentDefinition())

	if adds != 1 || transforms != 1 || replaces != 1 {
		t.Fatalf("event counts: add=%d transform=%d replace=%d, want 1/1/1", adds, transforms, replaces)
	}
	if lastPrev != math.Identity4() {
		t.Errorf("transform Previous = %v, want the prior identity placement", lastPrev)
	}
}

// TestSuppressVetoKeepsOccurrenceActive: a Before handler vetoes the suppress, so the
// definition op returns an AssemblyVetoError, the occurrence stays active, and no After
// event fires.
func TestSuppressVetoKeepsOccurrenceActive(t *testing.T) {
	asm := NewAssemblyComponentDefinition()
	o := placeUnit(asm, "a:1")

	event.Subscribe(asm.Events().Bus(), event.Before, func(_ event.Context, e OccurrenceSuppress) event.Outcome {
		return event.Veto("component is required")
	})
	afterFired := false
	event.Subscribe(asm.Events().Bus(), event.After, func(_ event.Context, e OccurrenceSuppress) event.Outcome {
		afterFired = true
		return event.Continue()
	})

	err := asm.SetOccurrenceSuppressed(o, true)
	var veto *AssemblyVetoError
	if !errors.As(err, &veto) {
		t.Fatalf("SetOccurrenceSuppressed err = %v, want AssemblyVetoError", err)
	}
	if o.Suppressed() {
		t.Error("occurrence was suppressed despite the veto")
	}
	if afterFired {
		t.Error("After event fired despite the Before veto")
	}
}

// TestSuppressCommitsWhenNotVetoed: with no veto the suppress applies and raises the
// After event carrying the new state.
func TestSuppressCommitsWhenNotVetoed(t *testing.T) {
	asm := NewAssemblyComponentDefinition()
	o := placeUnit(asm, "a:1")
	var got *OccurrenceSuppress
	event.Subscribe(asm.Events().Bus(), event.After, func(_ event.Context, e OccurrenceSuppress) event.Outcome {
		got = &e
		return event.Continue()
	})

	if err := asm.SetOccurrenceSuppressed(o, true); err != nil {
		t.Fatalf("SetOccurrenceSuppressed: %v", err)
	}
	if !o.Suppressed() {
		t.Error("occurrence not suppressed after a non-vetoed call")
	}
	if got == nil || !got.Suppressed || got.Occurrence != o {
		t.Errorf("After event = %+v, want one carrying Suppressed=true for the occurrence", got)
	}
}

// TestDeleteVetoKeepsOccurrence: a Before handler vetoes the delete, so the occurrence
// stays in the assembly.
func TestDeleteVetoKeepsOccurrence(t *testing.T) {
	asm := NewAssemblyComponentDefinition()
	o := placeUnit(asm, "a:1")
	event.Subscribe(asm.Events().Bus(), event.Before, func(_ event.Context, e OccurrenceDelete) event.Outcome {
		return event.Veto("locked")
	})

	if err := asm.DeleteOccurrence(o); err == nil {
		t.Fatal("DeleteOccurrence returned nil, want a veto error")
	}
	if asm.Occurrences().Count() != 1 {
		t.Errorf("occurrence count = %d after vetoed delete, want 1", asm.Occurrences().Count())
	}
}

// TestDeleteCommitsAndRaisesAfter: an unvetoed delete removes the occurrence and raises
// the After event.
func TestDeleteCommitsAndRaisesAfter(t *testing.T) {
	asm := NewAssemblyComponentDefinition()
	o := placeUnit(asm, "a:1")
	deleted := false
	event.Subscribe(asm.Events().Bus(), event.After, func(_ event.Context, e OccurrenceDelete) event.Outcome {
		deleted = e.Occurrence == o
		return event.Continue()
	})

	if err := asm.DeleteOccurrence(o); err != nil {
		t.Fatalf("DeleteOccurrence: %v", err)
	}
	if asm.Occurrences().Count() != 0 {
		t.Errorf("occurrence count = %d after delete, want 0", asm.Occurrences().Count())
	}
	if !deleted {
		t.Error("OccurrenceDelete After event did not fire for the deleted occurrence")
	}
}

// TestDragBatchRaisesOneTransformEvent ties the occurrence-layer batch to the event
// source: a suspended multi-step drag raises exactly one OccurrenceTransform on the bus.
func TestDragBatchRaisesOneTransformEvent(t *testing.T) {
	asm := NewAssemblyComponentDefinition()
	o := placeUnit(asm, "a:1")
	var transforms int
	event.Subscribe(asm.Events().Bus(), event.After, func(_ event.Context, e OccurrenceTransform) event.Outcome {
		transforms++
		return event.Continue()
	})

	asm.Occurrences().SuspendNotifications()
	for i := 1; i <= 50; i++ {
		o.SetTransform(math.Translation4(math.V3(float64(i), 0, 0)))
	}
	asm.Occurrences().ResumeNotifications()

	if transforms != 1 {
		t.Errorf("drag raised %d OccurrenceTransform events, want 1 (batched)", transforms)
	}
}
