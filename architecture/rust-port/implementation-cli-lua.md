# Isolated Lua 5.1 and CLI scripting — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This implements the CLI execution portion of
[ADR-0011](../decisions/ADR-0011-scripting.md), using the pinned Go
`script/engine.go`, `script/gopherlua/`, and `cmd/oblikovati-cli/cmd_script.go`
as behavior references. It does not complete the console/editor or rule triggers.

The selected adapter uses [mlua 0.12.2](https://github.com/mlua-rs/mlua) with its
Lua 5.1 and vendored features. Cargo records the exact selected dependencies.
The package reports MIT licensing and a Rust 1.88 minimum, below the application's
pinned Rust 1.99.0 toolchain. The native VM is imported only by the script VM
module. It is initialized only in an isolated worker process, not in the parent
that owns document sessions. The standalone worker binary supports development
and containment tests; the packaged CLI launches its own private worker entry,
so no second installed executable is required.

## Ownership and host calls

The parent owns documents, GUID issuance, persistence and the canonical FileHost.
The worker receives source, limits and method names; it owns no model pointers.
`oblikovati.call` and generated dotted method groups use one structural JSON
bridge. Only the parent invokes model methods, through existing session commands
and transactions. The child waits for each host reply, and the parent performs
one event at a time. `poll` supports a future native frame loop; CLI `wait`
services the same protocol. An engine lease prevents concurrent executions and
is released only after worker termination and thread cleanup.

The file-owner method inventory comes from the existing API-constant route arrays,
with stable sorting and deduplication. It describes ported routes with their owner
and argument preconditions; it does not advertise unconfigured application option
owners or promise that the whole API release is implemented. No copied wire DTO
schema or model operation exists in the interpreter adapter.

## Language and conversion

The worker runs actual Lua 5.1 and opens base, table, string and math support.
OS, filesystem, network, process, package loading, debug and FFI libraries are not
opened. Loading functions, manual garbage collection and legacy module/newproxy
entries are removed. Print output is captured as tab-separated tostring values
and forwarded to the parent output sink. Host API file operations remain deliberate
capabilities; this is not a claim that a script has no external effects.

Dense 1..n tables become arrays; other tables become objects with scalar stringified
keys. Empty tables become empty objects. JSON arrays and objects become Lua tables,
null becomes nil, and an empty host response becomes an empty table. Invalid UTF-8
Lua strings use replacement text, matching the Go JSON string conversion. Nested
conversion is bounded at depth 64, including cycles. Unsupported values and
colliding stringified keys are named refusals rather than silently changed data.
Whole-number values outside the Lua safe-integer range are refused before they can
silently corrupt API IDs. Non-finite numbers are refused too.

## Limits and containment

CLI defaults retain the Go runner's 30-second wall bound and 256 MiB Lua heap
budget. The Rust target adds explicit instruction, stack, output, host-call and
source bounds: 20 million accounted instructions, 256 frames, 1 MiB output,
100,000 host calls and 1 MiB source. The instruction hook accounts in blocks of
1,000, with a bounded final block. Lua heap allocation is capped by the VM.
Linux also bounds child address space, stack and CPU use, and disables core dumps.
The address-space limit includes a fixed 128 MiB process/scratch allowance beyond
the Lua heap. The native runtime can still abort or fault; process isolation keeps
that failure out of the document-owning parent.

A parent monitor terminates the worker on deadline or cancellation even while
Lua is waiting for a host reply. New calls stop after cancellation. A host call
already committed is retained; cancellation and script failure do not imply
rollback. The host cannot safely kill an in-progress modeling operation, so its
completion can delay the owner's return even after the VM stops. Host panics
stop script admission and return an error, without claiming recovery of partially
changed model state.

IPC frames and stream backlogs are bounded. Normal completion waits for stream
EOF before returning its result, so child cleanup and coverage/profile flushing
finish naturally. Deadline monitoring also bounds a worker that sends completion
but does not exit. Failure and Drop terminate/reap the child and join stream and
monitor threads. Drop is not a model transaction rollback.

## CLI behavior and acceptance

`oblikovati-cli script run file.lua [--doc input.opd] [--save output.opd]` reads
bounded UTF-8 source, optionally opens a document, runs the script, and saves the
active document only after success. An explicit save performed by a script remains
committed if a later script error occurs. Print reaches normal CLI output and
failures produce a nonzero exit with diagnostics. Native paths stay as OS strings
outside the wire boundary, and source/document storage are separately injected.

Tests use actual workers for Lua 5.1, typed groups, structural arrays, host errors,
parse-only checks, forbidden globals, instruction/stack/memory/output/host quotas,
wall timeout, Stop during a committed host call, owner panic and abrupt native
worker failure. The parent remains usable after a failed child. CLI tests check
parameter edits with undo/redo, open-before-run, save-after-success, explicit-save
retention on failure, named I/O errors and operand validation. Real executable
cases save an actual native part package, read it through `info`, and refuse
missing, non-UTF-8 and oversized source files. Named fake source/file/GUID classes
cover controlled I/O failures without substituting fake successful writes.

Run `cargo test -p oblikovati-script -p oblikovati-cli` for these modules and
`make gate` for full local Linux acceptance. Windows and macOS remain deferred.
The remaining R7 work includes console/editor integration, API completion/help,
document rules, trigger ordering, responsive foreground UI tests and full legacy
script corpus parity. No rendered or solid-feature acceptance is claimed here.

The [Script Console controller](implementation-script-console.md) now supplies
nonblocking Run/Check, Stop, owner-frame polling and copied output/outcome state.
The native panel and editor still need integration.
