# Datum API routing — partial R3

Tracking: [#3741](https://github.com/Oblikovati/Oblikovati/issues/3741).
The boundaries remain ADR-0004 (one authored program) and ADR-0005 (identity).
This document records implementation evidence, not a replacement ADR.

## Supported commands and queries

The file-owned Rust host implements the existing API point and axis create/list
methods. It imports all method constants, DTOs, and constructor kinds from
Oblikovati.API. Position and associative points, checked line axes, and directed axes through
current point references use the same ordered datum program as work planes.
Each command prepares its definition before ID allocation and encodes its reply
inside the candidate edit, before the session adopts that edit. Command groups,
commit veto, undo, redo, and permanent reservations use the existing owner.

Lists include the immutable origin center and three origin axes. Construction
filtering preserves each row's index in the full collection. Queries evaluate
current definitions and expose local health reasons. Failed, suppressed, blocked,
or cutoff geometry has absent coordinate fields; it cannot expose an older
result. A well-formed axis through coincident points is stored as an unhealthy
feature. Invalid coordinate shapes, zero line directions, missing references,
and unsupported constructors decline before an edit is adopted.

The current scope is an active part document. Nonempty occurrence paths and
unported constructor kinds produce named capability declines. Other relational points,
plane-intersection axes, assembly occurrence geometry, and native rendering
remain open.

## Permanent references and visibility

Authored datum references use `datum/v1/<positive canonical FeatureId>` in the
local document namespace. Encoding re-resolves the reference and verifies the
intended feature. Two-point inputs additionally verify the referenced source is
a point. Collection indices are display positions, not identity. Legacy ordinal
keys are not accepted. These references are model datum keys; they do not claim
Martensite B-rep naming or complete the pending product reference-key contract.

Point visibility is independent of construction status. Replacing a definition
retains visibility. A hidden datum requires program version 5. Visible records
omit the new field and restore it as true, preserving earlier record layouts;
a downgraded program containing hidden metadata is rejected. There is no new
public visibility setter in this change.

Associative point sources require program version 6 and source wrapper version 4.
See [association evidence](implementation-associated-points.md).

## Validation and released SDK

Tests use the canonical Rust typed client for default lists, creation, and list
options. They cover origin rows, shared permanent IDs, construction indices,
current driver changes, unhealthy replies, malformed requests, local part scope,
aborted groups, veto, undo/redo, save/reopen, copies, and hidden metadata storage.
Compatibility tests retain empty bytes, JSON null, and object requests.

[API #297](https://github.com/Oblikovati/Oblikovati.API/issues/297) added typed list
options in both languages, generated Rust parity tests, and tested wiki examples.
The owner published [v0.156.0](https://github.com/Oblikovati/Oblikovati.API/releases/tag/v0.156.0)
at `1d8c5ea4441873d2851273a880a90de5d68e0b24` after its CI, wiki, and release
checks passed. The application dependency declarations, lockfile, and release
manifest pin that exact source. The downloaded crate SHA-256 matches GitHub's
asset digest: `ccfdc2ce0add38c865580015c29a4b04f69276fb9e65419008e8fd70b0de17bf`.
[Application #3742](https://github.com/Oblikovati/Oblikovati/issues/3742) records
this adoption. No add-in or host test needs a raw JSON workaround for list options.

These model, file, and API tests do not establish kernel or viewport acceptance.
The application kernel pin remains empty until published packages are verified.
