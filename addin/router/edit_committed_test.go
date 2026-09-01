// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
)

// TestEditCommittedEmittedForMutationsOnly checks the router emits an EditCommitted event
// (carrying the wire method + args) for a document-mutating call and nothing for a
// read-only call — the capture seam for operational replication (ADR-0004).
func TestEditCommittedEmittedForMutationsOnly(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)

	var got []app.EditCommitted
	sub := event.Subscribe(s.Events(), event.After, func(_ event.Context, e app.EditCommitted) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})
	defer sub.Cancel()

	// Read-only call must not emit.
	call(t, r, s, "parameters.list", "{}", nil)
	if len(got) != 0 {
		t.Fatalf("read-only call emitted edit.committed: %+v", got)
	}

	// Mutation emits exactly one event with the originating method + args.
	call(t, r, s, "parameters.set", `{"name":"width","expression":"6 cm"}`, nil)
	if len(got) != 1 {
		t.Fatalf("mutation emitted %d edit.committed events, want 1", len(got))
	}
	if got[0].Method != wire.MethodParametersSet {
		t.Errorf("event method = %q, want %q", got[0].Method, wire.MethodParametersSet)
	}
	var a wire.ParameterSetArgs
	if err := json.Unmarshal(got[0].Args, &a); err != nil {
		t.Fatalf("event args not valid JSON: %v", err)
	}
	if a.Name != "width" || a.Expression != "6 cm" {
		t.Errorf("event args = %+v, want width=6 cm", a)
	}
}
