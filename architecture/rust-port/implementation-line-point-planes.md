# Associative line-and-point work planes

Tracking: [#3755](https://github.com/Oblikovati/Oblikovati/issues/3755).
Source: Go `lineAndPointPlaneDef` and its redefine slots at revision `aa8297d1`.
The current datum owner and persistent identity remain ADR-0004 and ADR-0005.

## Current inputs and shared frames

`WorkPlaneLinePoint` stores one line reference and one point reference. Replay
resolves the current complete line and point through the existing result table.
The line origin supplies the plane origin. Its checked unit direction supplies
the X axis unchanged. The line-to-point displacement determines the normal and
Y axis through the shared frame calculation used by three-point planes.

The construction never adds a unit direction to a large origin to invent a
second point; doing so could erase that direction through rounding. It keeps
the original origin and direction bits. Finite displacement subtraction,
overflow handling, direction normalization, and frame checks remain shared.
No tessellation or second geometry owner determines the result.

`WorkAxisReference` is the single origin/feature axis reference type.
`SketchAxisReference` remains its source-compatible export, so callers and
serialized variant names do not change. Plane inputs resolve the complete
line; sketch orientation continues to request only its current direction.
Origin axes add no feature reads. User axes and points attribute their permanent
IDs, and the existing graph validates dependency order.

Collinear or coincident line/point inputs have local failed health and no current
frame. Wrong kinds, unavailable drivers, suppression, and invalid dependency
order use the existing failure path. Valid current inputs recover the plane.
Authored normal flips and display metadata apply through the same replay owner.

The source wrapper requires version 7 and the mixed datum program requires
version 12. Earlier definitions retain their minimum versions. Stored sources
retain references, not evaluated frames. Unknown fields and older reader
versions cannot silently lose the definition.

## Canonical API and Linux evidence

The released API owns `WORK_PLANE_LINE_AND_POINT`. Creation requires exactly
two local references: axis then point. Listings report current geometry and
health, plus `Line` and `Through point` slots with no scalar slots. Redefine
accepts either or both unique reference slots, sharing one candidate and history
step with display edits. Bad kinds, bad references, forward dependencies,
duplicate slots, and scalar edits refuse the whole candidate. Existing plane
constructors share reference-only validation and slot metadata construction.

Tests cover all origin axes, current user line origins/directions, parameter
changes, preserved direction bits, extreme origins and near-collinear scales,
degeneracy and recovery, suppression, wrong kinds, dependency reads and declines,
versioned source equality, typed creation/listing, atomic repicks, undo/redo,
reopen, copies, and normal flips. Existing sketch orientation and three-point
plane tests remain required. Go `TestAddByLineAndPointBuildsPlane`,
`TestAddByLineAndPointDegenerate`, and `TestWorkPlaneRedefineSlotsPerKind` pass
at the fixed source revision.

The acceptance command is full Linux `make gate`. Other plane constructors,
assembly occurrences, kernel B-rep replay, and native viewport evidence remain
open. No kernel checkout or Windows/macOS pipeline is changed.
