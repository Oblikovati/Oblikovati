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

## Owner creation and health

The document owner creates a fresh Model scalar and permanent constraint on an
isolated candidate. `insert_scalar_expression` retains the source, dependencies,
and original bare-input unit. Creation does not freeze the formula. A source
change later recomputes the dimension through normal parameter propagation.
Length inputs use document display units and working scale; angles use the
captured angular unit. Current geometry supplies retained angle, offset, and
contact branches. Tests cover an angle whose stored coordinates have the wrong
turn for its current driving source.

Ownership and finite annotation validation happen before reservations. Malformed
source, self-reference, invalid healthy metric domains, or identity exhaustion
publish neither the parameter nor the relation. Attempted reservations remain
permanent. Reference role and annotation apply before the first solve. A reference
dimension can keep a failed unused formula without constraining its geometry.

Unresolved, incompatible, and failed arithmetic formulas remain authored with
failed parameter health. Strict sketch verification still records failure. DOF
inspection classifies unavailable driving targets and omits those rows only on an
isolated analysis copy, through the existing structural analyzer. Every omitted
row retains its constraint ID, parameter ID, and cause in the owner report. This
inspection neither solves nor changes recipe state; it never substitutes zero
or reports convergence. Healthy invalid geometry or metric domains remain errors.
Later parameter binding repairs the source and triggers normal verification.

Session and workspace commands prepare a projected reply before adoption. Their
normal event, veto, group, recompute, and history boundaries apply. Tests cover all
24 native forms with healthy and unresolved sources, formula propagation,
reference creation without transient conflict, three working scales, angular
units, malformed inputs, source errors, exhaustion, rejected replies, veto,
abort, undo/redo, deletion/recreation, save/reopen, and copied formulas/annotations.

## Remaining creation work

The public automation host now routes `client.dimension().add` through the owner
command for part and assembly documents. It uses the released request, method
constant, and result. Operand ownership and shape, orientation, and annotation
validation precede reservation. Replies convert working lengths to centimetres
and preserve radians, with checked finite values and integer indices before
candidate adoption.

Client tests cover the native constructor vocabulary, reference metadata,
optional annotations, ignored options, invalid operands, live driving formulas,
and undo/redo. Further API acceptance must cover working scales, failed-source
health, veto, save/reopen, and copied documents. Ellipse-radius and offset-spline
dimensions still need native geometry support; they remain required R3 work.
