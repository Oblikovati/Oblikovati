# Two-axis work planes

Tracking: [#3757](https://github.com/Oblikovati/Oblikovati/issues/3757).
Source: Go `twoLinesPlaneDef` and its redefine slots at revision `aa8297d1`.
ADR-0004 and ADR-0005 retain the datum and identity boundaries.

## Source and frame ownership

`ThroughTwoAxes` retains two ordered `WorkAxisReference` values in the single
mixed datum program. Replay resolves both current drivers in order. The first
axis supplies origin and X. The normalized cross product of the two directions
supplies the normal; normal crossed with X supplies Y. The existing checked
frame arithmetic is shared with the line-and-point and three-point constructors.
No coordinates are moved and no second geometry evaluator is introduced.

The Go constructor does not classify line coplanarity. The second axis origin
is not a placement input, so skew lines retain the same direction-defined
frame as the Go baseline. This is a construction frame, not evidence of a
B-rep intersection or containment operation. Parallel or opposite directions
cannot define a frame and produce local failed health. Suppressed and failed
drivers provide no current geometry; a successful later replay recovers it.

Origin axes require no feature reads. Owned axes attribute both dependencies,
with repeated references deduplicated by the existing read footprint. Source
edits refuse self and forward dependencies before adoption. Authored normal
flips remain metadata and survive driver changes.

The source wrapper requires version 9 and the datum program requires version
14. Older constructors retain their minimum versions. Exact source ordering,
permanent identity, metadata, and reservations survive storage round trips.
Older versions cannot accept the new constructor silently.

## API and acceptance evidence

The released API owns `WORK_PLANE_TWO_LINES`. Creation takes exactly two axis
references. Listing reports `Line 1` and `Line 2` reference slots, both of kind
axis, with no scalar slots. Either or both unique reference slots can be edited.
Repicks and display changes use one candidate and one history step. Invalid
reference kinds, missing references, duplicate or out-of-range slots, scalar
edits, and forward dependencies decline before publication. Unchanged edits
do not create a generation.

Model tests cover both current directions, first-origin edits, skew second
origins, ordering and flips, local failure and recovery, read attribution,
invalid dependencies and kinds, versions, exact recipes, malformed sources,
signed origin bits, invalid inputs, and generated small-angle directions.
Typed API tests cover creation, metadata, user axes, parallel failure recovery,
atomic edits, undo/redo, no-op edits, selective repicks, save/reopen, and copies.

Go `TestTwoLinesWorkPlane`, `TestWorkPlaneRedefineSlotsPerKind`, and
`TestWorkPlaneRedefineClosureOverEveryKind` pass at the fixed source revision.
The full acceptance command is local Linux `make gate`. Kernel B-rep replay,
assembly occurrence inputs, and native viewport acceptance remain open.
