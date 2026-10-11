# Current datum-point projection — partial R3

Tracking: [#3760](https://github.com/Oblikovati/Oblikovati/issues/3760).

`DocumentSnapshot::project_datum_point_into_sketch` resolves a document origin
or a stable datum-point feature into the current placed sketch frame. It uses
one immutable mixed-datum evaluation for the source, host plane, and orientation
axis. The analytic `SketchPlane::to_sketch` operation performs the projection.
No kernel mesh or stored last-good frame supplies an input.

The result contains the source identity, current model and local coordinates,
and an attributed read footprint. The footprint includes the sketch identity,
its host plane and orientation axis, and the point source when authored. Origin
sources add no feature ID. Local sketch dimension parameters do not supply the
projection frame; their reads remain in the sketch geometry footprint. Datum
features retain their own parameter and source dependencies in the shared graph.

This query does not require a solved sketch. Creation must locate its reference
anchor before the solver can ground that anchor. The query does not edit geometry,
create a fix constraint, allocate identities, or change verification stamps.
The caller must keep the source reference when it creates an associative binding.

Missing, suppressed, failed, rolled-back, and wrong-kind sources are errors.
Unavailable hosts and degenerate orientation axes retain placement errors.
Point declines name the sketch and source and retain the typed geometry cause.
Non-finite projection results are refused by the existing coordinate checks.
An independent failed datum does not invalidate an available origin projection.

The source/frame seam ports the analytic calculation used by Go
`model/sketch/projection.go` at baseline
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. It does not port Go's silent
freezing of a lost reference. Source loss must reach the Rust feature health.

Tests cover live parameter, host and axis edits, both orientation signs, origin
projection, read attribution, source and placement failures, numeric overflow,
unchanged recipe/verification state, and 64 generated translated/rotated frames
with off-plane sources. It now serves
[persisted bindings and source-driven solver refresh](implementation-associative-point-projections.md).
The auto-project-origin option and `sketch.create` routing remain required.
Vertex/edge projection needs the released kernel reference catalogue. Native
application and rendering acceptance remain open.
