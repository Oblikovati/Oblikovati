# Typed Go and Rust document workflows over C callbacks — partial R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This verifies the composition of the native callback binding, owner queue,
FileHost, public typed client, common model commands and native persistence.
See [callback evidence](implementation-native-host-callbacks.md).

A worker uses the released Rust `Client` over a named test transport that invokes
the exact C host function pointers. The native callback copies requests into the
bounded queue. A separate owner loop supplies FileHost when draining; the worker
never captures the document owner, filesystem, GUID issuer or observer factory.
The test uses isolated real temporary files for its native save/reopen acceptance
and a deterministic named GUID source. Callback and filesystem failures fail the
workflow; nothing is mocked as a successful save.

The typed client creates a part, authors a `width` parameter at 20 mm, creates one
sketch and one circle, grounds the circle's center, and creates its driving radius
dimension from `width`. The reported DOF is zero. Changing width to 35 mm updates
the measured circle. Undo restores the previous radius bits; redo restores the
edited bits. Save As writes an actual `.opd`, then close/open preserves the circle
radius bits, reference key, parameter expression, and clean document state while
assigning a new session ID. All response buffers are released through HostFree.

The final owner check uses `DocumentSnapshot::require_resolved_sketch`, not the
geometry-only readiness reader. A part sketch has projected origin inputs, so
current document resolution is part of its strict convergence/DOF acceptance.
The test keeps that guard and checks the reopened native model independently of
the client's response fields. The driver, schema and transport are not replaced
with a simplified fixture document or a different edit path.

Run `cargo test -p oblikovati-addin-host --test typed_workflow` for the focused
workflow and `make gate` for full local Linux acceptance. The test owner loop has
an explicit completion deadline and uses wake signals rather than fixed sleeps.

This is a typed Rust client calling the C ABI in the test worker. It is separate
from the real Go/Rust shared-library echo/lifecycle tests. Those two evidence sets
do not jointly prove a loaded add-in's typed document workflow. The loaded scenarios below extend this evidence. Managed host lifetime,
native window/CLI drain loops,
queued event delivery and foreground MCPBridge acceptance remain required. The
part has a fully constrained sketch recipe; no solid feature or rendered result
is claimed. Windows and macOS remain deferred.

## Loaded Rust add-in workflow

The same typed scenario now also runs inside an actual Rust cdylib. Its isolated
fixture Cargo project links the exact pinned API source plus serialization crates;
it imports no GPL application crate. The scenario and C transport source are
shared with the direct-client test so the library case cannot silently use a
smaller workflow. Native activation runs on a worker while the owner drains the
same FileHost and callback binding.

The add-in performs create, constrain, parameter edit, undo/redo, save, close and
reopen during activation. Its optional automation export returns only a copied
report afterward; it makes no synchronous host calls from that automation entry.
The report is copied and released through the library's ObkFree, then the host
independently checks the actual saved path, current resolved sketch, radius bits,
parameter presence, distinct reopened session ID and clean state. Deactivation
completes before callback retirement. Host response allocations return to zero.

The fixture compiler separates common SDK/resource discovery from the metadata
stub compiler and the full-client fixture compiler. Cargo uses a temporary project
and separate target directory, so nested fixture builds cannot contend for the
application's target lock or modify the API checkout. Fixture failures fail the
same focused test and full gate; they are not skip conditions.

## Loaded Go add-in workflow

An actual Go C-shared library now performs the same typed document scenario. Its
isolated Go module imports only the exact pinned API checkout. It uses the owner's
C header with `OBK_BUILDING_ADDIN`, including the canonical two-argument notification
export. Its caller receives the injected C callbacks, copies replies before
HostFree, and releases error replies through the same owner. No copied wire DTOs,
method names, GPL application dependency or sibling kernel checkout is used.

The Go typed client creates the part and parameter, grounds and dimensions the
circle to DOF zero, edits width, checks undo/redo radius bits, and saves, closes
and reopens a real native document. It checks the retained reference key and
parameter expression. The Rust owner independently checks the reopened model
with the same strict document resolution guard as the loaded Rust case. Both
libraries advertise the compiled API version and produce identical final radius
bits. The Go automation export only reports copied observations after activation;
it does not call back into the owner. Both libraries deactivate before retirement,
and all host reply allocations are released.

The fixture compiler builds both libraries outside the application tree. The Go
build uses `GOWORK=off`, `CGO_ENABLED=1`, and `GOPROXY=off`, so a missing pinned SDK
or local toolchain fails this test rather than using a workspace or network substitute.
The focused test exercises the notification export as well as activation,
automation report ownership and deactivation.

These loaded scenarios close the library-boundary gap for this document workflow.
Managed registration/shutdown, event attachment, native presentation and foreground
MCPBridge acceptance remain required. No solid feature or rendered part acceptance
is claimed. Windows and macOS remain deferred.
