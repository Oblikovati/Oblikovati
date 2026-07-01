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
	subs = append(subs, subscribeModelChanges(s.Workspace().Events(), sink)...)
	subs = append(subs, subscribeSessionUI(bus, sink)...)
	subs = append(subs, subscribeRepresentations(bus, sink)...)
	subs = append(subs, subscribeParameters(bus, sink)...)
	subs = append(subs, subscribeModeling(bus, sink)...)
	subs = append(subs, subscribeTransactions(bus, sink)...)
	subs = append(subs, subscribeFileAccess(s.Workspace().Events(), sink)...)
	subs = append(subs, subscribeAssemblies(s.Workspace().Events(), sink)...)
	subs = append(subs, subscribeFileUIHooks(bus, sink)...)
	return append(subs, subscribeUISurfaces(bus, sink)...)
}

// subscribeSessionUI relays the session bus's command, selection, environment, edit, camera and
// style change events (M05/M16) to the add-in sink.
func subscribeSessionUI(bus *event.Bus, sink Sink) []event.Subscription {
	subs := []event.Subscription{
		event.Subscribe(bus, event.Before, func(_ event.Context, e app.CommandStarted) event.Outcome {
			return relay(sink, wireEvent{Type: wire.EventCommandStarted, Command: e.ID})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.CommandEnded) event.Outcome {
			return relay(sink, wireEvent{Type: wire.EventCommandEnded, Command: e.ID, Failed: e.Failed})
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
		// M16-F03 (#404): the active view's camera moved (named-view restore / orientation jump).
		event.Subscribe(bus, event.After, func(_ event.Context, e app.CameraChanged) event.Outcome {
			return relay(sink, wireEvent{Type: wire.EventCameraChanged, Document: strconv.FormatUint(uint64(e.Document), 10)})
		}),
		// M16-F02 (#403/#408): a color/lighting style was added, edited, or deleted.
		event.Subscribe(bus, event.After, func(_ event.Context, e app.StyleChanged) event.Outcome {
			return relay(sink, wireEvent{Type: styleEventType(e.Kind), Name: e.Name})
		}),
	}
	return append(subs, subscribePanelEvents(bus, sink)...)
}

// subscribePanelEvents relays the dockable/task-panel control events — a control's value edit, a
// reference-list change, and a modal task-panel close — to the add-in sink (FEM Phase 0a-host).
func subscribePanelEvents(bus *event.Bus, sink Sink) []event.Subscription {
	return []event.Subscription{
		event.Subscribe(bus, event.After, func(_ event.Context, e app.PanelValueChanged) event.Outcome {
			return relayJSON(sink, wire.PanelValueChangedEvent{
				Type: wire.EventPanelValueChanged, WindowId: e.WindowID, ControlId: e.ControlID, Value: e.Value,
			})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.PanelReferencesChanged) event.Outcome {
			return relayJSON(sink, wire.PanelReferencesChangedEvent{
				Type: wire.EventPanelReferencesChanged, WindowId: e.WindowID,
				ControlId: e.ControlID, Refs: e.Refs, Action: e.Action,
			})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.TaskPanelClosed) event.Outcome {
			return relayJSON(sink, wire.TaskPanelClosedEvent{
				Type: wire.EventTaskPanelClosed, ID: e.ID, Accepted: e.Accepted,
			})
		}),
	}
}

// subscribeParameters relays the granular parameter-change notification (#148) — the parameter's
// new state, beyond the generic edit.committed — to the add-in sink.
func subscribeParameters(bus *event.Bus, sink Sink) []event.Subscription {
	return []event.Subscription{
		event.Subscribe(bus, event.After, func(_ event.Context, e app.ParameterChanged) event.Outcome {
			return relayJSON(sink, wire.ParameterChangedEvent{
				Type:     wire.EventParameterChanged,
				Document: uint64(e.Document),
				Parameter: wire.ParameterInfo{
					Name: e.Name, Kind: e.Kind, Expression: e.Expression, Value: e.Value,
				},
			})
		}),
	}
}

// featureEventType maps a feature-lifecycle op to its wire event constant (references all three so
// the API↔router parity guard sees each one relayed).
func featureEventType(op app.FeatureOp) string {
	switch op {
	case app.FeatureAdded:
		return wire.EventFeatureAdded
	case app.FeatureDeleted:
		return wire.EventFeatureDeleted
	default:
		return wire.EventFeatureEdited
	}
}

// subscribeModeling relays the granular feature-lifecycle and sketch-edit notifications (#148): a
// feature was added/edited/deleted, or a sketch's edit mode was entered/exited — beyond the batched
// model.changed, so an add-in reacts per feature/sketch without diffing the tree.
func subscribeModeling(bus *event.Bus, sink Sink) []event.Subscription {
	return []event.Subscription{
		event.Subscribe(bus, event.After, func(_ event.Context, e app.FeatureLifecycleChanged) event.Outcome {
			return relayJSON(sink, wire.FeatureLifecycleEvent{
				Type: featureEventType(e.Op), Document: uint64(e.Document), Feature: e.Feature, Name: e.Name, Kind: e.Kind,
			})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.SketchEditChanged) event.Outcome {
			return relayJSON(sink, wire.SketchEditEvent{
				Type: sketchEditEventType(e.Entered), Document: uint64(e.Document), Sketch: e.Sketch, Name: e.Name,
			})
		}),
	}
}

// sketchEditEventType maps the entered/exited flag to its wire event constant (references both so
// the parity guard sees each relayed).
func sketchEditEventType(entered bool) string {
	if entered {
		return wire.EventSketchEditEntered
	}
	return wire.EventSketchEditExited
}

// subscribeRepresentations relays the representation / model-state change notifications (#901): the
// request/response methods are handled in addin/router; these forward the matching app events so a
// subscriber learns of an activation/capture (Name carries the representation / model-state name).
func subscribeRepresentations(bus *event.Bus, sink Sink) []event.Subscription {
	return []event.Subscription{
		event.Subscribe(bus, event.After, func(_ event.Context, e app.RepresentationActivated) event.Outcome {
			return relay(sink, wireEvent{Type: wire.EventRepresentationActivated, Name: e.Name})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.RepresentationCaptured) event.Outcome {
			return relay(sink, wireEvent{Type: wire.EventRepresentationCaptured, Name: e.Name})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.ModelStateActivated) event.Outcome {
			return relay(sink, wireEvent{Type: wire.EventModelStateActivated, Name: e.Name})
		}),
	}
}

// subscribeDocuments wires the workspace document events; a flavored document's
// lifecycle additionally services its owner via client.operation (M05-F15).
func subscribeDocuments(ws *event.Bus, sink Sink) []event.Subscription {
	return []event.Subscription{
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentCreated) event.Outcome {
			return relayDocumentLifecycle(sink, e.Document, wire.EventDocumentCreated, "created")
		}),
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentOpened) event.Outcome {
			// DocumentOpened carries only the name (the document object is paged in afterwards).
			return relay(sink, wireEvent{Type: wire.EventDocumentOpened, Document: e.FullDocumentName})
		}),
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentSave) event.Outcome {
			return relayDocumentLifecycle(sink, e.Document, wire.EventDocumentSaved, "saved")
		}),
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentClose) event.Outcome {
			return relayDocumentLifecycle(sink, e.Document, wire.EventDocumentClosed, "closed")
		}),
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.DocumentActivate) event.Outcome {
			return relayDocumentLifecycle(sink, e.Document, wire.EventDocumentActivated, "activated")
		}),
	}
}

// relayDocumentLifecycle forwards one document lifecycle event (and services a flavored
// document's owner via client.operation).
func relayDocumentLifecycle(sink Sink, d *doc.Document, eventType, operation string) event.Outcome {
	relayClientOperation(sink, d, operation)
	return relay(sink, wireEvent{Type: eventType, Document: d.DisplayName(), ID: uint64(d.ID())})
}

// subscribeModelChanges relays the committed batch of model changes on a document (#148): the
// feature/sketch/parameter mutations the engine just applied, so an add-in re-queries it.
func subscribeModelChanges(ws *event.Bus, sink Sink) []event.Subscription {
	return []event.Subscription{
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.ModelChanged) event.Outcome {
			name := ""
			if e.Document != nil {
				name = e.Document.DisplayName()
			}
			return relayJSON(sink, wire.ModelChangedEvent{Type: wire.EventModelChanged, Document: name, Changes: len(e.Changes)})
		}),
	}
}

// subscribeUISurfaces wires the M05 UI-surface events: browser panes and dockable
// windows (F03), progress/balloon/prompt feedback (F09), mini-toolbars (F07).
func subscribeUISurfaces(bus *event.Bus, sink Sink) []event.Subscription {
	subs := []event.Subscription{
		event.Subscribe(bus, event.After, func(_ event.Context, e app.BrowserPaneNodeActivated) event.Outcome {
			return relayJSON(sink, wire.BrowserNodeEvent{
				Type: wire.EventBrowserNode, Pane: e.Pane, Node: e.Node, Gesture: e.Gesture, MenuItem: e.MenuItem,
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
	wire.JointEventPayload | wire.ParameterChangedEvent |
	wire.ModelChangedEvent | wire.PanelValueChangedEvent |
	wire.FeatureLifecycleEvent | wire.SketchEditEvent |
	wire.PanelReferencesChangedEvent | wire.TaskPanelClosedEvent](sink Sink, ev E) event.Outcome {
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
