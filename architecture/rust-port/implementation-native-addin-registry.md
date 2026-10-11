# Native add-in registration and coordinated cleanup — partial R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This implements the native registration and lifetime portion of
[ADR-0002](../decisions/ADR-0002-api-and-addin-host.md). It builds on the
[loaded Go/Rust document scenarios](implementation-native-typed-workflow.md)
and [callback binding](implementation-native-host-callbacks.md).

`NativeAddInRegistry` owns the single host callback binding and native library
wrappers. Loading requires the existing unsafe trust boundary and exact version
admission; it does not activate the library. Empty and duplicate IDs are refused
without replacing the current registration. Loaded mappings remain resident,
including rejected registrations. The registry implements the published
`contract::AddInRegistry` and returns the canonical `wire::AddInInfo`. It copies
identity fields from `wire::AddInManifest`; malformed manifest JSON leaves those
fields empty, matching the Go host, while the export ID remains authoritative.

Registration order governs listing and startup selection. Fresh entries use the
published startup policy. Disabled entries refuse activation, demand entries can
be activated explicitly, and already active entries are idempotent. Policy edits
do not stop an active entry. Unregister requires inactive state; its preference
survives re-registration. Unknown and invalid policy values are named errors. Deactivation of an unknown
ID remains a no-op, as in the Go host.
The registry delegates notification and opaque automation calls to the existing
canonical ABI wrapper and its response allocator rules. There is no new wire DTO
or method string, and no claim that unimplemented host methods are available.

Failed activation and deactivation retain `NeedsCleanup`. They are not listed as
active, but cannot be unregistered or reactivated until cleanup succeeds. The
host exposes this lifecycle state separately from the public active flag.

## Shutdown order

Explicit shutdown first stops new registrations, activation, notifications and
automation routing. It attempts every registered cleanup in registration order,
including partially initialized entries. A failure does not prevent cleanup of
later entries. It returns all named failures and remains in stopping state; it
keeps C callback dispatch live so foreign cleanup can make required host calls.
The caller retains the host and drains the owner queue during cleanup and retries.
Successful entries are already inactive and do not repeat foreign deactivation.

Only successful cleanup of every entry retires callback admission. The caller
then joins the lifecycle worker and closes the owner queue. Deactivation success
must mean the trusted add-in joined its callback workers; a host timeout cannot
kill foreign code or prove that it stopped. General application front-door
admission and pending owner requests still need orchestration by the application
loop. During foreign cleanup the C endpoint intentionally remains available.

Drop does no foreign deactivation and is not successful application shutdown.
It retires the owned binding through its existing Drop behavior; it cannot replace
explicit cleanup with live dispatch. Resident mappings protect Go runtime threads
from unloading, but do not prove that add-in commands, resources or workers were
cleaned up. No mutable model or native window is captured by the registry.

## Acceptance

Five isolated-process tests use actual Rust and Go shared libraries with one
queued callback identity per process. They check registration order, published
contract queries, manifest identity, disabled/demand policy, idempotent Go
activation, notification, copied automation replies and inactive reload. Admission
checks include empty, duplicate, null, missing-export and newer-version libraries,
a missing file, invalid policy and malformed manifest JSON.

Failure cases prove activation quarantine, all-entry shutdown after one cleanup
failure, stopped notifications and lifecycle admission, a still-working queued
callback during cleanup retry, and retirement only after successful retry. A real
Rust deactivation export also makes a synchronous host call; shutdown runs on a
worker while the owner drains that request, and only then retires callbacks.
The tests fail on compiler, callback or cleanup errors; no successful foreign call
is substituted by an inline mock.

Run `cargo test -p oblikovati-addin-host --test registry` for these checks and
`make gate` for full local Linux acceptance. Process isolation respects the
context-free callback identity's permanent retirement rule.

Remaining R4 work includes installed-bundle discovery, persistent user policies,
wire routing through the live application owner, event attachment, native/CLI
application startup and shutdown loops, and foreground MCPBridge acceptance.
The registry's unsafe lifecycle methods still require foreign affinity and a live
drain loop. Windows and macOS remain deferred.
