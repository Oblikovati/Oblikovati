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
