# Rust planar sketch recipes

Tracking: [#3716](https://github.com/Oblikovati/Oblikovati/issues/3716).
This prepares the application-owned sketch model for document persistence under
ADR-0003, ADR-0005 and ADR-0006.

## Authored state

`Sketch2D::recipe()` copies points, lines, circles, arcs and authored constraints
in their existing collection order. `SketchRecipe` schema version 1 stores stable
entity/constraint IDs, coordinates, independent circle radii, shared references,
arc direction, parameter IDs and retained dimension/contact branches. Arc radius
is derived; there is no stored radius variable for an arc.

The recipe excludes dual numbers, solver columns/results, projections and arc
circularity rows. Structural equations are rebuilt by the existing compiler.
No storage codec owns a second declaration of model entities or constraints.
Enum names use snake case; an independent fixed JSON fixture gates the names,
field order and scalar ID representation. The enclosing document supplies units,
sketch identity and placement. This recipe is not a public API wire contract,
a legacy Go recipe importer, or an entire document format.

## Restore and numeric fidelity

`Sketch2D::from_recipe()` requires the supported schema and restores into a fresh
owner using the same validated add methods as an authored edit. Missing references,
duplicate/zero identities, nonfinite coordinates, invalid circle radii and bad
shared-contact shape are refused before a sketch can be returned. An error cannot
partly replace an existing owner. Unknown record fields, missing required fields
and unknown enum variants are refused during decoding.

Draft lines and arcs keep the same validity boundary as authored edits: coincident
endpoints and unfinished arc circularity can be saved. A restored draft is not
an accepted profile or solved feature. Healthy parameter targets, equation rank,
finite arc bounds and kernel acceptance remain later gates. Parameter references
are retained without resolving them during standalone geometry restore.

The JSON dependency enables correctly rounded float parsing. Tests require exact
coordinate bits after encode/decode, including negative zero, the smallest
positive subnormal, large finite values and ordinary decimals. Working-unit values
are not converted by the sketch recipe. Schema changes must have an explicit
version and migration; unrecognized versions cannot be guessed or repaired.

## Evidence and remaining integration

Tests round-trip every current authored relation with no derived solver records,
retain shared endpoint IDs and direction, and compare deterministic output with a
fixed schema fixture. Malformed recipe tests cover schema, references, identity,
values and retained contact shape. A restored parameter-driven rectangle solves
with zero DOF, changes width, solves again and round-trips the solved candidate.
An empty sketch and editable degenerate draft also round-trip.

Document snapshot/history ownership, document-level sketch identity/placement,
package encoding, legacy import, save/reopen session tests and UI/API commands
remain open. The new recipe alone does not add sketches to a document file.

## Placement frames

Tracking: [#3717](https://github.com/Oblikovati/Oblikovati/issues/3717).
`SketchPlane` now supplies the static placement prerequisite for a document-owned
sketch. It retains a finite origin and authored unit in-plane axes, checks their
unit and angular deviations with explicit versioned metric options, and derives
a right-handed unit normal. The default metric bounds come from Go's
`math.DefaultTolerance` at the fixed migration baseline; they are not kernel
modelling or approximation tolerances. Zero bounds permit strict checks.

XY, XZ and YZ frames use their exact standard basis. `to_model` preserves Go's
ordered origin/axis translations, and `to_sketch` projects the model displacement
onto the saved axes. Input, displacement and result coordinates must be finite.
No saved axis is normalized, rotated or repaired. The angular check measures the
normalized axis dot so loose unit-deviation bounds cannot accept parallel axes
as perpendicular. An undefined normal is always refused.

`SketchPlaneRecipe` stores only origin and axes; restoration repeats validation
under the supplied frame options. Four Go fixtures cover standard and rotated/
translated placement, projection and normal. Independent model coordinates,
out-of-plane projection, exact saved-axis bits, malformed recipes and invalid
options/frames have tests. Coordinate scale checks cover `1e-200` through `1e200`;
nonfinite queries and arithmetic overflow are named failures. This does not yet
implement parameter-driven work planes, face hosting, document sketch collections,
or a saved document placement section.

## Document-owned placed collections

Tracking: [#3718](https://github.com/Oblikovati/Oblikovati/issues/3718).
`DocumentSketches` now owns ordered `PlacedSketch` records with a typed saved
sketch ID, authored name, validated static placement and planar geometry. Add,
remove, rename, geometry replacement and plane replacement are explicit edit
boundaries. A failed edit does not partly change the collection. Duplicate
explicit names remain allowed, matching Go's `AddNamed`; automatic names are
not implemented by this collection.

Entity IDs must have one owner across the document's sketches. The existing
UUIDv5 `sketchEntity:<id>` name has no sketch ID, so accepting the same entity ID
in another sketch would create an ambiguous persistent reference. Shared
endpoints inside one sketch still reference one owned point. Sketch IDs retain
their separate `sketch:<id>` namespace, and authored constraint IDs remain local
to their sketch. The collection allocates no identity; a permanent external
allocator supplies reservations. Live sketch/entity ID lists support those
separate allocation owners. Neither removing a record nor a failed add changes
an allocator supplied by the caller.

Collection recipe version 1 stores placement options as well as axes, names,
IDs and the geometry recipe. `SketchPlane` now retains its validated option
values for that capture. Restore validates the complete collection into a fresh
owner and refuses schema, geometry, frame and global entity-identity errors.
No solver result, normal cache or shared parameter table is duplicated in these
records; solves receive the document's parameter table explicitly.

Tests cover deterministic recipe round trips, original order, names, shared and
separate identity namespaces, custom frame options, whole-collection atomic
failure, geometry and placement edits, removal, external reservation behavior,
malformed restore and UUID continuity/separation. A restored placed sketch edits
its dimension through zero and back, retains zero DOF, then maps the solved point
into the saved XZ placement. Document snapshot/history integration, permanent
reservation storage, package encoding, legacy import and session save/reopen
remain the next integration work; this collection does not itself write files.

## Package codec and reservation storage

Tracking: [#3719](https://github.com/Oblikovati/Oblikovati/issues/3719).
The persistence boundary now stores `model.planarSketches` in document schema 3.
The section contains the versioned collection `recipe`, `sketchHighWater`, and
`entityHighWater`. `restore_sketches()` restores the complete validated collection
and two separate permanent `RecipeIdAllocator` owners. A saved mark must cover
every live ID; a later capture cannot decrease either previously stored mark.
Deleting every sketch still writes the reservation marks so a reopened allocator
does not reuse deleted identities. Maximum `u64` marks restore and report
exhaustion on the next allocation.

`with_sketches()` produces a candidate package, preserving other model and root
sections. The added sketch recipe participates in the existing model digest.
Missing Rust sketch sections mean an empty collection with zero reservations;
a new empty document does not gain an unnecessary section. An explicit Rust
section must use schema 3 and its exact supported shape. Unknown fields, invalid
recipe versions, bad references/identities and malformed marks are refused.

Nonempty legacy Go `model.sketches` requires an explicit importer before this
codec can restore or replace sketch state. The raw package envelope still retains
that section, but it is not interpreted as an empty Rust collection. Empty/null
legacy collections are accepted as empty. This prevents a future snapshot save
from silently replacing unported legacy geometry.

Eight codec tests cover native YAML round trips, an independent fixed schema
fixture, circular dimensions and retained shared contacts, custom placement
options, exact coordinate bits, unrelated section retention, digest changes,
permanent deleted-ID reservations, mark underflow/decrease/exhaustion, malformed
sections and legacy refusal. Capture and restore errors leave the original
package unchanged. The codec has no filesystem I/O. Document snapshot/history
ownership and session atomic save/reopen remain the next integration layer;
this change does not yet attach these recipes to a live file workspace.
