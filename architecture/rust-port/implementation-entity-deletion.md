# Native sketch point ownership and entity deletion — partial R3

Decision: [ADR-0017](../decisions/ADR-0017-sketch-point-ownership.md).
Parent work: [#3789](https://github.com/Oblikovati/Oblikovati/issues/3789).
Go source revision: `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
References: `model/sketch/edit_delete.go`, `edit_delete_test.go` (#1232),
`sketch_entities.go`, and `blocks_create.go::dropConstraintsOnVars`.

## Owned entities and solver points

`add_point` retains an independent point marker. `add_defining_point` adopts a
shared solver point without a standalone entity row. Both keep permanent IDs,
coordinates and solver columns. `entity_ids` records visible authored entities;
`owned_entity_ids` also includes defining points for document reservations,
collision checks and round-trip reference names. A marker can define curves and
remains when its last curve is removed. Defining points are not reconstructed
from geometric proximity or inferred from current curve incidence.

Planar schema 12 records explicit defining-point IDs in authored point order,
along with all earlier schema metadata. Invalid, repeated, reordered and curve
IDs in this list are refused. Versions 1-11 cannot contain it and retain their
existing point ownership. Save/restore preserves signed-zero coordinates and
visible order. Native part and assembly packages reserve the highest defining
point ID even when all visible entity IDs are lower.

## One deletion path

`delete_entities` validates the entire typed selection and the seed's relations
before mutation. A single dependency closure covers points, lines, circles,
arcs, ellipses, fit/control parents and offsets. Removing a point retires curves
that need it; removing a parent retires derived offsets. Defining points that no
survivor needs are pruned. Unselected independent markers remain. Numeric ID
order and working scale do not select a different path.

The candidate uses the existing relation validator to find references that lost
ownership. The seed was validated and geometry values did not change, so these
failures identify missing ownership. Existing relation deletion removes text
placement, reference role and drive limits. Construction and centerline roles
are retained only for survivors. A complete recipe restore validates the candidate
before assignment. Returned retired relations retain their parameter identities
for the later document-owned cascade. No parameters are deleted by the sketch.

Unlike Go's raw entity-pointer helper, this typed method refuses any unknown ID
atomically. Duplicate selections retire an identity once. Empty selection is a
no-op. Survivors keep their identities, authored order and coordinate bits.

## Evidence and remaining work

The shared primitive factory now adopts new line, circle, ellipse and arc points
as defining points. Explicit point creation remains a standalone marker. Reused
owned points keep their existing role and identity. The existing reservation
report still includes every created point and curve, while public entity rows
include one row per primitive. The part's optional projected origin retains its
existing marker; assembly creation keeps its existing no-origin behavior.

Typed public-client acceptance checks all five primitive kinds in both document
owners, authored row indices, exact reserved IDs, hidden-point key resolution,
schema 12, save/reopen and copy namespaces. Existing session tests check shared
point reuse, successful solve evidence, source/identity retention, rejected replies,
veto, abort, undo and redo. The earlier test that equated entity enumeration with
every owned coordinate record now checks the two namespaces separately.

Model tests cover every native curve kind, shared points, standalone markers,
dependent offsets, orphan points and bound relations, dimension metadata cleanup,
invalid/empty/duplicate selections, reversed numeric dependency order, three
working scales, native solver viability, schema refusal/round-trip, collision
checks and reference-key retirement. Package tests cover both document owners,
hidden high-water IDs and byte-identical re-encoding. Run `make gate` for the full
Linux workspace suite.

This covers the native model boundary and the five current public primitive
factories. Spline and other pending curve factories still need defining-point
construction. Canonical entity deletion, document model-parameter
cascade, annotations, history/veto/abort, copy/reopen/live editor acceptance and
feature-consumer guards are incomplete. Do not close #3789 or infer general
application deletion acceptance from these native tests.
