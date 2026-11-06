# Associative three-point work planes

Tracking: [#3754](https://github.com/Oblikovati/Oblikovati/issues/3754).
The datum program, evaluation, and persistent identity boundaries remain
ADR-0004 and ADR-0005. Source: Go `threePointPlaneDef`, `planeThroughPoints`,
and its redefine slots at migration revision `aa8297d1`.

## Current frames and source ownership

A three-point source stores three ordered point references. Replay resolves
their current values through the existing datum result table. The first point
supplies the origin, the first-to-second displacement supplies the X direction,
and their cross product with the first-to-third displacement supplies the
normal. The Y direction completes the right-handed frame. The shared sketch
frame checks validate the result. Reversing the last two references reverses
orientation; the source order is significant even when coordinates coincide.

Direction normalization precedes cross products to avoid overflow at extreme
finite scales. The existing finite-point subtraction routine moved unchanged
from the axis value into the shared frame direction module. Its overflow path
and provenance comments remain. Point coordinates are never moved or perturbed.
Frame construction uses neither tessellation nor a second datum owner.

Repeated or collinear points produce local failed health with no current frame.
The source remains authored, independent features continue, and valid current
drivers or a valid repick recover it. Suppression, cutoff, wrong kinds, and
invalid dependency order use the existing replay and feature graph rules.

The source wrapper requires version 6 and the mixed datum program requires
version 11. Older readers cannot silently ignore the new definition. Earlier
constructors retain their minimum versions. Storage captures references, exact
source order, metadata, and permanent reservations rather than evaluated frames.

## Canonical API and Linux evidence

The released API already defines `WORK_PLANE_THREE_POINTS`. Typed creation
requires exactly three local point references. Listings report current geometry,
health, and ordered `Point 1`, `Point 2`, and `Point 3` reference slots with no
scalar slots. Redefine accepts any set of unique slots 0 through 2. Invalid
slots, duplicate slots, scalar edits, bad kinds, and forward dependencies decline
the whole candidate. Display changes and repicks remain one atomic command.

Tests cover parameter-driven orientation, flips, local failure and recovery,
dependency footprints, versions, exact source comparison, oblique analytic
normals, point containment, generated extreme and near-collinear values, typed
creation and listings, atomic repicks, unchanged edits, undo/redo, group abort,
commit veto, and source preservation through reopen and independent copies.
Go `TestThreePointWorkPlaneAndDegenerate`,
`TestRedefineThreePointPlaneRepicksPoint`, and
`TestWorkPlaneRedefineSlotsPerKind` provide the fixed-source regression checks.

Full Linux `make gate` remains the acceptance command. Other plane constructors,
assembly occurrence scopes, kernel B-rep replay, and native viewport evidence
remain open. No kernel checkout or Windows/macOS pipeline is changed.
