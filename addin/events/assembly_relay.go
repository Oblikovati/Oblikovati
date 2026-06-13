// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"sync"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/event"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/occurrence"
)

// Assembly occurrence events fire on a bus owned by each assembly's component
// definition (compdef.AssemblyEvents), not on the session or workspace bus — so the
// relay cannot subscribe once at startup. subscribeAssemblies watches the workspace
// for assembly documents and wires each one's bus to the sink, rewiring if a
// document's content is swapped and unwiring when it closes (M11-F07, #723).

// SubscribeAssembly wires one assembly definition's occurrence bus to sink, tagging
// every relayed event with document so an add-in can attribute it. It returns the
// five subscriptions (add/delete/replace/transform/suppress) so the caller can cancel
// them. This is the pure relay — the host-facing wiring is [subscribeAssemblies].
func SubscribeAssembly(bus *event.Bus, document doc.ID, sink Sink) []event.Subscription {
	return []event.Subscription{
		event.Subscribe(bus, event.After, func(_ event.Context, e compdef.OccurrenceAdd) event.Outcome {
			return relayJSON(sink, occurrencePayload(wire.EventOccurrenceAdded, document, e.Occurrence))
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e compdef.OccurrenceDelete) event.Outcome {
			return relayJSON(sink, occurrencePayload(wire.EventOccurrenceDeleted, document, e.Occurrence))
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e compdef.OccurrenceReplace) event.Outcome {
			return relayJSON(sink, occurrencePayload(wire.EventOccurrenceReplaced, document, e.Occurrence))
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e compdef.OccurrenceTransform) event.Outcome {
			return relayJSON(sink, transformPayload(document, e.Occurrence, e.Previous))
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e compdef.OccurrenceSuppress) event.Outcome {
			p := occurrencePayload(wire.EventOccurrenceSuppressed, document, e.Occurrence)
			p.Suppressed = e.Suppressed
			return relayJSON(sink, p)
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e compdef.AssemblyFeaturesRecomputed) event.Outcome {
			return relayJSON(sink, assemblyFeaturesPayload(document, e))
		}),
	}
}

// assemblyFeaturesPayload renders a feature-program recompute as the wire change event,
// mapping each feature's model health snapshot to its wire shape.
func assemblyFeaturesPayload(document doc.ID, e compdef.AssemblyFeaturesRecomputed) wire.AssemblyFeaturesChangedEvent {
	feats := make([]wire.AssemblyFeatureHealth, len(e.Features))
	for i, f := range e.Features {
		feats[i] = wire.AssemblyFeatureHealth{ID: f.ID, Suppressed: f.Suppressed, Health: f.Health}
	}
	return wire.AssemblyFeaturesChangedEvent{
		Type:     wire.EventAssemblyFeaturesChanged,
		Document: uint64(document),
		Features: feats,
	}
}

// occurrencePayload renders an occurrence's identity (document, session id, instance
// name) into the shared wire payload for the given event type.
func occurrencePayload(eventType string, document doc.ID, o *occurrence.Occurrence) wire.OccurrenceEventPayload {
	return wire.OccurrenceEventPayload{
		Type:       eventType,
		Document:   uint64(document),
		Occurrence: o.ID(),
		Name:       o.Name(),
	}
}

// transformPayload renders an occurrence move, carrying its new placement and the
// prior one (for a coalesced drag, previous is the pre-drag placement).
func transformPayload(document doc.ID, o *occurrence.Occurrence, previous math.Matrix4) wire.OccurrenceEventPayload {
	p := occurrencePayload(wire.EventOccurrenceTransformed, document, o)
	cur := wireMatrix(o.Transform())
	prev := wireMatrix(previous)
	p.Transform, p.Previous = &cur, &prev
	return p
}

// wireMatrix converts a model transform to the contract's matrix value type.
func wireMatrix(m math.Matrix4) types.Matrix { return types.Matrix{Cells: m.Cells()} }

// subscribeAssemblies watches the workspace document bus and keeps each open
// assembly document's occurrence bus wired to sink. It wires on create and activate
// (reading the live content, since AddAssembly installs the real definition after the
// create event) and unwires on close, deduping by bus identity so a content swap
// rewires exactly once.
func subscribeAssemblies(ws *event.Bus, sink Sink) []event.Subscription {
	w := &assemblyWatcher{sink: sink, wired: map[doc.ID]wiredAssembly{}}
	return []event.Subscription{
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentCreated) event.Outcome {
			w.wire(e.Document)
			return event.Continue()
		}),
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentActivate) event.Outcome {
			w.wire(e.Document)
			return event.Continue()
		}),
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentClose) event.Outcome {
			w.unwire(e.Document.ID())
			return event.Continue()
		}),
	}
}

// wiredAssembly records which bus a document is wired to and the relay subscriptions,
// so a content swap (different bus) can be detected and the old subscriptions cancelled.
type wiredAssembly struct {
	bus  *event.Bus
	subs []event.Subscription
}

// assemblyWatcher tracks the per-document occurrence-bus relays. Its map is guarded
// because document lifecycle events and occurrence events may fire from different
// goroutines.
type assemblyWatcher struct {
	sink  Sink
	mu    sync.Mutex
	wired map[doc.ID]wiredAssembly
}

// wire subscribes d's assembly occurrence bus to the sink, if d holds assembly content
// not already wired to that same bus. A document whose content was swapped to a new
// assembly definition (a different bus) has its stale relay cancelled and replaced.
func (w *assemblyWatcher) wire(d *doc.Document) {
	asm, ok := d.Content().(*compdef.AssemblyComponentDefinition)
	if !ok {
		return
	}
	bus := asm.Events().Bus()
	w.mu.Lock()
	defer w.mu.Unlock()
	if cur, ok := w.wired[d.ID()]; ok {
		if cur.bus == bus {
			return
		}
		cancelAll(cur.subs)
	}
	w.wired[d.ID()] = wiredAssembly{bus: bus, subs: SubscribeAssembly(bus, d.ID(), w.sink)}
}

// unwire cancels and forgets a document's occurrence relay (on close).
func (w *assemblyWatcher) unwire(id doc.ID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if cur, ok := w.wired[id]; ok {
		cancelAll(cur.subs)
		delete(w.wired, id)
	}
}

// cancelAll cancels every subscription in subs.
func cancelAll(subs []event.Subscription) {
	for _, s := range subs {
		s.Cancel()
	}
}
