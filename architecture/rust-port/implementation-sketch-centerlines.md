# Sketch centerline roles

[Issue #3777](https://github.com/Oblikovati/Oblikovati/issues/3777) supplies the
centerline prerequisite for inferred line creation under
[issue #3775](https://github.com/Oblikovati/Oblikovati/issues/3775).

The canonical API already defines `AddSketchEntityArgs.centerline`. A centerline
is a reference line and an implicit axis candidate for revolve. The typed Rust
line constructor now carries this flag. `Sketch2D` stores the role separately
from explicit construction, matching the Go `entityBase` behavior at the
[source baseline](source-map.md). Clearing either role preserves the other.
Effective construction and profile candidate selection exclude centerlines.
Only owned lines can carry this role; identities and defining points do not change.

`centerline_ids` returns authored order. `centerline_axis` returns no candidate
when none is marked, the owned line when exactly one has distinct endpoints, or
a named error for ambiguity or coincident endpoints. It returns analytic line
references; the later feature owner must transform their points through the sketch
plane. Axis lookup does not infer an axis from display facets or proximity.

Planar recipe version 4 records ordered `centerlines`, complete entity order,
and independent explicit construction metadata. The outer native document remains
schema 3. Versions 1-3 remain readable without centerlines. A recipe with axis
roles upgrades to version 4; clearing them restores the version required by its
remaining content. Duplicate, out-of-order, non-line, foreign, and cross-version
metadata are refused. Recipe comparison includes the role, so undo and dirty
checks can detect it. No new numeric equation is introduced by the flag.

Model tests cover role combinations, idempotence, profile candidates, ownership,
unique/ambiguous/degenerate axes, signed-zero coincidence, finite scale extremes,
versioned round trips, malformed metadata, legacy restore, and equal solver
coordinates, termination, and freedom with and without an axis role. Persistence
tests check the native boundary and independent roles after restore. Named file
owners exercise typed creation with shared endpoints, current DOF-zero evidence,
undo/redo, group abort, rejected reply, save/reopen, and copy. Reserved curve IDs
remain consumed when creation is undone or rejected.

The Go centerline/profile/mirror corpus passes at the fixed source baseline. It
supports the reference behavior; Rust tests here cover the role, axis references,
and profile candidate exclusion. They do not establish profile arrangement,
mirror, revolve, or released kernel acceptance. Native viewport evidence remains
open. API line creation and inferred relation application remain #3775.
