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
	"strconv"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
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
	Name     string `json:"name,omitempty"` // style name (M16-F02 style events)
}

// styleEventType maps a style-change kind to its wire event constant (references all three so
// the API↔router parity guard sees each one relayed).
func styleEventType(k app.StyleChangeKind) string {
	switch k {
	case app.StyleAdded:
		return wire.EventStyleAdded
	case app.StyleDeleted:
		return wire.EventStyleDeleted
	default:
		return wire.EventStyleChanged
	}
}

// Subscribe wires the session's document, command, and UI-surface events to sink
// and returns the subscriptions so the caller can cancel them on shutdown.
func Subscribe(s *app.Session, sink Sink) []event.Subscription {
	bus := s.Events()
	subs := subscribeDocuments(s.Workspace().Events(), sink)
	subs = append(subs,
		event.Subscribe(bus, event.Before, func(_ event.Context, e app.CommandStarted) event.Outcome {
			return relay(sink, wireEvent{Type: wire.EventCommandStarted, Command: e.ID})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.CommandEnded) event.Outcome {
			return relay(sink, wireEvent{Type: "command.ended", Command: e.ID, Failed: e.Failed})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.SelectionChanged) event.Outcome {
			return relayJSON(sink, wire.SelectionChangedEvent{Type: wire.EventSelectionChanged, Count: e.Count})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.EnvironmentChanged) event.Outcome {
			return relayJSON(sink, wire.EnvironmentChangedEvent{Type: wire.EventEnvironmentChanged, Environment: e.Environment})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.EditCommitted) event.Outcome {
			return relayEdit(sink, e)
		}),
		// M16-F03 (#404): the active view's camera moved (named-view restore / orientation
		// jump). The add-in reads the new frame via get_camera; wire.CameraChangedEvent is the
		// richer DTO for an out-of-band transport.
		event.Subscribe(bus, event.After, func(_ event.Context, e app.CameraChanged) event.Outcome {
			return relay(sink, wireEvent{Type: wire.EventCameraChanged, Document: strconv.FormatUint(uint64(e.Document), 10)})
		}),
		// M16-F02 (#403/#408): a color/lighting style was added, edited, or deleted.
		event.Subscribe(bus, event.After, func(_ event.Context, e app.StyleChanged) event.Outcome {
			return relay(sink, wireEvent{Type: styleEventType(e.Kind), Name: e.Name})
		}),
	)
	subs = append(subs, subscribeTransactions(bus, sink)...)
	subs = append(subs, subscribeFileAccess(s.Workspace().Events(), sink)...)
	subs = append(subs, subscribeAssemblies(s.Workspace().Events(), sink)...)
	subs = append(subs, subscribeFileUIHooks(bus, sink)...)
	return append(subs, subscribeUISurfaces(bus, sink)...)
}

// subscribeDocuments wires the workspace document events; a flavored document's
// lifecycle additionally services its owner via client.operation (M05-F15).
func subscribeDocuments(ws *event.Bus, sink Sink) []event.Subscription {
	return []event.Subscription{
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentCreated) event.Outcome {
			relayClientOperation(sink, e.Document, "created")
			return relay(sink, wireEvent{Type: "document.created", Document: e.Document.DisplayName(), ID: uint64(e.Document.ID())})
		}),
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentSave) event.Outcome {
			relayClientOperation(sink, e.Document, "saved")
			return relay(sink, wireEvent{Type: "document.saved", Document: e.Document.DisplayName(), ID: uint64(e.Document.ID())})
		}),
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentActivate) event.Outcome {
			relayClientOperation(sink, e.Document, "activated")
			return relay(sink, wireEvent{Type: "document.activated", Document: e.Document.DisplayName(), ID: uint64(e.Document.ID())})
		}),
	}
}

// subscribeUISurfaces wires the M05 UI-surface events: browser panes and dockable
// windows (F03), progress/balloon/prompt feedback (F09), mini-toolbars (F07).
func subscribeUISurfaces(bus *event.Bus, sink Sink) []event.Subscription {
	subs := []event.Subscription{
		event.Subscribe(bus, event.After, func(_ event.Context, e app.BrowserPaneNodeActivated) event.Outcome {
			return relayJSON(sink, wire.BrowserNodeEvent{
				Type: wire.EventBrowserNode, Pane: e.Pane, Node: e.Node, Gesture: e.Gesture,
			})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.DockableWindowChanged) event.Outcome {
			return relayJSON(sink, wire.DockableWindowChangedEvent{
				Type: wire.EventDockableWindowChanged, ID: e.ID, Visible: e.Visible,
			})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.ProgressCancelled) event.Outcome {
			return relayJSON(sink, wire.ProgressCancelledEvent{Type: wire.EventProgressCancelled, ID: e.ID})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.BalloonTipClicked) event.Outcome {
			return relayJSON(sink, wire.BalloonTipClickedEvent{Type: wire.EventBalloonTipClicked, ID: e.ID})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.PromptAnswered) event.Outcome {
			return relayJSON(sink, wire.PromptAnsweredEvent{
				Type: wire.EventPromptAnswered, ID: e.ID, Answer: e.Answer, Remembered: e.Remembered,
			})
		}),
	}
	return append(subs, subscribeMiniToolbars(bus, sink)...)
}

// subscribeMiniToolbars wires the mini-toolbar and dialog events (M05-F07/F08).
func subscribeMiniToolbars(bus *event.Bus, sink Sink) []event.Subscription {
	subs := []event.Subscription{
		event.Subscribe(bus, event.After, func(_ event.Context, e app.MiniToolbarControlChanged) event.Outcome {
			return relayJSON(sink, wire.MiniToolbarChangedEvent{
				Type: wire.EventMiniToolbarChanged, Toolbar: e.Toolbar, Control: e.Control,
				Value: e.Value, Checked: e.Checked, Number: e.Number, Selected: e.Selected,
			})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.MiniToolbarCommitted) event.Outcome {
			return relayJSON(sink, wire.MiniToolbarCommittedEvent{
				Type: wire.EventMiniToolbarCommitted, Toolbar: e.Toolbar, Gesture: e.Gesture,
			})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.FileDialogChosen) event.Outcome {
			return relayJSON(sink, wire.FileDialogChosenEvent{
				Type: wire.EventFileDialogChosen, ID: e.ID, Paths: e.Paths, Cancelled: e.Cancelled,
			})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.WebDialogChanged) event.Outcome {
			return relayJSON(sink, wire.WebDialogChangedEvent{
				Type: wire.EventWebDialogChanged, ID: e.ID, Visible: e.Visible,
			})
		}),
	}
	return append(subs, subscribeGizmos(bus, sink)...)
}

// subscribeGizmos wires the triad/manipulator events (M05-F13).
func subscribeGizmos(bus *event.Bus, sink Sink) []event.Subscription {
	return []event.Subscription{
		event.Subscribe(bus, event.After, func(_ event.Context, e app.TriadDragged) event.Outcome {
			return relayJSON(sink, wire.TriadDragEvent{
				Type: wire.EventTriadDrag, Phase: e.Phase, Segment: e.Segment,
				MoveType: e.MoveType, Delta: types.Matrix{Cells: e.Delta}, Context: e.Context,
			})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.TriadSegmentChanged) event.Outcome {
			return relayJSON(sink, wire.TriadSegmentEvent{
				Type: wire.EventTriadSegment, Segment: e.Segment, Hovered: e.Hovered,
			})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.ManipulatorDragged) event.Outcome {
			return relayJSON(sink, wire.ManipulatorDragEvent{
				Type: wire.EventManipulatorDrag, Gizmo: e.Gizmo, Handle: e.Handle,
				Phase: e.Phase, Position: e.Position, Context: e.Context,
			})
		}),
	}
}

// relayJSON serializes a wire event DTO and hands it to sink (passive listener).
// Generic so each call site stays statically typed (the house no-`any` rule).
func relayJSON[E wire.BrowserNodeEvent | wire.DockableWindowChangedEvent |
	wire.ProgressCancelledEvent | wire.BalloonTipClickedEvent | wire.PromptAnsweredEvent |
	wire.MiniToolbarChangedEvent | wire.MiniToolbarCommittedEvent |
	wire.FileDialogChosenEvent | wire.WebDialogChangedEvent |
	wire.SelectionChangedEvent | wire.EnvironmentChangedEvent |
	wire.TriadDragEvent | wire.TriadSegmentEvent | wire.ManipulatorDragEvent |
	wire.ClientOperationEvent | wire.TransactionEventPayload |
	wire.FileResolutionEventPayload | wire.FileDirtyEventPayload |
	wire.FileDialogHookPayload | wire.OccurrenceEventPayload |
	wire.AssemblyFeaturesChangedEvent | wire.ConstraintEventPayload |
	wire.JointEventPayload](sink Sink, ev E) event.Outcome {
	if b, err := json.Marshal(ev); err == nil {
		sink(b)
	}
	return event.Continue()
}

// relayClientOperation services a flavored document's owner: a subtyped document's
// lifecycle additionally emits client.operation (M05-F15, Oblikovati#665).
func relayClientOperation(sink Sink, d *doc.Document, operation string) {
	if d.SubType() == "" {
		return
	}
	relayJSON(sink, wire.ClientOperationEvent{
		Type: wire.EventClientOperation, Document: uint64(d.ID()),
		SubType: string(d.SubType()), Operation: operation,
	})
}

// relayEdit serializes a committed edit as the contract's [wire.EditCommittedEvent] (the
// mutation expressed as the wire request that produced it) and hands it to sink, so a
// collaboration add-in can replay it on remote peers (ADR-0004).
func relayEdit(sink Sink, e app.EditCommitted) event.Outcome {
	ev := wire.EditCommittedEvent{
		Type:     wire.EventEditCommitted,
		Document: uint64(e.Document),
		Method:   e.Method,
		Args:     json.RawMessage(e.Args),
	}
	if b, err := json.Marshal(ev); err == nil {
		sink(b)
	}
	return event.Continue()
}

// relay serializes ev and hands it to sink; it never vetoes (passive listener).
func relay(sink Sink, ev wireEvent) event.Outcome {
	if b, err := json.Marshal(ev); err == nil {
		sink(b)
	}
	return event.Continue()
}
