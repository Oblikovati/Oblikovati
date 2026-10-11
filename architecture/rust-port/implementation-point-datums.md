# Position points and two-point axes — partial R3

Tracking: [#3740](https://github.com/Oblikovati/Oblikovati/issues/3740).
Go sources: `model/feature/work_axis_point.go` and `serialize_work_axis.go` at
migration baseline `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
This extends ADR-0004's existing datum program. It adds no separate point owner,
feature namespace, dependency graph, replay controller or storage section.

Position points store three constant working-unit lengths or stable parameter
IDs. The model reads current parameter model values. It does not save closures
or evaluated coordinates. Plane offsets and point coordinates use one
`DatumLength` resolver, finite-length policy and exact source comparison.
The old plane scalar name remains an alias, and its diagnostics are preserved.
Invalid point coordinates identify their component and retain the scalar cause.

An axis through two points stores ordered typed references to the origin or
current datum points. The first point remains its origin. Direction is derived
by the existing checked line constructor. Coincident, unavailable and wrong-kind
references decline before any line result is published. Failed points block
dependent axes; suppressed points give unavailable-driver declines. Independent
datums continue. Owned sketch orientation consumes the current resulting axis.

Normal finite differences retain the original line constructor arithmetic and
exact source bits. When subtraction overflows, scaling bounds the differences
before normalization. Finite component differences are retained independently,
so adjacent large coordinates do not lose their small relative displacement.
Origins are never scaled or moved in the result. These are model datum values,
not a B-rep intersector or a new kernel topology predicate.

One typed current result table now holds plane, axis and point views. Projection
checks the selected kind. No view exposes a previous evaluation's geometry.
Position points and two-point axes require program version 4 and definition
wrapper version 3. Plane and line-only programs retain earlier versions.
The existing versioned `model.workPlanes` key remains the compatibility anchor.
Older program versions cannot adopt these definitions.

Tests cover current parameter updates, directed references and exact reads,
wrong kinds, failed coordinates, coincident points, suppression, cutoff, source
version refusal and signed coordinate bits. Numerical tests cover extreme and
subnormal values, adjacent large-coordinate increments and 256 point pairs.
The finite cases reproduce checked explicit lines bit for bit. The Go two-point
axis fixture is retained as a hard assertion, and its baseline test runs on Linux.
File-session tests cover undo/redo, save, reopen, copies, dependent sketch
placement and construction-chain deletion without reuse of permanent IDs.

Plane-intersection axes, relational point definitions, coordinate systems,
and edge references remain open. Public creation/list routing and hidden metadata
version 5 are recorded in [API evidence](implementation-datum-api.md). Native
rendering and kernel acceptance are not established by these model and file tests.
