# Parameter-driven line-angle plane model

Tracking: [#3758](https://github.com/Oblikovati/Oblikovati/issues/3758).
API integration: [#3759](https://github.com/Oblikovati/Oblikovati/issues/3759).
Source: Go `linePlaneAnglePlaneDef`, `planeThroughLineAtAngle`, and
`math.Rotation4` at revision `aa8297d1`. ADR-0004 and ADR-0005 retain the
model and identity boundaries.

## Current source and evaluation

`AtAngleAboutAxis` retains the axis reference, base-plane reference, and one
`DatumAngle` source in the common datum program. An angle is a checked finite
radian constant or a stable parameter reference. Length and angle sources
share current model-quantity lookup; each checks its own dimension. Preferred
display units and the document length scale cannot turn a length into an angle.
Parameter failures retain their causes; wrong dimensions retain the offending
quantity and driver in `DatumAngleError`.

Replay resolves the current axis, current base frame, and current angular model
value in that order. The axis origin places the result. The base origin has no
placement role. The base normal is projected perpendicular to the axis, then
rotated with the Go Rodrigues matrix order and normalized. The existing
normal-defined frame constructor supplies the same conditioned roll rule.
Authored normal flips remain metadata and do not overwrite scalar sources.

The projected reference and rotated normal must be perpendicular to the axis
within the existing frame angular policy. This is a frame metric check, not a
B-rep topology predicate. It certifies that projection roundoff cannot create
an arbitrary reference roll when the base normal follows the axis. Undefined
or invalid frames decline locally and provide no current geometry. Failed
or suppressed drivers do not fall back to stored frames. Recovery reads the
current source again.

Read footprints include both owned geometry drivers and the angular parameter.
Origin geometry and constant angles add no driver reads. Self and forward
references cannot enter the program. Source equality preserves scalar bits,
including signed zero, as well as permanent parameter and geometry references.

The source wrapper requires version 10 and the datum program requires version
15. Older constructors retain their minimum versions. Restore validates angular
constants before adoption. Source order, normal flips, and permanent reservations
survive package round trips without evaluated frame caches.

## Evidence and remaining integration

Model tests cover analytic signed rotations, degree-to-radian values, live
parameter and geometry changes, base-origin independence, flips, suppression,
wrong kinds and dimensions, missing parameters, read attribution, versions,
source equality, unknown fields, and generated oblique axes. Rotation checks
verify repeatable normals, line containment, and the signed sine/cosine of the
requested angle against the frame policy.
Storage tests cover exact package restoration and reject nonfinite angular
constants, malformed sources, and unknown constructor fields.

Go `TestLinePlaneAndAngle`, `TestWorkPlaneRedefineSlotsPerKind`, and
`TestWorkPlaneRedefineClosureOverEveryKind` pass at the fixed source revision.
The full acceptance command is local Linux `make gate`.

This item supplies model recipes, replay, and storage. Public API creation,
scalar ownership, listing, and redefine remain tracked by #3759. Unsupported
commands retain named declines. Assembly occurrences, kernel B-rep consumers,
and native display acceptance remain open.
