# Owned sketch mirror — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Go reference: `model/sketch/edit_ops.go` and `addin/router/sketch_edit.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The released API 0.158.0 defines `sketch.transform` with `op: "mirror"` and an
owned mirror-line ID. The Rust host now resolves that line from the current
sketch during the command and returns fresh selected IDs. No wire contract,
client binding, method string, or dependency pin changes.

Translation copies and mirror copies use one preparation, reservation, adoption,
report, history, and recompute path. The old separate document translation-copy
entry point is removed. Existing model and session translation-copy wrappers
remain callers of that shared path. The reflection matrix uses the existing
scale-safe direction normalization and overflowing endpoint subtraction helpers.
Coincident endpoints, missing/non-line axes, invalid selections, and non-finite
prepared coordinates or affine offsets decline before identity reservation.
Subnormal directions and overflowing endpoint differences are tested. There is
no new runtime epsilon or tolerance policy. Existing move/rotation arithmetic
is unchanged, including the signed-zero translation regression.

Reflection transforms each shared defining point once and each selected ellipse
axis once. It reverses the sweep direction of selected arcs. This intentionally
corrects Go's clone path, which retains `CounterClockwise` after reflection and
therefore selects the complementary sweep. A regression run with reversal
disabled returned midpoint x = -sqrt(2) instead of +sqrt(2); restoring reversal
passes. Both minor and major sweeps are checked. Curve roles, source constraints,
source geometry, IDs, and existing references stay intact. New copied entities
have independent identities; hidden points remain hidden unless selected.

The mirror line can itself be selected. Duplicate selections are unique.
The axis ID belongs to the sketch; it is not a centimetre coordinate and is not
scaled again at the API boundary. An edit reads the current solved axis. A test
that moved an axis sharing points with an arc initially assumed a rigid result;
the structural arc solve correctly changed those points. The corrected test
checks the actual current axis against an independent orthogonal projection
formula. Mirror creates geometry copies, not an associative symmetry constraint.

Model tests cover horizontal and oblique reflection, shared hidden points,
ellipse directions, spline points, minor/major arcs, duplicate selections,
reflection of the axis itself, double reflection, invalid axes, atomic refusal,
source isolation, curve roles, and JSON reload. API tests cover working scales,
part/assembly owners, current axis edits, keys, history, abort, saved reload,
read-only refusal, observer veto, response rejection, and consumed reservations.
Existing move, rotate, and copy tests remain active. The baseline Go point-mirror
and centerline-mirror tests pass; neither checks Go's arc sweep defect.

The native test mirrors the stored parameter-driven prism's four profile lines
across its first line through the typed API. It checks reflected coordinates,
source rows, fresh keys, and reference resolution. As with translation copy,
the eight free copied coordinates cause the existing whole-sketch readiness
check to decline. The native scene removes the unavailable body and reports
`FreeVariables(8)`; undo restores the byte-identical original image. X11 pixel
checks verify actual presentation under isolated 1920×1080×24 Xvfb, software
Vulkan, and validation layer 1.4.321.

Elliptical arcs, independent offset-spline copies, per-entity style tables, trim,
split, extend, interactive mirror UI, hardware presentation, and general profile
regions remain open. This step does not close R3 or the full-port epic.

The full local `make gate` passes for all 13 crates in the discovered workspace:
all targets/features, Clippy, formatting, instrumented native tests, doc tests,
Markdown/link/SPDX checks, workflow lint, and duplication. Every crate exceeds
80% line coverage. Model coverage is 98.80% (17,771/17,986), session 99.05%, and
automation 96.26%. The reflection/point-transform implementation, copy operation
classification, shared copy preparation, document/session commands, workspace
wrappers, and wire transform route each have 100% line coverage. Duplication is
0.75% (502 of 66,840 Rust lines). Cross-platform review found no new tolerance,
platform-sized identity, map-order dependency, or changed move/rotation arithmetic.
Kernel algorithms, recognizers, fallback sites, and assertions are unchanged.
