# Native add-in lifecycle and reply ownership — partial R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This extends [native admission](implementation-native-addin-probe.md) under
[ADR-0002](../decisions/ADR-0002-api-and-addin-host.md), using the fixed Go
`head/internal/addinhost/dl_shared.go` implementation.

`open_addin` admits a trusted library and retains its exact API function exports.
`probe_library` now uses the same admission path and returns copied metadata.
`LoadedAddIn` implements explicit activation, notification, optional automation,
and deactivation. Both callbacks supplied to Activate use the published C types;
none of these operations invents a context argument or a new wire contract.

A wrapper begins Inactive. Activation is allowed only there. Success marks it
Active; an unsuccessful activation leaves NeedsCleanup because foreign partial
initialization cannot be rolled back by Rust. Deactivation is an explicit cleanup
path, and failure also leaves NeedsCleanup. Only successful deactivation returns
to Inactive. Notifications and automation require Active; cleanup state refuses
further work. Already-inactive deactivation is a no-op. No foreign work occurs
on Rust drop: cleanup can require a live owner dispatcher and must be scheduled.
Opened libraries remain resident under the existing policy.

The wrapper is not Sync and lifecycle methods borrow it exclusively. This does
not prove library isolation: repeated dlopen can resolve the same foreign state.
Unsafe lifecycle callers must obey affinity, exclude aliasing foreign callers,
and retain host callbacks/context until deactivation succeeds. Foreign faults
remain outside Rust containment. Native callback registration and queued
activation orchestration are still required.

Automation treats method/payload as the target add-in's opaque contract. It
refuses an absent optional export and embedded NUL method names. Request/event
lengths must fit signed C int; empty inputs use null plus zero. Native output
starts with null/zero slots. Negative response lengths and null/nonzero replies
are refused before Rust references are created. Positive nonnull extents remain
the trusted library's safety obligation.

A reply guard calls the add-in's own ObkFree once for every nonnull response,
including error, zero-length allocation, and invalid signed-length paths. Bytes
are copied before release. Successful responses remain opaque bytes; failure
responses require UTF-8 and retain the status/method/error text in their decline.
The host never substitutes its allocator for the add-in's allocator.

Seven new tests cover states, repeated activation, cleanup failures, inactive
refusal, optional automation, notification failure, NUL methods, C-int limits,
empty/null input and reply shapes, malformed/error responses and exact release
counts. Rust fixtures poison and retain retired buffers so copy-before-free is
observable without reading freed memory. Their release counter counts each
query response too; the assertions account for those buffers explicitly.

The fixed real Go echo fixture activates using a named Rust host buffer store,
reads its response, and releases it through the supplied ObkHostFree. The host
then copies the Go-owned automation reply and invokes the Go ObkFree. Notify
and deactivation complete through the same resolved exports. This is real
cross-runtime echo/ownership evidence, not a typed document workflow. Full
local acceptance is `make gate`; Windows and macOS remain deferred.

Application add-in registry, installed-bundle discovery, queued C callbacks,
callback retirement, host error-buffer ownership, event transport, automation
reentry, worker activation coordination, typed Go/Rust document workflows, and
foreground MCPBridge acceptance remain required. The unsafe low-level wrapper
is not yet a managed application add-in host.

The [native host callback binding](implementation-native-host-callbacks.md) now
connects the C signatures to queued transport and tracks host-owned replies.
Managed lifecycle and typed document workflow acceptance remain open.

The native registry now owns registration policy and explicit cleanup orchestration;
see [registry implementation evidence](implementation-native-addin-registry.md).
Application startup, discovery, wire routing and foreground acceptance remain open.
