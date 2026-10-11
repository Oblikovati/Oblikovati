# Work-plane API routing — partial R3

Tracking: [#3744](https://github.com/Oblikovati/Oblikovati/issues/3744).
The single datum owner remains ADR-0004, and permanent identity remains ADR-0005.
The application still needs kernel-backed geometry and native rendering evidence.

## Supported behavior

The file-owned host now implements canonical work-plane creation and listing.
Fixed-frame creation checks three finite origin and direction components,
normalizes direction magnitudes with the shared model routine, then validates a
right-handed orthogonal frame. It preserves the source origin. Invalid frames
are refused before permanent allocation. Visibility and construction are distinct
saved metadata, and replies are encoded before the session adopts the candidate.

Origin planes remain immutable, grounded, and hidden by default. User fixed and
offset planes come from the same ordered datum program as points and axes. One
list pipeline filters by source kind, preserves full collection indices through
construction filtering, and evaluates current geometry once. References use the
existing verified document-local datum scheme, not collection ordinals.

Plane listings include current origins, normals, visibility, construction state,
and health. Failed, suppressed, or cutoff geometry has absent coordinate fields
and a local health reason. Offset definitions additionally report the base-plane
slot and current distance in the document's preferred units. An unavailable
scalar is not replaced with a made-up value, and display overflow is a named
error. No display mesh is used for model geometry.

## Released API and checks

[API #298](https://github.com/Oblikovati/Oblikovati.API/issues/298) adds work-plane
list options to both SDKs and extends the generated checks and tested wiki
examples. [API v0.157.0](https://github.com/Oblikovati/Oblikovati.API/releases/tag/v0.157.0)
passed owner CI, wiki, release, and archive checks. The application pins version
`0.157.0` and commit `8a23d7aee54e018821ce942054649d8e5b8bdb68` in all four
consumer declarations, Cargo.lock, and release.toml. The downloaded crate hash
matches GitHub's asset digest:
`a968656690192cf285e3e47aeadd9882da8302d94c3d7182ed7957d5520cd6f1`.

Tests cover origins, normalized extreme directions, invalid frames, stable keys,
construction indices, visibility, current offsets and display units, local
health, undo/redo, file replay, group abort, commit veto, and scope declines.
Go origin and fixed-frame router tests provide the migration comparison.

## Remaining work

Creation currently accepts the fixed-frame constructor. Offset expression
creation must retain an owned expression and its parameter dependencies; it must
not evaluate once and store a constant. Other plane definitions, redefine,
normal flip, assembly occurrence contexts, and display rectangle metadata remain
open. Unsupported constructors and occurrence scopes have named declines.
Listing an existing offset source does not establish its create/edit API.

[Owned scalar expressions](https://github.com/Oblikovati/Oblikovati/issues/3745)
track offset creation and edit prerequisites for this same datum owner.

[Stored scalar ownership](implementation-scalar-ownership.md) records the first
completed prerequisite; offset create/redefine acceptance remains open.
