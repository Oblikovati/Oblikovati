// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
)

// TestRepresentationEditsReplicate proves an assembly-representation edit over the wire now emits
// edit.committed for collaboration replication (ADR-0004). Capturing an LOD rep and creating a model
// state were registered read-only before #1426, so these document edits were silently not replicated;
// a read-only rep query still emits nothing.
func TestRepresentationEditsReplicate(t *testing.T) {
	r, s, _, _ := assemblySessionWithBoxes(t, 0, 5)

	var got []app.EditCommitted
	sub := event.Subscribe(s.Events(), event.After, func(_ event.Context, e app.EditCommitted) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})
	defer sub.Cancel()

	call(t, r, s, "lodReps.list", `{}`, nil)
	if len(got) != 0 {
		t.Fatalf("read-only lodReps.list emitted edit.committed: %+v", got)
	}

	call(t, r, s, "lodReps.capture", mustJSON(t, wire.CaptureRepArgs{Name: "simplified"}), &wire.LODResult{})
	if len(got) != 1 || got[0].Method != wire.MethodLODRepsCapture {
		t.Fatalf("lodReps.capture replication = %+v, want one edit.committed for that method", got)
	}

	call(t, r, s, "modelStates.create", mustJSON(t, wire.CreateModelStateArgs{Name: "state"}), &wire.ModelStateResult{})
	if len(got) != 2 || got[1].Method != wire.MethodModelStatesCreate {
		t.Fatalf("modelStates.create replication = %+v, want a second edit.committed for that method", got)
	}
}
