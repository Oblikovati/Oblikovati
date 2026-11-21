# Script console over native desktop services — partial R4/R7

The executable startup order is now document owner, native add-in services when
selected libraries exist, then the script console. The console borrows the
native service's catalog caller during its ordinary owner poll. Native callbacks
and Lua model requests therefore use the same document owner, and Lua can inspect
the completed native registration snapshot. No VM or model moves to the foreign
library worker.

`DesktopScripts::with_methods` injects the configured host inventory. Native
startup includes canonical AddIns List/Get constants alongside the existing file
host methods. Unsupported native lifecycle mutations are not advertised. Empty
add-in startup retains the file-only console profile. One script engine is created
before native service admission, so engine setup failure cannot leave a started
foreign lifecycle without cleanup.

A real script-console integration test loads an API-only Rust document add-in,
services startup, then runs an isolated Lua worker. Lua lists the native catalog,
gets its registered identity, verifies completed activation, and lists the same
reopened document. The test uses the production console and native service types,
not a forged Lua result. Existing executable Go/Rust startup and script-worker tests
remain required by the full local gate.

The lifecycle shutdown order is console worker, native add-ins/callbacks, then
document queue. Native lifecycle wire commands, policy persistence, installed
bundles, event delivery, GUI console attachment, and foreground MCPBridge remain
open. This is local Linux integration; it does not claim complete script/API
coverage, solid-feature rendering, or other-platform acceptance. No push or PR
is included.

Validation on 2026-10-12: `make gate` passed for all 13 workspace crates on
local Linux with llvmpipe and the local Vulkan validation layer. Head production
line coverage was 94.85%; every crate exceeded 80%, and duplication remained
below 3%. Formatting, clippy, build, tests, doctests, documentation, SPDX,
workflow checks, and architecture checks passed. The native AMD presentation
fault recorded in `diagnostic-native-present-pixels.md` remains open.
