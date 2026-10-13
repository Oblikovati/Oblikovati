# Owned sketch selection translation — partial R3

This step implements selection translation through typed document commands and
canonical `sketch.transform` with `op: "move"`. It implements
[issue #3770](https://github.com/Oblikovati/Oblikovati/issues/3770).

Inspection and translation share one owned shape catalogue for points, lines,
circles, and circular arcs. The old inspection-only catalogue was moved into the
shared module. Translation resolves the full selection, collects unique defining
points, and prepares every finite output before mutation. It moves shared points
once, including when a point and several incident curves are selected together.
Duplicate selections follow the same unique-point rule as Go. The report lists
authored defining point IDs in entity order, not final solver displacement.

The edit keeps curve identities, incidence, circle radius, arc direction,
constraints, construction roles, entity order, and reference keys. It allocates
no IDs. Selecting one point can deform its incident curves. Unknown entities,
nonfinite vectors, and overflowing output coordinates decline atomically with
the offending value. The scalar point transform retains Go's affine expression,
including signed-zero behavior; an independent Go capture returns `0 0` for the
fixture checked by the Rust bit assertion.

The document command invalidates numerical evidence and uses automatic
current-source recompute. Fixed constraints and grounded projections still
control their points after the seed edit. Numerical failures remain visible
through the existing health path. Response preparation errors and observer veto
discard the candidate. Groups, abort, undo/redo, and save/reopen use normal history.

The wire path imports API 0.157.0 DTOs, method constants, and typed client methods.
It converts finite centimetre vectors to document working units and rejects an
unrepresentable conversion before the command runs. Like Go, the API declines
empty selections. Typed model calls permit an empty selection as a no-op.
Ordinary zero moves create no committed step or saved recipe change; accepted
equal candidates retain the normal history generation tick. Signed-zero changes
remain real recipe bit changes. Part and assembly documents share the path;
other document kinds and read-only hosts refuse the edit.

Tests cover the four current kinds, shared points, duplicates, unrelated points,
radius/direction/roles/order/keys, invalid selections and vectors, unit and output
overflow, signed zeros, atomicity, fixed and grounded recompute, no-op committed
history, identity reservations, response failure, veto, groups, undo/redo, and
native file lifecycle. Independent Go shared-point, selected-point, and router
error tests pass at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

Rotation is now covered by the [rotation evidence](implementation-sketch-rotation.md).
Copy, mirror, trim, split, extend, additional geometry kinds, and native drag remain required. Unported wire operations decline by name; this step does
not claim the full transform family or native rendering acceptance.
