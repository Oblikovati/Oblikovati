# Native add-in owner runtime — partial R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This composes [native registration](implementation-native-addin-registry.md),
[bounded dispatch](implementation-dispatch.md) and
[loaded typed workflows](implementation-native-typed-workflow.md) under ADR-0002.

`NativeAddInRuntime` owns one native registry on one persistent worker thread and
one bounded owner dispatch queue. All library loads, metadata reads, activation,
notification, automation and cleanup use that assigned thread. The runtime and
its owner drain capability cannot move to another thread. Neither the worker,
its control messages nor the C callback registry captures documents, file access,
GUID sources, native windows or a model caller. The caller supplies its borrowed
FileHost or other canonical caller each time the owner polls.

Native controls have nonblocking start and poll operations. A native frame can
start activation and continue draining callbacks on later frames. A CLI owner can
use `wait`, which drains the same queue until control completion. There is at
most one outstanding native control; starting another is a named decline, not an
unbounded worker backlog. Payloads and paths are copied into worker ownership.
A separate cloned typed transport can serve worker clients through the same
bounded owner queue. It contains no model and refuses owner-thread synchronous
calls as the underlying dispatch contract requires.

The injected wake signal notifies the native loop on both admitted host requests
and native control completion. The CLI wait also checks completion between wake
signals. The native control itself has no timeout or cancellation claim: a hung
in-process library cannot be safely killed. C host-call deadlines keep their
existing meaning, including accepted requests that may execute after timeout and
must not be retried automatically.

## Registry queries during activation

The owner keeps a copied canonical registry snapshot from completed controls.
`addins.list` and `addins.get` are served from it while foreign activation waits
for host calls. They do not need to borrow a registry that is executing native
code, or block on the worker that is waiting for their reply. The activating
entry remains inactive in this snapshot until activation completes, as in the
Go registry. Updates publish after each completed control, including failed
activation and cleanup. Cleanup quarantine remains available through the host
lifecycle accessor, separately from the public active flag.

These routes import API method constants and canonical request/reply types. Other
methods reach the borrowed model caller. The shared JSON argument/reply functions
were moved out of automation's root into one owned module, so the runtime does not
copy decode rules. Object/null arguments, errors and nullable collection layers
retain the existing wire behavior. This runtime composition implements these two
registry query routes; it does not claim native lifecycle wire routes are complete.

## Cleanup and failure

Shutdown keeps the owner drain available while the persistent worker cleans up
all libraries. Successful all-entry cleanup retires callbacks, closes owner
admission and joins the worker before returning the shutdown completion. Failed
cleanup publishes quarantine state and leaves the worker and dispatch available
for explicit retry. Libraries must join their callback workers before reporting
successful deactivation. Native libraries stay mapped for process lifetime.

Owner panics are contained by existing dispatch before a C reply is returned;
activation receives an error and remains quarantined until cleanup. Unexpected
worker loss declines the pending control and closes owner admission. It does not
assert rollback of a partial model change or recovery of foreign state. Drop is
not coordinated application shutdown: it closes queue admission but performs no
foreign deactivation. Explicit shutdown with a live drain loop remains required.

## Acceptance

The real Go and Rust API-only document add-ins now run through this production
runtime. During activation both typed clients list and fetch their own registered
identity, then create a constrained part sketch, edit its parameter, undo/redo,
save, close and reopen a real native file. The owner independently checks the
resolved reopened model, radius bits, parameter presence, session identity and
clean state. Both languages produce identical final radius bits.

Additional real-library cases certify thread affinity across activation,
notification, automation and cleanup, and make host calls from both activation
and deactivation. Other cases check cleanup retry, owner panic containment,
pending-control refusal and retired typed transports. A named failed wake signal
injects a worker panic after a completed load; a channel gate makes the following
accepted control lose its worker deterministically. This deliberately violates
the normal nonblocking/nonpanicking wake contract to test failure containment;
it is not claimed as a normal wake implementation.

Shared fixture assertions, identity generation, reports and process isolation
replace duplicated test scaffolding. No callback identity is reinstalled in a
process. Run `cargo test -p oblikovati-addin-host --test native_runtime` for the
focused runtime tests and `make gate` for full local Linux acceptance.

Remaining work includes executable native-window and CLI/script integration,
installed bundle discovery, persisted user policy, native lifecycle wire routing,
queued event delivery and foreground MCPBridge acceptance. The frame/wait runtime
is available for these integrations; no rendered part or solid feature acceptance
is claimed. Windows and macOS remain deferred.

The [CLI Lua port](implementation-cli-lua.md) now executes scripts in an isolated
worker while the parent services the canonical file owner. It uses structural
script IPC rather than loading the VM into the native add-in worker. Executable
native-window integration and remaining lifecycle wire routes remain open.

The [desktop native service adapter](implementation-desktop-native-addins.md) now
services real Go/Rust document add-ins through the production native frame loop.
Executable startup selection and foreground MCPBridge acceptance remain open.
