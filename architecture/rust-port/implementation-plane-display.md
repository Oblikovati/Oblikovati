# Authored work-plane display settings

Tracking: [#3749](https://github.com/Oblikovati/Oblikovati/issues/3749).
The datum owner and reference boundary remain ADR-0004 and ADR-0005.

## Source and edit order

Canonical `workPlanes.redefine` accepts display edits on fixed and offset planes.
It applies auto-resize, grounded state, then explicit size, as Go #1851 does.
Explicit size disables auto-resize. A later flag edit retains the explicit size.
Grounded state is a saved associativity attribute; it does not freeze formulas.

Size is two finite three-component corners in API centimetres. The source permits
unordered, coincident, or noncoplanar corners, so restore adds no extra geometric
restriction. Empty size is a no-op. Without explicit size, the current frame
produces a rectangle with a five-centimetre half extent along each frame axis.
Working scale converts the origin to centimetres; preferred units do not change
this rectangle. A normal flip changes derived corners. Explicit corners remain
unchanged. Failed geometry has no derived rectangle and retains explicit source.

## Ownership and storage

Display changes retain the feature key, definition, scalar ownership, and reads.
Mixed scalar, reference, and display edits use one candidate and one command.
Malformed edits are refused before adoption. Empty or unchanged edits do not
create a document generation. Origin planes remain immutable.

Nondefault display metadata requires datum program version 9. Default metadata
is omitted and retains the earlier minimum version. Unknown fields, nonfinite
corners, older program versions, and metadata on a point or axis are refused.
Recipe capture, restore, history creation, edits, and refresh use the same metadata
validation boundary. Exact source comparison includes signed-zero corner bits.

## Linux evidence and remaining work

Model tests cover physical units, defaults, precedence, orientation, exact source
comparison, version refusal, replacement, and history adoption. Typed API tests
cover mixed edits, live formulas, local health, undo, abort, veto, copy, and reopen.
Package tests cover stored corners and strict metadata restore. The source Go
`TestWorkPlaneDisplayStateRoundTrips` and `TestWorkPlaneAutoResizeRoundTrips`
regressions pass at the fixed migration baseline.

Acceptance requires the full local Linux `make gate`. Applying these settings to
the native viewport remains R1V/R4 work. These tests do not certify rendering or
kernel geometry. Windows/macOS pipelines and publication readiness are unchanged.
