# Authored work-plane normal flips

Tracking: [#3748](https://github.com/Oblikovati/Oblikovati/issues/3748).
The single datum owner and permanent reference boundary remain ADR-0004 and
ADR-0005. This is application frame and recipe evidence, not kernel or native
viewport acceptance.

## Evaluation and source

The canonical typed `workPlanes.flipNormal` command toggles a saved orientation
flag on a placed plane. It retains the feature key, definition, scalar ownership,
and dependency footprint. One replay path evaluates the current definition, then
applies the orientation before publishing its result to dependent datums.

The frame reverses its Y axis, as the Go #1851 implementation does. It leaves
the origin and X axis unchanged and retains a right-handed validated basis.
Dependent offsets use the new normal with their unchanged signed distance.
Hosted sketches use the current frame and retain their local geometry. Two flips
restore the bit-exact authored frame, including signed-zero axis components.
No display facets, cached coordinates, or second parameter graph are involved.

Failed or suppressed definitions retain their source and flip flag, expose no
stale geometry, and apply the flag when evaluation recovers. Replacing a plane's
definition retains its orientation. A point or axis cannot acquire plane-only
orientation metadata: recipe restore/save, history construction, candidate edits,
and refresh adoption all reject it with a typed cause.

## Storage and command ownership

A true `normal_flipped` field requires datum program version 8. False is omitted,
so ordinary sources retain their prior minimum version and layout. Old program
versions with a true flip are refused. Identity reservations are unchanged.

The command uses the existing file-owned transaction and encodes its reply before
candidate adoption. Origin planes are immutable. Full collection indices include
hidden construction planes. Undo/redo, group abort, commit veto, copy, and reopen
use the existing document history and persistence boundaries.

## Linux evidence and remaining scope

Model tests check frame orientation, exact involution, dependent offsets, source
and read retention, version refusal, non-plane metadata rejection, definition
replacement, history adoption, and hosted sketch projection. Typed API tests
check identity, scalar ownership, collection indices, undo/redo, abort, veto,
copy/reopen, and failed-driver recovery. Package tests preserve the owned formula
and orientation and reject flip metadata under an older program version.

The Go `TestFlipNormalReversesAndPersists` and `TestFlipNormalRoundTrips` tests
pass at the fixed source baseline. Acceptance also requires the full local Linux
`make gate` before commit and push. Windows/macOS pipelines, kernel pins, and
publication readiness are unchanged. Other work-plane constructors remain open.
[Display rectangle and associativity edits](implementation-plane-display.md) now
use the same metadata validation boundary.
