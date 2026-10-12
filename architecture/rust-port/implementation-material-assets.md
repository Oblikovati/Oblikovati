# Typed material and appearance assets — partial R5/R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
The Go reference is `model/material` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`, under ADR-0010.

The Rust model now owns material and appearance records that implement the
released API `Material` and `Appearance` contracts. Storage and client adapters
can read the canonical `MaterialInfo` and `AppearanceInfo` records. Public
property groups, source values, and symmetry classes are imported from the API;
no duplicated public DTO or renderer-owned material schema is introduced.

Material records retain density in g/cm³, mechanical, thermal, electrical,
magnetic, directional elastic constants and expansion, and appearance identity.
The read contract reports an unset symmetry as isotropic while the saved input
retains its empty class. Appearance records retain all nine OpenPBR groups,
including the four optional surface-vector overrides. These are asset values,
not analytic mass calculations or a claim of OpenPBR rendering compliance.

The common edit path preserves identity and source. Built-in edits are inert,
as in Go. Named copies own all property groups and optional vectors; changing
one cannot modify its source. Failed editable replacements leave the old record
unchanged. Admission refuses empty/NUL identities, unknown origins and symmetry
classes, negative density, and nonfinite nested API values. The API serializer
provides the numeric finiteness checks without a second field list. This layer
does not certify stiffness, directional strength, or complete simulation input.
Those checks remain required before an analysis consumer accepts an asset.

`AssetCollection` owns snapshots by exact ID and emits them in stable ID order.
`ScopedAssets` resolves document, project, then built-in definitions. Missing
IDs are named errors rather than silent substitution. A document snapshot stays
unchanged after a project definition is replaced and resolves after the source
project is absent. These tests use canonical API record round trips; document
package integration and project filesystem libraries remain open.

The fixed Go material suite passed locally. Rust tests exercise every contract
accessor and property group, optional vector ownership, editable/read-only
sources, atomic failure, symmetry normalization, API round trips, scope
precedence, missing identity, deterministic replacement, and project-independent
snapshot reads. Run `cargo test -p oblikovati-model --test assets`, then
`make gate` for full local Linux checks.

Catalog seeding with original source citations, library revision tracking,
assignment references and occurrence overrides, document recipe attachment,
wire routing, asset editors, legacy appearance conversion, rendering, and
analytic mass integration remain open. The baseline assignment regression
for Go issue #1103 puts explicit part appearance before a material appearance;
ADR-0010 currently describes a different order. Resolve that evidence with an
explicit superseding decision before implementing the assignment resolver;
do not change the accepted historical record in place.

No kernel code or codec is added. Tests are limited to this Linux host, with
no push or PR and no Windows/macOS acceptance claim.

## Local validation receipt

On 2026-10-12, `make gate` passed for all 13 workspace crates with llvmpipe and
the local Vulkan validation layer. Model production line coverage was 98.81%;
every crate exceeded 80%. Repository duplication was 0.62%. The gate included
formatting, clippy, build, tests, doctests, documentation, SPDX, workflow and
architecture checks. After the final test-fixture refactor, all 15 asset tests
and workspace clippy passed again. The existing AMD native presentation fault
remains open; this receipt does not claim that it is fixed.
