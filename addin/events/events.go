// SPDX-License-Identifier: GPL-2.0-only

// Package events forwards session/document events to a sink as small JSON objects,
// so the bridge add-in can relay them to a connected LLM (e.g. "a document was
// created", "a command finished"). It is pure Go and headless-testable; the head
// wires the sink to the loaded add-ins' Notify entry point (the C ABI).
//
// Doc events fire on the workspace bus, command events on the session bus, so both
// are subscribed.
package events

import (
	"encoding/json"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/event"
	"github.com/Oblikovati/oblikovati/model/doc"
)

// Sink receives one serialized event. It must be safe to call from event-emitting
// goroutines and must not block (the bridge buffers/relays asynchronously).
type Sink func(eventJSON []byte)

// wireEvent is the JSON shape sent to the sink. Fields are omitempty so each event
// carries only what is relevant (type is always present).
type wireEvent struct {
	Type     string `json:"type"`
	Document string `json:"document,omitempty"`
	ID       uint64 `json:"id,omitempty"`
	Command  string `json:"command,omitempty"`
	Failed   bool   `json:"failed,omitempty"`
}

// Subscribe wires the session's document and command events to sink and returns the
// subscriptions so the caller can cancel them on shutdown.
func Subscribe(s *app.Session, sink Sink) []event.Subscription {
	ws := s.Workspace().Events()
	bus := s.Events()
	return []event.Subscription{
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentCreated) event.Outcome {
			return relay(sink, wireEvent{Type: "document.created", Document: e.Document.DisplayName(), ID: uint64(e.Document.ID())})
		}),
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentSave) event.Outcome {
			return relay(sink, wireEvent{Type: "document.saved", Document: e.Document.DisplayName(), ID: uint64(e.Document.ID())})
		}),
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentActivate) event.Outcome {
			return relay(sink, wireEvent{Type: "document.activated", Document: e.Document.DisplayName(), ID: uint64(e.Document.ID())})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.CommandEnded) event.Outcome {
			return relay(sink, wireEvent{Type: "command.ended", Command: e.ID, Failed: e.Failed})
		}),
	}
}

// relay serializes ev and hands it to sink; it never vetoes (passive listener).
func relay(sink Sink, ev wireEvent) event.Outcome {
	if b, err := json.Marshal(ev); err == nil {
		sink(b)
	}
	return event.Continue()
}
