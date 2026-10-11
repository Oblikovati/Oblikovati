# Associative datum points — partial R3

Tracking: [#3743](https://github.com/Oblikovati/Oblikovati/issues/3743).
The single authored program and permanent identity boundaries remain ADR-0004
and ADR-0005. No second datum graph or kernel algorithm is introduced.

`PointAtReference` ports Go's `pointRefPointDef` and `WorkPoints.AddByPoint` from
`model/feature/work_relational_defs.go` at migration baseline
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. It stores an origin or permanent
point reference and attributes its direct feature dependency. Replay obtains the
current point from the existing result table without arithmetic or coordinate
changes. Chained references preserve coordinate bits, including signed zero.
Suppressed, failed, wrong-kind, or cutoff inputs cannot expose stale coordinates.
Independent datums continue through the same replay controller.

The source definition wrapper is version 4, and a program containing this kind
requires version 6. Older versions are rejected. Other datum kinds retain their
existing minimum versions. Recipes store the reference, not its evaluated point;
exact comparison distinguishes different source identities even when coordinates
coincide. Existing file history and permanent reservations remain the owners.

The host implements the existing `WORK_POINT_ON_POINT` constructor through the
canonical typed `work_points().create` method and `CreateWorkPointArgs`. Exactly
one live point reference is required before allocation. Point listings preserve
the constructor kind, collection index, visibility, construction flag, and local
health. These points can drive directed axes and later associative datums.

Tests cover exact current coordinates, chained reads, independent health,
suppression, cutoff, wrong kinds, source-version refusal, typed creation,
malformed references, current axis consumers, undo/redo, save/reopen, and copies.
The Go point-by-point, unresolved-reference, and relational storage tests pass on
Linux at the migration baseline. Native rendering and kernel-backed feature
acceptance remain open.
