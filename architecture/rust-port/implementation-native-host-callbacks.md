# Native host callbacks and owned replies — partial R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This connects the [native lifecycle adapter](implementation-native-addin-lifecycle.md)
and [owner dispatcher](implementation-dispatch.md) under ADR-0002. Reference:
Go `head/internal/addinhost/hostcall.go` at the fixed migration revision.

`HostCallbackBinding` installs one injected Send + Sync API Caller transport and
returns the exact `ObkHostCall`/`ObkHostFree` function pointers. The native registry
holds only a transport endpoint and endpoint-use counter, never a model or UI.
The application must inject its queued transport; the owner supplies the actual
host only while draining. No unsafe Send/Sync implementation is added.

The callback checks missing output slots before writing anything, then initializes
valid output slots to null/zero. Negative C lengths, null methods and null nonempty
payloads produce owned error replies before references are made. Method strings
require UTF-8. Method and request bytes are copied before dispatch. Native callers
still owe readable NUL strings/input extents and writable aligned output slots;
scalar/null checks cannot prove arbitrary pointer validity.

Success and error responses are host-owned allocations, including empty success.
They retain Go's trailing storage byte, excluded from the C response length.
Only ObkHostFree removes a live allocation from the host arena. Null frees do
nothing; unknown addresses are refused and counted without dereferencing them.
The native caller must free each returned live allocation once. An address lookup
does not certify stale-pointer ownership after allocator address reuse.

Close stops new callback admission. Endpoint leases acquired before close can
finish; close does not revoke an accepted queue mutation. The use counter measures
transport endpoint leases, not proof that every foreign thread has joined. Owner
queue shutdown and native worker joins remain explicit separate operations.
HostFree and response bookkeeping remain available after binding retirement so
late replies can be released. A Rust dispatch panic is contained at the C entry,
returns an owned error and retires admission. Foreign faults/aborts are not
contained, and model rollback is not inferred from panic containment.

The context-free ABI cannot identify an old function pointer's host generation.
A used callback identity therefore cannot be replaced or reinstalled in the same
process after retirement. This prevents a stale add-in callback from reaching a
new session endpoint. It does not prevent ordinary add-in reload while the same
host binding remains active. A reusable host-generation design would need separate
quiescence evidence or a contract change; it is not silently assumed here.

Seven tests run callback cases in isolated child processes, because identity
retirement is process-wide. They cover copied success and error ownership,
empty responses, shape/UTF-8 rejection, invalid outputs, unknown/null frees,
ordinary transport errors, late free after binding drop, duplicate admission,
panic retirement, retired calls, and accepted timeout with a dropped source buffer.
Real Go and Rust libraries activate on workers through one queued callback binding;
the owner drains the echo call and each add-in releases its host-owned response.
The timeout case proves the queued copied bytes still execute after the C caller
returns. These are ABI/queue echo tests, not a typed document workflow.

Full Linux validation is `make gate`. The binding remains a low-level transport
component. Managed registry/shutdown ordering, native event sink attachment,
typed Go/Rust document edits, CLI/window drain integration and foreground
MCPBridge acceptance remain required. Windows/macOS tests remain deferred and
their explicit failing pipeline definitions are unchanged.

A [typed Rust document workflow](implementation-native-typed-workflow.md) now
verifies create/edit/undo/save/reopen through these C callbacks and the file owner.
Loaded Go/Rust add-in document workflows remain separate required evidence.
