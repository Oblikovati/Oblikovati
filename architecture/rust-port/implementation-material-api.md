# Session-owned material catalog reads — partial R5/R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563),
under ADR-0010. Go `addin/router/material.go` at the fixed baseline serves these
catalog queries without an active document. Rust now retains that behavior.

Every `FileWorkspace` owns an injected catalog source and a lazy admission result.
The default source loads the embedded Go catalog. A workspace loads it once, on
its first valid catalog query, and retains either the immutable catalog or its
named failure. There is no global library, filesystem lookup, worker-owned model,
or repeated parse per API call. Source injection allows a named failing provider
to test the same production path. A failed catalog does not prevent document
queries and cannot become an empty success response.

The immutable file query host implements canonical materials/appearances List/Get
methods. The synchronous file host delegates to that same path after its existing
owner/reentry check. Lists preserve category display order and use the released
API DTOs. Material replies normalize an unset symmetry to isotropic, as the Go
router's read contract does. Unknown IDs and malformed Get arguments are named
errors; malformed arguments are rejected before catalog loading.

The file host inventory includes the four implemented reads, so CLI Lua, the
desktop script console, and native add-ins use the ordinary model caller.
Unported creation, update, assignment and physical-properties methods are not
advertised. The catalog is session-scoped; these queries create no document or
model transaction and do not call Martensite.

## Numeric response regression

A real Lua catalog request exposed the existing bridge's whole-number test:
the valid floating-point electrical resistivity `1e16` was rejected as an unsafe
identifier. Response conversion now checks the JSON numeric representation.
API floating-point tokens remain Lua doubles, including whole-valued physical
scalars. Integer tokens beyond Lua's exact range remain refused. The regression
covers positive/negative integral doubles, a fractional scalar, and both signs of
unsafe integer IDs. Lua argument conversion retains its separate exact-integer
rules; numeric write conformance remains required before asset mutations ship.

## Local evidence

Named source tests cover lazy admission, one successful load, retained failure,
malformed arguments before loading, unknown identities, the read inventory, and
queries with no document. Both immutable and mutable API hosts use the released
typed client. Real CLI and desktop child VMs list the full catalog and read steel
and its appearance. The steel density assertion is the catalog's sourced 7.87
g/cm³, not the unrelated 7.85 value used by a synthetic asset test fixture.

The shared API-only Rust and Go document add-ins now list both catalogs and read
steel and its appearance before their constrained-circle workflow. The full gate
runs those libraries through the owner callback queue, native desktop services,
and executable startup. This is C ABI integration evidence; it does not replace
foreground MCPBridge or material-rendering acceptance.

Run `make gate` on this Linux host for all workspace checks. Editable project and
document assets, persistence attachment, assignment-order ADR resolution, UI
editors, render appearance selection, analytic mass properties, and complete
material API conformance remain open. No kernel or exchange implementation is
added. Windows/macOS remain deferred; no push or PR is included.

## Validation receipt

On 2026-10-12, `make gate` passed for all 13 workspace crates on local Linux with
llvmpipe and the local Vulkan validation layer. Session production coverage was
98.97%, automation 96.45%, script 97.51%, and every crate exceeded 80%.
Repository duplication was 0.62%. Formatting, clippy, build, tests,
doctests, documentation, SPDX, workflow and architecture checks passed. Real
Go/Rust native add-in callbacks, executable startup, CLI Lua, and desktop Lua
passed with the new catalog queries. The AMD native presentation fault remains
open.
