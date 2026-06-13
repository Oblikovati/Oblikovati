// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"encoding/json"
	"sync"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// occRecorder collects the occurrence push events the relay forwards (thread-safe;
// occurrence events may fire from any goroutine).
type occRecorder struct {
	mu  sync.Mutex
	got []wire.OccurrenceEventPayload
}

func (r *occRecorder) sink(b []byte) {
	var p wire.OccurrenceEventPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return
	}
	r.mu.Lock()
	r.got = append(r.got, p)
	r.mu.Unlock()
}

// ofType returns the recorded payloads whose Type matches.
func (r *occRecorder) ofType(eventType string) []wire.OccurrenceEventPayload {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []wire.OccurrenceEventPayload
	for _, p := range r.got {
		if p.Type == eventType {
			out = append(out, p)
		}
	}
	return out
}

func (r *occRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.got)
}

// TestSubscribeAssemblyRelaysLifecycle drives the curated occurrence events through
// the pure relay and checks each reaches the sink as the right wire payload, tagged
// with the document.
func TestSubscribeAssemblyRelaysLifecycle(t *testing.T) {
	asm := compdef.NewAssemblyComponentDefinition()
	var rec occRecorder
	subs := SubscribeAssembly(asm.Events().Bus(), doc.ID(4), rec.sink)
	defer cancelAll(subs)

	part := compdef.NewPartComponentDefinition()
	o := asm.Place("box:1", part, math.Identity4())
	o.SetTransform(math.Translation4(math.V3(5, 0, 0)))
	if err := asm.SetOccurrenceSuppressed(o, true); err != nil {
		t.Fatalf("suppress: %v", err)
	}
	if err := asm.DeleteOccurrence(o); err != nil {
		t.Fatalf("delete: %v", err)
	}

	added := rec.ofType(wire.EventOccurrenceAdded)
	if len(added) != 1 || added[0].Document != 4 || added[0].Occurrence != o.ID() || added[0].Name != "box:1" {
		t.Fatalf("added events = %+v, want one for box:1 on document 4", added)
	}
	moved := rec.ofType(wire.EventOccurrenceTransformed)
	if len(moved) != 1 || moved[0].Transform == nil || moved[0].Previous == nil {
		t.Fatalf("transformed events = %+v, want one carrying both placements", moved)
	}
	if moved[0].Transform.Cells[3] != 5 || moved[0].Previous.Cells[3] != 0 {
		t.Errorf("transform placements new/prior X = %v/%v, want 5/0", moved[0].Transform.Cells[3], moved[0].Previous.Cells[3])
	}
	supp := rec.ofType(wire.EventOccurrenceSuppressed)
	if len(supp) != 1 || !supp[0].Suppressed {
		t.Errorf("suppressed events = %+v, want one with Suppressed=true", supp)
	}
	if del := rec.ofType(wire.EventOccurrenceDeleted); len(del) != 1 {
		t.Errorf("deleted events = %+v, want one", del)
	}
}

// TestSubscribeAssemblyReplaceRelays checks the replace event surfaces (it carries
// identity only — the swapped-in definition is re-queried by the add-in).
func TestSubscribeAssemblyReplaceRelays(t *testing.T) {
	asm := compdef.NewAssemblyComponentDefinition()
	var rec occRecorder
	subs := SubscribeAssembly(asm.Events().Bus(), doc.ID(1), rec.sink)
	defer cancelAll(subs)

	o := asm.Place("pin:1", compdef.NewPartComponentDefinition(), math.Identity4())
	asm.Occurrences().Replace(o, compdef.NewPartComponentDefinition())

	if rep := rec.ofType(wire.EventOccurrenceReplaced); len(rep) != 1 || rep[0].Occurrence != o.ID() {
		t.Errorf("replaced events = %+v, want one for the occurrence", rep)
	}
}

// TestSubscribeAssemblyCoalescesDrag pins the batch contract over the wire: a
// suspended multi-step drag relays exactly one transformed push carrying the net move
// (pre-drag placement as Previous, final as Transform).
func TestSubscribeAssemblyCoalescesDrag(t *testing.T) {
	asm := compdef.NewAssemblyComponentDefinition()
	var rec occRecorder
	subs := SubscribeAssembly(asm.Events().Bus(), doc.ID(2), rec.sink)
	defer cancelAll(subs)

	o := asm.Place("slider:1", compdef.NewPartComponentDefinition(), math.Identity4())

	asm.Occurrences().SuspendNotifications()
	for i := 1; i <= 50; i++ {
		o.SetTransform(math.Translation4(math.V3(float64(i), 0, 0)))
	}
	asm.Occurrences().ResumeNotifications()

	moved := rec.ofType(wire.EventOccurrenceTransformed)
	if len(moved) != 1 {
		t.Fatalf("drag relayed %d transformed pushes, want 1 (coalesced)", len(moved))
	}
	if moved[0].Transform.Cells[3] != 50 || moved[0].Previous.Cells[3] != 0 {
		t.Errorf("coalesced move new/prior X = %v/%v, want 50/0 (net move)", moved[0].Transform.Cells[3], moved[0].Previous.Cells[3])
	}
}

// TestSubscribeAssembliesWiresActivatedDocument checks the workspace watcher wires an
// assembly document's bus on activation (after AddAssembly installs the live content)
// and relays its occurrence events, and stops once the document closes.
func TestSubscribeAssembliesWiresActivatedDocument(t *testing.T) {
	s := app.NewSession()
	var rec occRecorder
	subs := subscribeAssemblies(s.Workspace().Events(), rec.sink)
	defer cancelAll(subs)

	d, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(d); err != nil { // rewires to the live content's bus
		t.Fatalf("activate: %v", err)
	}
	asm := d.Content().(*compdef.AssemblyComponentDefinition)
	asm.Place("box:1", compdef.NewPartComponentDefinition(), math.Identity4())

	added := rec.ofType(wire.EventOccurrenceAdded)
	if len(added) != 1 || added[0].Document != uint64(d.ID()) {
		t.Fatalf("after activate, added events = %+v, want one tagged with document %d", added, d.ID())
	}

	// After close, further mutation must not relay.
	before := rec.count()
	if err := s.Workspace().Close(d, true); err != nil {
		t.Fatalf("close: %v", err)
	}
	asm.Place("box:2", compdef.NewPartComponentDefinition(), math.Identity4())
	if rec.count() != before {
		t.Errorf("relayed %d events after close, want %d (unwired)", rec.count()-before, 0)
	}
}
