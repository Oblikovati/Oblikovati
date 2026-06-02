# Core 06 — Transactions (commands) & events

*Modernizes M04 (transactions, undo, events, change manager). Two modernizations:
undo as the command pattern (realtime-3d §11) instead of a COM `TransactionManager`,
and events as a typed bus instead of COM connection points + `out HandlingCode`.*

## Undo/redo as commands (realtime-3d §11)

Every mutation of a document goes **through a command** that can `Apply` and
`Revert`. A history stack is the undo/redo store. This replaces the COM
`TransactionManager`/checkpoints while keeping its core discipline ("every visible
edit is a named, undoable unit" — parametric-cad §8).

```go
package command
type Command interface {
    Label() string                 // the undo-menu text (was Transaction.DisplayName)
    Apply(*doc.Document) error
    Revert(*doc.Document) error
}
type History struct{ done, undone []Command }
func (h *History) Do(d *doc.Document, c Command) error  // Apply + push; clears redo
func (h *History) Undo(d *doc.Document) ; Redo(d *doc.Document)
```

Principles:
- **One command type per logical action** (`AddFeature`, `EditDefinition`,
  `Rename`, `Suppress`, `MoveOccurrence`, `SetParameter`) — not whole-document
  snapshots. Cheaper, and the intent is self-documenting (realtime-3d §11).
- **All edits route through commands** — including those issued by the gRPC `Edit`
  service (ADR-0003) — so history is always complete regardless of who edited.
- **Composite commands** (a `Batch`) group sub-edits into one undo step (replaces
  COM nested/global transactions and `MergeWithPrevious`).
- **Apply marks the dependency DAG dirty**, which enqueues an **async recompute**
  (ADR-0007); the command returns immediately, the rebuild happens off-loop.

### Why not literally port `TransactionManager`?

The COM transaction log recorded low-level inverse operations generically. The
command pattern is the same idea expressed as **typed, intentional objects** —
which is more debuggable, serializes naturally (a command *is* the gRPC edit
payload), and aligns the in-proc and RPC edit paths. Checkpoints become "remember
the history length"; `GoToCheckPoint` is "undo to that length."

## Events as a typed bus

COM events used connection points: an `XEventsObject` + `XEventsSink_Event` split,
delivering `(args…, EventTimingEnum BeforeOrAfter, NameValueMap Context, out
HandlingCodeEnum)`. We replace all of it with a typed, generic event bus.

```go
package event
type Phase uint8; const ( Before Phase = iota; After )

type Event interface{ ID() TypeID }                  // e.g. DocumentClosing, FeatureAdded
type Handler[E Event] func(ctx Context, e E) Outcome // Outcome = Continue | Veto(reason)

type Bus struct{ ... }
func Subscribe[E Event](b *Bus, p Phase, h Handler[E]) Subscription  // typed, no sink interface
func Emit[E Event](b *Bus, p Phase, e E) (Outcome, error)            // fan-out, collects veto
```

This maps the COM concepts to idiomatic Go:

| COM connection-point concept | Typed bus |
|---|---|
| `XEventsObject` + `XEventsSink_Event` split | `Subscribe[E](bus, phase, handler)` — one call, generic |
| `EventTimingEnum BeforeOrAfter` | `Phase` argument (`Before`/`After`) |
| `out HandlingCodeEnum` veto | handler returns `Outcome` (`Continue`/`Veto(reason)`) |
| `NameValueMap Context` | typed event struct `E` (fields, not a string map) |
| per-event delegate type (316 of them) | one generic `Handler[E]`; events are plain structs |

- **Veto**: a `Before` handler returning `Veto(reason)` cancels the operation; the
  emitter checks the aggregate outcome before proceeding. Replaces the awkward
  `out` parameter with a return value.
- **Scoped buses** (application / document / modeling) group events for
  discoverability, exactly as the COM event sets did — just typed.
- **Frame-safe delivery**: handlers that mutate the document enqueue a command/
  deferred runner (core/00 phase 1) rather than mutating mid-emit.

## Out-of-process events (add-ins)

The same events stream to gRPC add-ins via `Events.Subscribe` (ADR-0003). For
vetoable `Before` events the add-in must reply `allow|veto{reason}` within a
**deadline**; past the deadline the host proceeds. This fixes a real COM footgun:
a hung add-in can no longer freeze the host on a veto.

## Change processing

The COM `ChangeManager`/`ChangeProcessor` (let add-ins participate in edits) maps
onto: subscribe to `Before`/`After` model events + issue commands. No separate
framework needed — the command + event primitives compose into it.
