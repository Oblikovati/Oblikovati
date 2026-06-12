// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// The M04-F05 relays (Oblikovati#613): transaction lifecycle, file-access and
// file-UI hook events forwarded to the add-in sink as their wire payloads. All
// relays are After-phase observers — the vetoable/answerable Before phase is
// in-proc only until the event transport (#148) carries answers back.

// subscribeTransactions forwards the five transaction stream moves, each tagged
// with the cursor point it acted on (commit=current, undo=previous, redo=next).
func subscribeTransactions(bus *event.Bus, sink Sink) []event.Subscription {
	return []event.Subscription{
		event.Subscribe(bus, event.After, func(_ event.Context, e app.TransactionCommitted) event.Outcome {
			return relayTransaction(sink, wire.EventTransactionCommitted, e.Document, e.Label, types.TransactionPointCurrent)
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.TransactionUndone) event.Outcome {
			return relayTransaction(sink, wire.EventTransactionUndone, e.Document, e.Label, types.TransactionPointPrevious)
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.TransactionRedone) event.Outcome {
			return relayTransaction(sink, wire.EventTransactionRedone, e.Document, e.Label, types.TransactionPointNext)
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.TransactionAborted) event.Outcome {
			return relayTransaction(sink, wire.EventTransactionAborted, e.Document, e.Label, types.TransactionPointCurrent)
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.TransactionDeleted) event.Outcome {
			return relayTransaction(sink, wire.EventTransactionDeleted, e.Document, "", types.TransactionPointUnknown)
		}),
	}
}

// relayTransaction renders one stream move into the shared wire payload.
func relayTransaction(sink Sink, eventType string, d doc.ID, label string, p types.TransactionPoint) event.Outcome {
	return relayJSON(sink, wire.TransactionEventPayload{
		Type: eventType, Document: uint64(d), Label: label, Point: p,
	})
}

// subscribeFileAccess forwards the workspace-bus file events: resolution
// outcomes and clean→dirty transitions.
func subscribeFileAccess(ws *event.Bus, sink Sink) []event.Subscription {
	return []event.Subscription{
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.FileResolution) event.Outcome {
			return relayJSON(sink, wire.FileResolutionEventPayload{
				Type: wire.EventFileResolution, RequestedName: e.RequestedName, ResolvedName: e.Resolved(),
			})
		}),
		event.Subscribe(ws, event.After, func(_ event.Context, e doc.FileDirty) event.Outcome {
			return relayJSON(sink, wire.FileDirtyEventPayload{
				Type: wire.EventFileDirty, Document: uint64(e.Document.ID()),
				FullDocumentName: e.Document.FullDocumentName(),
			})
		}),
	}
}

// subscribeFileUIHooks forwards the session-bus file-UI flow observations.
func subscribeFileUIHooks(bus *event.Bus, sink Sink) []event.Subscription {
	subs := []event.Subscription{
		event.Subscribe(bus, event.After, func(_ event.Context, e app.FileNew) event.Outcome {
			return relayJSON(sink, wire.FileDialogHookPayload{Type: wire.EventFileNew, DocumentType: e.DocumentType})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.FileOpenFromMRU) event.Outcome {
			return relayJSON(sink, wire.FileDialogHookPayload{Type: wire.EventFileOpenFromMRU, FileName: e.FullDocumentName})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.PopulateFileMetadata) event.Outcome {
			return relayJSON(sink, wire.FileDialogHookPayload{
				Type: wire.EventFilePopulateMetadata, Metadata: metadataEntries(e.Entries()),
			})
		}),
	}
	return append(subs, subscribeFileDialogHooks(bus, sink)...)
}

// subscribeFileDialogHooks forwards the three dialog-replacement hooks with
// whatever path a handler supplied.
func subscribeFileDialogHooks(bus *event.Bus, sink Sink) []event.Subscription {
	return []event.Subscription{
		event.Subscribe(bus, event.After, func(_ event.Context, e app.FileNewDialog) event.Outcome {
			return relayJSON(sink, wire.FileDialogHookPayload{Type: wire.EventFileNewDialog, TemplateFile: e.Supplied()})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.FileOpenDialog) event.Outcome {
			return relayJSON(sink, wire.FileDialogHookPayload{Type: wire.EventFileOpenDialog, FileName: e.Supplied()})
		}),
		event.Subscribe(bus, event.After, func(_ event.Context, e app.FileSaveAsDialog) event.Outcome {
			return relayJSON(sink, wire.FileDialogHookPayload{
				Type: wire.EventFileSaveAsDialog, FileName: e.Supplied(), SaveCopyAs: e.SaveCopyAs,
			})
		}),
	}
}

// metadataEntries maps the app's collected metadata into the wire shape.
func metadataEntries(values []app.FileMetadataValue) []wire.FileMetadataEntry {
	out := make([]wire.FileMetadataEntry, len(values))
	for i, v := range values {
		out[i] = wire.FileMetadataEntry{Name: v.Name, Value: v.Value}
	}
	return out
}
