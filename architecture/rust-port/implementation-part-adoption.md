# Session part result adoption — partial R1/R3

The session now captures a sealed part-evaluation request with immutable recipe
inputs, document identity, and history generation. The history has one
`AtomicU64` generation counter. All existing edit, abort, undo, redo, and derived
refresh publication sites update that same counter. A read-only weak scope lets
queued work detect supersession or close without borrowing live model state or
keeping a closed document alive. Reopened histories have distinct scopes even
when the document ID and generation values match.

The request consumes its captured inputs through the existing ordered part
replay and returns a sealed completed receipt. Cancellation is checked between
entries and after the final operation. An incomplete replay drops its derived
outputs and cannot produce an adoptable receipt. A running native operation is
not interrupted; a later cancellation or adoption check discards its result.

`DocumentSession::adopt_part_evaluation` checks document identity, history
ownership, and the current generation before replacing the injected typed
`PartEvaluationSlot`. An existing slot also retains its document ownership until
explicit `clear`. Rejection preserves the previous result. Adoption changes no
recipe, reservation, undo/redo entry, or transaction event.

File-owned documents use the same methods through their existing borrowed
`SessionCommands` capability. A saved prism package is reopened and evaluated
through this route without making the document dirty.

Exact queries use `slot.current(&session)` and reject stale or foreign results.
The returned reference also borrows the session, so an owner cannot edit while
that exact query remains in use. A compile-fail check enforces this lifetime.
Presentation can use `slot.latest`, which exposes its document, generation, and
current/stale flag together with its report. It cannot claim an old result as a
current generation. Clearing storage retires its native outputs on the owner
thread. No type erasure or unsafe `Send`/`Sync` implementation was added.

Named-fake tests cover late results, failed destination checks, empty storage,
close, edits during evaluation, undo/redo/abort, and unchanged history/events.
A waiting worker observes an owner edit while retaining the original recipe.
Released Martensite tests check analytic prism volume across adoption, a late
result, depth edits, undo, redo, and close. A compile-fail check verifies that a
completed receipt containing a retained native prism cannot move across threads.

This establishes generation-checked storage for the current diagnostic part
outputs. It does not claim a complete extrude operation, a native background
scheduler, incremental prefix reuse, general scene publication, or the full
kernel validity/tolerance/naming contract. Those remain open in ADR-0004 and
ADR-0007. Required legacy kernel capabilities still belong in Martensite.

Run `CARGO_NET_OFFLINE=true make gate` for the full local Linux checks.
