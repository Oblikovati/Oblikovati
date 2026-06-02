// SPDX-License-Identifier: GPL-2.0-only

package event

import "context"

// TypeID is an event type's stable, language-independent tag, used when an event
// crosses the gRPC seam to add-ins (ADR-0003). In-process dispatch keys on the Go
// type, not this id; the id is the wire identity.
type TypeID uint32

// Event is any value the bus can carry. Concrete events are plain structs whose
// fields are the typed payload (replacing COM's NameValueMap context).
type Event interface {
	// EventID returns the stable type tag (never renumber across versions).
	EventID() TypeID
}

// Phase is when a handler runs relative to the operation — the EventTimingEnum.
type Phase uint8

const (
	// Before runs prior to the operation and may veto it.
	Before Phase = iota
	// After runs once the operation has committed; its result is final.
	After
)

// HandlingCode is a handler's disposition — the HandlingCodeEnum.
type HandlingCode uint8

const (
	// NotHandled: the handler did not consume the event; processing continues.
	NotHandled HandlingCode = iota
	// Handled: the handler dealt with the event (informational; not a veto).
	Handled
	// Abort: a Before handler vetoes the operation, which is cancelled.
	Abort
)

// Outcome is what a handler returns and what [Emit] aggregates. Use the
// constructors rather than building it directly.
type Outcome struct {
	Code   HandlingCode
	Reason string // why the operation was vetoed, when Code == Abort
}

// Continue reports that the handler did not consume or block the event.
func Continue() Outcome { return Outcome{Code: NotHandled} }

// Handle reports that the handler dealt with the event without vetoing it.
func Handle() Outcome { return Outcome{Code: Handled} }

// Veto cancels the operation, carrying a human-readable reason for the UI.
func Veto(reason string) Outcome { return Outcome{Code: Abort, Reason: reason} }

// Vetoed reports whether the outcome cancels the operation.
func (o Outcome) Vetoed() bool { return o.Code == Abort }

// Context is passed to every handler: the phase it is running in and a
// context.Context for cancellation/deadlines (the latter bounds how long an
// out-of-process add-in may take to answer a veto).
type Context struct {
	Ctx   context.Context
	Phase Phase
}

// Handler reacts to events of one type E. Returning a vetoing [Outcome] from a
// Before handler cancels the operation.
type Handler[E Event] func(Context, E) Outcome
