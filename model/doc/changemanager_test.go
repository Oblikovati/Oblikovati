// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"testing"

	"github.com/Oblikovati/oblikovati/event"
)

// recordingProcessor captures the change batches it is given.
type recordingProcessor struct {
	name string
	seen []ModelChanged
}

func (p *recordingProcessor) Name() string { return p.name }
func (p *recordingProcessor) ProcessChange(e ModelChanged) error {
	p.seen = append(p.seen, e)
	return nil
}

func TestRegisteredProcessorInvokedOnModelChange(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	cm := NewChangeManager(ws.Events())
	defer cm.Close()
	proc := &recordingProcessor{name: "bom"}
	cm.Register(proc)

	d, _ := ws.Add(Part, "p.obk", true)
	// An edit within a transaction commits and announces its changes.
	err := ws.NotifyModelChanged(
		d,
		ChangeDefinition{Kind: ObjectAdded, ObjectLabel: "Extrude1"},
		ChangeDefinition{Kind: ObjectModified, ObjectLabel: "Sketch1"},
	)
	if err != nil {
		t.Fatalf("NotifyModelChanged: %v", err)
	}
	if len(proc.seen) != 1 {
		t.Fatalf("processor invoked %d times, want 1", len(proc.seen))
	}
	if len(proc.seen[0].Changes) != 2 || proc.seen[0].Changes[0].ObjectLabel != "Extrude1" {
		t.Errorf("processor saw wrong changes: %+v", proc.seen[0].Changes)
	}
	if proc.seen[0].Document != d {
		t.Error("processor saw the wrong document")
	}
}

func TestProcessControlEnableDisable(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	cm := NewChangeManager(ws.Events())
	proc := &recordingProcessor{name: "p"}
	reg := cm.Register(proc)
	d, _ := ws.Add(Part, "p.obk", true)

	reg.SetEnabled(false)
	_ = ws.NotifyModelChanged(d, ChangeDefinition{Kind: ObjectModified})
	if len(proc.seen) != 0 {
		t.Error("disabled processor was invoked")
	}
	if reg.Enabled() {
		t.Error("Enabled() true after SetEnabled(false)")
	}

	reg.SetEnabled(true)
	_ = ws.NotifyModelChanged(d, ChangeDefinition{Kind: ObjectModified})
	if len(proc.seen) != 1 {
		t.Errorf("re-enabled processor invoked %d times, want 1", len(proc.seen))
	}

	if reg.Processor() != proc {
		t.Error("Processor() returned the wrong processor")
	}
	if !reg.Unregister() || reg.Unregister() {
		t.Error("Unregister behavior wrong")
	}
	_ = ws.NotifyModelChanged(d, ChangeDefinition{Kind: ObjectDeleted})
	if len(proc.seen) != 1 {
		t.Error("unregistered processor still invoked")
	}
}

func TestModelChangeCanBeVetoed(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	cm := NewChangeManager(ws.Events())
	proc := &recordingProcessor{name: "p"}
	cm.Register(proc)
	d, _ := ws.Add(Part, "p.obk", true)

	// A Before participant can veto a change; the After dispatch then never runs.
	sub := event.Subscribe(ws.Events(), event.Before, func(_ event.Context, _ ModelChanged) event.Outcome {
		return event.Veto("change rejected")
	})
	if err := ws.NotifyModelChanged(d, ChangeDefinition{Kind: ObjectModified}); err == nil {
		t.Error("model change was not vetoed")
	}
	if len(proc.seen) != 0 {
		t.Error("processor ran despite a vetoed change")
	}
	sub.Cancel()
}
