# Normal-to-curve work planes from axes

Tracking: [#3756](https://github.com/Oblikovati/Oblikovati/issues/3756).
Source: Go `normalToCurvePlaneDef`, `planeFromOriginNormal`, and its redefine
slots at revision `aa8297d1`. The datum and identity owners remain ADR-0004
and ADR-0005.

## Supported source and current geometry

The Go baseline uses work axes as the supported curves for this constructor.
The Rust `NormalToAxis` source retains the same axis-point reference pair used
by line-and-point planes. The variant selects the operation; the source pair
does not store a second frame or introduce a second dependency owner.

Replay resolves the current axis first and the current point second, matching
Go's driver order. The point supplies the origin and the axis direction supplies
the normal. The axis origin has no placement role. The point need not lie on the
axis. Driver changes remain live, and failed or suppressed drivers have no
current plane. Recovery and normal flips use the existing replay owner.

`SketchPlane::from_origin_normal` constructs the orthonormal frame through the
shared direction arithmetic and frame checks. In-plane roll is unconstrained by
this constructor, so it retains the Go conditioned seed rule: cross the normal
with X, except when the absolute X component exceeds 0.9, when Y is used.
The threshold controls roll and conditioning, not modelling tolerance. The
strict boundary, its adjacent values, and signed principal frames are tested.
Coordinates are not perturbed and no display geometry supplies model inputs.

The source wrapper requires version 8 and the mixed datum program requires
version 13. Earlier constructors retain their existing minimum versions.
Sources, metadata, and permanent reservations survive exact storage round trips;
older programs and unknown source fields cannot silently lose the definition.

## API, history, and evidence

The released API owns `WORK_PLANE_NORMAL_TO_CURVE`. Creation accepts exactly
two local references: axis then point. The list reports current geometry and
health with `Curve` and `Point` slots of kinds axis and point, and no scalars.
Parsing and reference revision share one axis-point module with line-and-point
planes. Redefine accepts either or both unique slots and applies repicks and
display changes in one candidate and history step. Wrong kinds, malformed
references, scalar edits, duplicate slots, and forward dependencies decline
before adoption. Unchanged sources create no generation.

Model tests cover live point/axis edits, origin independence, flips, suppression,
wrong kinds, read footprints, invalid dependencies, versions, source equality,
unknown fields, signed origins, invalid normals, seed boundaries, and generated
oblique normals. Typed API tests cover creation and listing, user and origin
axes, atomic edits, undo/redo, no-op edits, selective repicks, reopen, copies,
and retained flips. Existing line-and-point tests remain required for the
shared parser and revision path.

Go `TestNormalToCurveWorkPlane`, `TestWorkPlaneRedefineSlotsPerKind`, and
`TestWorkPlaneRedefineClosureOverEveryKind` pass at the fixed source revision.
The acceptance command is full Linux `make gate`. Nonlinear curves, assembly
occurrence inputs, native display, and kernel B-rep replay remain open. No
kernel checkout or Windows/macOS pipeline is changed.
