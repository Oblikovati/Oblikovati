# Owned sketch primitive creation — partial R3

This step implements the common document command for point, line, circle, and
circular arc creation. It implements
[issue #3769](https://github.com/Oblikovati/Oblikovati/issues/3769).

`SketchPrimitiveCreation` carries typed working-unit coordinates and explicit
curve construction roles. Each defining point can be new or an existing point
owned by that sketch. Shared points retain identity, including grounded datum
anchors. Plain points have no construction flag. Arc inputs retain the authored
sweep direction and use the existing structural circularity equation.

`DocumentSnapshot::create_sketch_primitive` validates the owner, all point inputs,
and circle radius before reserving IDs. It builds on a private geometry copy and
publishes through the existing collection edit boundary. Invalid references,
coordinates, radius, or exhausted IDs cannot publish partial points or curves.
The common positive-radius guard also serves direct circle edits; there is no
second validation rule for commands.

New points receive permanent document-global IDs in constructor order, followed
by the curve ID. A standalone point uses its one point ID as its entity ID.
The report distinguishes all defining points from newly created entities. Shared
points consume no new reservation. Existing geometry, constraints, associations,
roles, entity order, and reference identities remain owned by the same sketch.

The session command uses normal history and automatic current-source recompute.
Response preparation happens before adoption. Response errors, veto, abort, and
undo never rewind reserved IDs. Save/reopen retains retired reservations. Groups
use the caller's history label. Workspace commands do not change document focus.

Numerical convergence does not imply zero free variables. New free geometry has
current solve evidence; the strict feature read path continues to refuse it until
fully constrained. A degenerate arc records its named solver failure through the
existing feature-health path. Creation does not claim that malformed geometry is
verified or silently repair it after solving.

Tests cover all four shapes, fully and partly shared points, grounded anchors,
construction roles, existing dimension constraints, both arc directions, invalid
inputs, identity exhaustion, snapshot atomicity, response errors, observer veto
and commit order, undo/redo, abort, groups, canonical inspection, reference keys,
part and assembly owners, save/reopen/copy, and retired reservations. Independent
Go primitive, shared-point, construction, and arc circularity tests pass at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

This is the application edit boundary. Canonical `sketch.addEntity` dispatch,
constructor overloads, inference, dimension expression authoring, native tools,
and kernel profile consumers remain required. No new API contract is introduced.
