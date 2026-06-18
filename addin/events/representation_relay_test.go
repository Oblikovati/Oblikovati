// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/event"
)

// TestForwardsRepresentationEvents: the representation / model-state change events emitted by the
// representation router handlers are relayed to the add-in sink (#901) — closing the gap the
// parity guard found (the methods were handled, only the change notifications were missing).
func TestForwardsRepresentationEvents(t *testing.T) {
	s := app.NewSession()
	var rec recorder
	if subs := Subscribe(s, rec.sink); len(subs) == 0 {
		t.Fatal("Subscribe returned no subscriptions")
	}

	event.Emit(s.Events(), event.After, app.RepresentationActivated{Kind: "design", Name: "Master"})
	event.Emit(s.Events(), event.After, app.RepresentationCaptured{Kind: "lod", Name: "Lightweight"})
	event.Emit(s.Events(), event.After, app.ModelStateActivated{Name: "Primary"})

	got := rec.types()
	for _, want := range []string{"representations.activated", "representations.captured", "modelStates.activated"} {
		if !has(got, want) {
			t.Errorf("missing %s in relayed events %v", want, got)
		}
	}
}
