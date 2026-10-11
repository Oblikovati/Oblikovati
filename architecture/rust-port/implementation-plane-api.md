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
`0.157.0` and commit `8a23d7aee54e018821ce942054649d8e5b8bdb68` in all
consumer declarations, Cargo.lock, and release.toml. The downloaded crate hash
matches GitHub's asset digest:
`a968656690192cf285e3e47aeadd9882da8302d94c3d7182ed7957d5520cd6f1`.

Tests cover origins, normalized extreme directions, invalid frames, stable keys,
construction indices, visibility, current offsets and display units, local
health, undo/redo, file replay, group abort, commit veto, and scope declines.
Go origin and fixed-frame router tests provide the migration comparison.

## Remaining work

Creation accepts fixed frames, [owned offset expressions](implementation-offset-expressions.md),
and [associative plane-and-point sources](implementation-plane-point.md), plus
[three-point planes](implementation-three-point-planes.md) and
[line-and-point planes](implementation-line-point-planes.md), and
[normal-to-curve planes from axes](implementation-normal-axis-planes.md). Offset redefine
retains expressions and applies scalar/reference edits atomically.
[Normal flip](implementation-plane-flip.md) preserves authored orientation through
replay and storage. [Display settings](implementation-plane-display.md) retain
flags and rectangle metadata. Other plane definitions and assembly occurrence
contexts remain open. Unsupported constructors and occurrence
scopes have named declines. The geometry and native viewport acceptance checks
remain open.

The [two-axis constructor](implementation-two-axis-planes.md) retains ordered
live axis drivers and supports atomic reference edits through the released API.
