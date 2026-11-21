# Owned sketch line split — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Go reference: `model/sketch/edit_trim.go` and `addin/router/sketch_edit.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The released API 0.158.0 defines `sketch.transform` with `op: "split"` and a
centimetre pick point in `vector`. The Rust host now implements that operation.
It validates the complete selection, then splits its first entity, which must
be a line. Other valid selected entities remain unchanged, as in the Go route.
The response contains the retained first-line ID followed by the new second-line
ID. No contract, binding, wire method, or dependency pin changes.

The model projects the pick orthogonally onto the finite line support. A split
requires an interior parameter and a representable junction distinct from both
endpoints. It reserves a new defining-point ID and a new line ID before adopting
a complete candidate. The first line keeps its identity and first endpoint;
the second line keeps the original last endpoint. Both share the new hidden
junction. Other curves that use the original endpoints keep those identities.
The new half inherits explicit construction and centerline roles separately.

Existing constraints, dimensions, and projection bindings are retained. No
collinearity, length driver, or other authored constraint is added implicitly.
Constraints that name the original line continue to govern its first half;
constraints on the original endpoint pair continue to refer to that pair.
This is a seed geometry edit, not an associative split feature. The current
diagnostic prism's explicit selected-line list is not expanded automatically;
general profile-region consumption remains separate port work.

Projection scales the line direction before forming its squared length. Tests
cover working-coordinate scales from 1e-150 to 1e150. The Go fixed parameter
margin of 1e-9 is not copied: a representable strict interior junction can be
closer to an endpoint. Non-finite input, overflowing coordinate differences or
normalized offsets, non-interior projections, coincident endpoints, and rounded
junctions that collapse onto an endpoint are explicit declines before reservation.
This bounded floating-point construction does not claim an exact-predicate
intersection kernel or support for all extreme finite-coordinate configurations.
There is no new runtime tolerance literal, perturbation, or retry.

Copy, mirror, and split now share identity reservation validation and role
transfer. The duplicate copy-only reservation implementation is removed.
`SketchEntityEditError` preserves geometry and allocator failures as typed causes;
the existing `SketchCopyError` name remains an alias. Display labels now describe
a sketch entity edit rather than mislabeling split failures as copy failures.
Failed response preparation, observer veto, group abort, and undo retain consumed
IDs. Save/reopen preserves those reservations. Geometry is published atomically.

Model tests cover off-line and diagonal picks, shared endpoint incidence, hidden
junctions, stable source IDs, roles, retained constraints, JSON reload, scale,
near-endpoint picks, invalid input, representational collapse, and typed allocator
exhaustion without a partial split. The copy/mirror reservation regressions remain
active against the shared allocator. API tests cover centimetre conversion,
first-selected semantics with full validation, original and new keys, part and
assembly owners, empty/missing owners, read-only refusal, reply rejection, veto,
undo/redo, abort, and saved reload.

The native test splits the first line of the stored parameter-driven prism
through the typed API. It verifies both half-line coordinates, original/new
reference keys, reference resolution, and unchanged other sketch rows. The new
junction has two free coordinates. The whole-sketch readiness check reports
`FreeVariables(2)` and removes the unavailable body; undo restores the byte-identical
original image. The existing X11 pixel check verifies actual presentation under
isolated 1920×1080×24 Xvfb with software Vulkan and validation layer 1.4.321.

Trim, extend, other split curve kinds, associative profile rewriting, interactive
split UI, hardware presentation, and full R3 acceptance remain open.

The baseline Go `TestSplitLineMakesTwoAtPoint` test passes. The full local
`make gate` passes for all 13 crates in the discovered workspace, including all
targets/features, Clippy, formatting, instrumented native tests, doc tests,
Markdown/link/SPDX checks, workflow lint, and duplication. Every crate exceeds
80% line coverage. Model coverage is 98.78% (17,921/18,143), session 99.05%, and
automation 96.26%. Projection, shared identity/role handling, document/session
split commands, workspace routing, and the wire transform route each have 100%
line coverage; split adoption has 90.14%. Duplication is 0.76% (510 of 67,096 Rust
lines). Cross-platform and comment review found no platform-sized permanent ID,
new runtime epsilon, unordered allocation, or kernel algorithm change. This is
partial port evidence, not full milestone acceptance.
