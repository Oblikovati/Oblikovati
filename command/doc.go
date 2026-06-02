// SPDX-License-Identifier: GPL-2.0-only

// Package command is the undo/redo engine, modeled as the command pattern rather
// than a literal COM TransactionManager (architecture core/06, realtime-3d §11).
// Every model mutation goes through a [Command] that can [Command.Apply] and
// [Command.Revert]; a [History] is the undo/redo store. This keeps the COM
// discipline — "every visible edit is a named, undoable unit" (parametric-cad §8) —
// while being more debuggable and serializing naturally (a command is, in effect,
// the gRPC edit payload).
//
// The COM M04 vocabulary maps on cleanly:
//
//	TransactionManager.StartTransaction/End/Abort  →  History.Begin / Transaction.Commit / Abort
//	Transaction (the named undo unit)              →  a committed [Batch] with a label
//	nested / global transactions                   →  nested [Transaction]s folding into a parent
//	MergeWithPrevious                              →  Transaction.MergeWithPrevious (combine batches)
//	SuppressChangeNotifications                    →  History.SuppressNotifications + coalesced notify
//	UndoTransaction/RedoTransaction                →  History.Undo / Redo
//	SetCheckPoint/GoToCheckPoint                    →  History.SetCheckPoint / GoToCheckPoint
//
// Commands are self-contained (Apply/Revert take no document argument): each
// captures whatever it mutates. That is what lets a [Batch] span more than one
// document — the COM "global transaction" — and keeps the engine decoupled from
// any single document. Recompute/notification is a single coalesced callback
// ([History.OnChange]) fired once per committed step, undo, or redo — the seam the
// async recompute engine (ADR-0007, M07+) plugs into.
package command
