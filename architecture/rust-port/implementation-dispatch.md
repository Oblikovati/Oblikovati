# Owner-thread dispatch queue — partial R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This implements the dispatch boundary in
[ADR-0002](../decisions/ADR-0002-api-and-addin-host.md), using Go
`addin/dispatch/queue.go` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The automation crate's `owner_dispatch` returns a cloneable `QueuedCaller` and
one `OwnerDispatch`. The transport implements the released API's `Caller`, so a
worker uses the existing typed client. Requests own copied method names and
argument bytes. The shared queue contains no model, kernel handle, host closure,
or UI object. The owner supplies its borrowed `SessionHost` or another `Caller`
when draining. A compile-fail test proves that the drain capability cannot move
to a worker thread; no unsafe code or thread-trait override is used.

Admission is FIFO and bounded by the injected capacity. Zero capacity follows
Go's minimum of one. Producers wait for capacity within one monotonic deadline
that also bounds waiting for the reply. Successful admission invokes the injected
wake callback once, outside the queue lock (#1493). A CLI can explicitly use
`PollingDispatchWakeup`; native presentation must supply its real window wake.
Wake failure names the error but does not revoke an admitted request.

A timeout before admission leaves no queued request. A timeout after admission
returns a named error and leaves the request eligible to run. Dropped reply
receivers cannot block the owner. Neither timeout nor wake failure authorizes an
automatic retry. This preserves the Go accepted-mutation rule.

Drain runs only the current frame budget. Zero means the admission snapshot at
entry, so concurrent arrivals wait for the next frame instead of prolonging an
unbounded drain. This makes the Go source's "currently queued" contract explicit
and satisfies the Rust bounded-frame design. The owner releases capacity before
calling the host and holds no queue mutex during model work. Synchronous calls
from the creator/owner thread are refused before admission, including calls made
by a callback during drain.

Close is idempotent. It stops admission, drops undrained requests, releases reply
and capacity waiters, and rejects future calls. Owner drop does the same.
Ordinary host errors reach their matching waiter and leave the queue available.
A Rust host panic reaches the active waiter as an error and stops admission;
remaining requests are abandoned. This does not claim model rollback after a
panic, recovery of the host, or containment of faults in foreign libraries.

Fifteen queue tests cover FIFO, limits, capacity/minimum capacity, deadline
classification, delayed execution, copying after caller timeout, wake failure,
polling mode, owner reentry, domain errors, panic shutdown, drop, close, and
arrivals during a drain. Two typed-client tests edit a real parameter-driven
circle through `SessionHost`, compare queued geometry bits with a direct edit,
verify undo, and prove that a timed-out edit later runs once with one undo step.
The pinned Go `addin/dispatch` suite passes. Full acceptance is `make gate`.

Native C callback admission, signed C lengths and buffer ownership, cancellation
before admission beyond deadlines, callback registry lifetime, library activation
and deactivation, event backpressure, and actual native-loop integration remain
required. This queue is a tested transport boundary, not a live add-in host or
proof of Go/Rust shared-library compatibility.
