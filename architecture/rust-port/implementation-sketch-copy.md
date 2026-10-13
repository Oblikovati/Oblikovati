# Owned sketch copy — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Go reference: `model/sketch/edit_ops.go` and `addin/router/sketch_edit.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The released API 0.158.0 already defines `sketch.transform` with `op: "copy"`
and a centimetre vector. The Rust host now implements this operation through
its document command. The response lists fresh selected entity IDs in input
order. No public contract or dependency pin changes.

Copy validates the complete selection, prepares translated defining points
through the shared transform path, reserves every required identity, and appends
to an owned candidate. Publication occurs only after the complete copy succeeds.
Points, lines, circles, circular arcs, ellipses, and control/fit splines retain
their shared point relationships. Radii, ellipse axes, spline kinds and closure,
and construction/centerline roles remain intact. An explicitly selected point
becomes a standalone copied marker; implicit defining points remain hidden.
The model report includes every source/copy pair, including those hidden points.
Source geometry, constraints, dimensions, and projection bindings are unchanged.
Authored constraints are not duplicated, consistent with Go `CopyEntities`.

Repeated selections yield one copy per selected entity. This differs from Go's
repeated curve copies and avoids duplicate standalone/shared point identities.
Selecting an offset spline directly produces a named decline before reservation;
Go's default branch silently skips this unsupported kind. Elliptical arcs,
per-entity style tables, mirror, trim, split, extend, and interactive native copy
remain separate work. This is not full transform-family or R3 acceptance.

The command recomputes current sources and supports response rejection, observer
veto, grouped abort, undo/redo, storage, and reopen. Consumed IDs survive rejected
publication and history reversal. A rejected command can therefore mark the
document as needing a save even when its geometry is unchanged. Reload retains
that reservation state. Invalid selection or coordinates consume no IDs.

Model tests check the complete provenance map, shared points, hidden/visible
ownership, independent subsequent edits, exact ellipse axes, both spline kinds,
roles, unchanged constraints, JSON round trips, invalid selection, finite output,
offset refusal, exhaustion with typed causes, and faulty injected allocators.
Typed API tests check working scales, canonical responses, new reference keys,
part/assembly ownership, source geometry, read-only refusal, veto, response failure,
undo/redo, abort, and reservation persistence after reopen. The baseline Go
`TestCopyEntities*` tests pass, including its two planar copy tests; this does not
establish Rust support for every Go geometry kind.

The native test opens the stored parameter-driven prism and copies its four
profile lines through the typed API. It checks every copied coordinate and
reference-key resolution. Copy adds eight free point coordinates. The existing
whole-sketch readiness rule correctly reports `FreeVariables(8)` and removes the
unavailable body from the displayed scene. Undo restores the byte-identical
original image. The first test incorrectly expected the prism to remain visible;
a repeat run identified the exact typed readiness cause. No consumer rule was
relaxed. X11 pixel checks verify actual presentation on isolated 1920×1080×24
Xvfb with software Vulkan and validation layer 1.4.321. Hardware presentation
and general profile-region acceptance remain open.

The final full local `make gate` passes for all 13 crates in the discovered
workspace, including all targets/features, Clippy, formatting, instrumented
native tests, doc tests, Markdown/link/SPDX checks, workflow lint, and duplication.
Every crate exceeds 80% line coverage. Model coverage is 98.80% (17,697/17,912),
session 99.04%, and automation 96.26%. The copy preparation/report, document and
session commands, workspace wrappers, and wire transform route have 100% line
coverage; copy adoption and identity validation exceed 96%. Duplication is 0.74%
(494 of 66,710 Rust lines). Review found no new tolerance values, platform-sized
identity values, map-order dependency, or kernel algorithm changes. This evidence
uses the exact released API/kernel pins and does not close the full-port epic.
