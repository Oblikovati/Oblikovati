# Native catalog queries on the desktop owner — partial R4

`NativeCatalogCaller` combines a completed native catalog snapshot with an
injected model caller. It uses the same List/Get router as native C callbacks,
imports canonical DTOs and method constants, and forwards other calls unchanged.
The snapshot contains copied `AddInInfo` values; no library exports, native worker,
model state, or lifecycle alias is copied into it. Borrowed and owned typed clients
can use this caller. A fresh caller is required for a later completed catalog.

`NativeAddInDesktop::script_host` now supplies this catalog caller over its existing
document host. Direct owner calls through the service adapter therefore see the
registered native add-ins rather than the base file host's catalog. A query does
not start foreign work or require synchronous native-worker reentry. In-flight
activation is not declared complete by a catalog read.

Named model-caller tests verify canonical List/Get results, missing IDs, malformed
requests, and exact forwarding of an unrelated payload once. The real Go/Rust
native-window workflow queries each newly loaded identity through the desktop
owner and checks the listed and described records before activation. This extends
the existing callback-side catalog checks using the same live registration state.

Read-only catalog queries are this change's scope. Activation, deactivation,
automation, and policy mutations still need owner control routing that avoids
synchronous owner reentry. Full script method advertisement, installed policy,
native event delivery, GUI management, and foreground MCPBridge remain open.
The default script console wraps the document owner inside native startup;
complete script-service transport composition is not claimed by this adapter.
Validation is local Linux; no remote push, PR, or deferred-platform test is included.

Full local `make gate` passed across all thirteen crates with software Vulkan
and validation enabled. Native add-in host production line coverage was 96.82%;
duplication remained below 3%. The known AMD presentation acceptance remains open.
