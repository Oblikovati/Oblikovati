# Owned dimension creation

Tracking: [#3787](https://github.com/Oblikovati/Oblikovati/issues/3787).
Go reference: `model/sketch/dimension.go`, `dimension_advanced.go`,
`serialize_restore_constraint.go`, and `addin/router/sketch_dimensions.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

## Native constructor inputs

The model has typed constructor operands for distance, radius, diameter, arc
length, line angle, three-point angle, point-line offset, and tangent distance.
They use the existing native relation constructors and ownership checks.
Angle, offset, and tangent-distance constructors retain their branch from
geometry. Their inputs require a current verified owner seed. They do not add
another geometry algorithm. Radius and diameter accept circles and arcs.
Offset and tangent-distance inputs retain the linear-diameter and near/far
options. Shared arc contact continues through the existing contact mechanism.

Tests cover the 24 existing dimensional forms, measured quantities, parameter
categories, invalid operands, zero and duplicate constraint IDs, and atomic
native append failures.

## Permanent names

Each sketch has a permanent next ordinal for generated parameter names, starting
at `d0`. Allocation skips names already present in the document parameter table
and consumes each skipped ordinal. Reservation happens before source parsing.
The counters belong to document history's reservation owner, outside recipe
snapshots. Failed commands, aborted groups, undo, and deleted sketches do not
release them. A reservation without a recipe change still makes a saved document
dirty through the existing reservation comparison.

Go's `DimensionConstraints.nextName` increments a per-sketch counter. Go restore
recreates dimensions through the constructors; its saved `SketchData.Seq` is the
global creation stamp, not that dimension-name counter. Rust explicitly stores
`dimensionNameOrdinals` beside the schema-3 placed-sketch reservation marks.
This mapping contains sketch IDs and next ordinals, including retired sketches.
Save, reopen, templates, and copies carry the mapping with the owner.

Old records without the mapping bootstrap from surviving unrenamed dimension
parameter names of the canonical form `dN`. Renamed parameters remain collision
checks but do not set the generated-name counter. Old files cannot recover names
that they did not save. An explicit mapping must cover live generated names,
use reserved nonzero sketch namespaces, and retain every previous nonzero mark.
Restore rejects null, non-map, negative, invalid, and short marks. Exhaustion
returns an error and does not wrap the counter.

Tests cover independent namespaces, collisions, renamed numeric names,
exhaustion, legacy bootstrap, malformed storage, retired marks, failed commands,
abort, undo/redo, save/reopen, copies, and refusal before document registration.

## Remaining creation work

The native inputs and name owner are foundations. The application does not yet
implement `client.dimension().add` or its creation reply. The next changes must
create a fresh owned Model scalar and permanent constraint on one isolated
candidate, retain the authored formula, apply reference and annotation options
before the first solve, and prepare measured value and DOF before adoption.
Both part and assembly hosts must use the released API-owned DTOs and constants.
Ellipse-radius and offset-spline dimensions still need native geometry support;
they remain required R3 work.
