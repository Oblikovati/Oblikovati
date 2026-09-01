// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
)

// TestDrawingEditsReplicate proves a drawing edit over the wire emits edit.committed for
// collaboration replication (ADR-0004). The whole drawing authoring surface (views/annotations/
// dimensions/sketches/sheets) was registered read-only before #1426, so wire-driven drawing edits were
// silently not replicated to collaborators. A read-only drawing query still emits nothing.
//
// Drawing UNDO is covered separately by TestCentralSeamRecordsDrawingViewUndo: since #1448 gave
// DrawingContent recipe-snapshot support, the same mutating classification now also records an undo
// step, so a wire drawing edit both replicates and is undoable.
func TestDrawingEditsReplicate(t *testing.T) {
	t.Parallel()
	r, s := drawingViewSession(t) // a part with geometry + a drawing whose model reference is set

	var got []app.EditCommitted
	sub := event.Subscribe(s.Events(), event.After, func(_ event.Context, e app.EditCommitted) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})
	defer sub.Cancel()

	call(t, r, s, "drawingViews.list", `{}`, nil)
	if len(got) != 0 {
		t.Fatalf("read-only drawingViews.list emitted edit.committed: %+v", got)
	}

	call(t, r, s, "drawingViews.addBase", `{"name":"FRONT","orientation":"front","scale":2,"centerXmm":120,"centerYmm":100}`, nil)
	if len(got) != 1 {
		t.Fatalf("drawingViews.addBase emitted %d edit.committed events, want 1", len(got))
	}
	if got[0].Method != wire.MethodDrawingViewsAddBase {
		t.Errorf("replicated method = %q, want %q", got[0].Method, wire.MethodDrawingViewsAddBase)
	}
}
