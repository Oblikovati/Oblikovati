# Typed Rust document workflow over C callbacks — partial R4

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
do not jointly prove a loaded add-in's typed document workflow. Real Go and Rust
add-in document scenarios, managed host lifetime, native window/CLI drain loops,
queued event delivery and foreground MCPBridge acceptance remain required. The
part has a fully constrained sketch recipe; no solid feature or rendered result
is claimed. Windows and macOS remain deferred.
