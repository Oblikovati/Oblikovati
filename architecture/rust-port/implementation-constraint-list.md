# Geometric constraint list API

Tracking: [#3780](https://github.com/Oblikovati/Oblikovati/issues/3780).
This follows the [inferred line API](implementation-line-api.md) and makes its
returned geometric indices inspectable through the canonical typed client.

## Read-only projection

`sketch.constraints` imports the API method constant, `ListConstraintsResult`,
and `ConstraintInfo`. Both file hosts reach the same immutable query. The active
owner must be a part or assembly and the sketch index must identify an ordered
sketch. Invalid owners, indices, or JSON return named errors.

Rows follow `Sketch2D::geometric_constraints` in stored order. The shared model
classification excludes dimensional relations; projection and arc equations are
numerical sources and are absent from this authored list. The query does not
solve, read tessellation, change focus, consume IDs, or alter document history,
dirty state, coordinates, or numerical freshness. Failed dimension health does
not prevent inspection of valid authored relations.

The mapping covers all 18 current geometric forms: coincidence, two-point
horizontal/vertical alignment, single-line horizontal/vertical, point fix,
line/arc midpoint, point-on-line, four line relations, point-on-circular-curve,
concentric, equal radius, and both tangent representations. It uses API-owned
kind constants and actual authored operands. Point/line/circular pairs preserve
constructor order; a single-line relation names its line, rather than expanding
its endpoints. Both tangent representations use the canonical `tangent` kind.

Each row has a checked nonnegative i64 geometric index. Inferred-line replies
and list rows share one conversion helper. Constraint identity stays separate
from position: earlier removals can change the enumeration index without
changing the surviving constraint's identity. Current authored relations are
user-owned and therefore deletable; structural sources are excluded. Custom
client/name metadata is absent because custom relations are not yet ported.
Unknown future representations decline at projection rather than returning an
invented `unknown` row. Direct dimensional projection is also refused by name.
An empty result encodes `constraints` as an empty array.

## Reference and validation

Go reference commit: `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
The mapping follows `addin/router/sketch_enumerate.go` and ordered operands in
`model/sketch/constraint_kind.go`. Independent reference tests
`TestConstraintKindSelfDescription`, `TestSketchEnumerateShapes`, and
`TestAddEntityReportsAppliedInference` pass. The first checks model kind/operand
self-description; the latter tests exercise wire enumeration and real inferred
line indices. They do not prove support for Go relations absent from Rust.

Typed-client tests assert every mapped kind and operand order for both part and
assembly owners. The mixed corpus contains dimensions before, between, and
after geometric relations, including an unavailable parameter. Tests check
failed health inspection without mutation, current parameter changes, empty
origin sources, a solved arc with structural equations, canonical inferred-line
indices, malformed requests and owner/index declines, undo/redo, save/reopen,
and copy. Private tests check dimensional projection refusal and index bounds.

The full Linux gate covers all present Rust workspace members, lint,
documentation, SPDX, coverage, and duplication. This step does not implement
constraint authoring/deletion through the API, the remaining Go relation
vocabulary, interactive drawing, profiles, a native viewport, or kernel
integration. Those remain part of the application port. Windows/macOS remain
deferred.
