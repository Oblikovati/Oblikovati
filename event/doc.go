// SPDX-License-Identifier: GPL-2.0-only

// Package event is the typed event bus — the uniform eventing mechanism reused
// across the application, replacing COM connection points (XEventsObject +
// XEventsSink split, EventTimingEnum, out HandlingCode, NameValueMap context) with
// idiomatic generics (architecture core/06).
//
// The COM concepts map on directly:
//
//	XEventsObject + XEventsSink_Event split  →  Subscribe[E](bus, phase, handler) — one generic call
//	EventTimingEnum BeforeOrAfter            →  Phase (Before / After)
//	out HandlingCodeEnum veto                →  the handler returns an Outcome (Continue / Veto / Handle)
//	NameValueMap context                     →  the typed event struct E (fields, not a string map)
//	316 per-event delegate types             →  one generic Handler[E]; events are plain structs
//
// A Before handler returning [Veto] cancels the operation: the emitter checks the
// aggregate [Outcome] before proceeding. Events are plain structs implementing
// [Event]; subscription is type-safe — a Handler[DocumentClosing] only ever sees
// DocumentClosing values. The [Context] carries the phase and a context.Context so
// out-of-process add-ins can be given a veto deadline (core/06).
package event
