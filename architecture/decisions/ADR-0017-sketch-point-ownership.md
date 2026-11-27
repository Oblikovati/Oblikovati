# ADR-0017: Standalone sketch entities and defining-point ownership

Date: 2026-10-11. Status: Accepted for the Rust target design.
Boundary: standalone sketch entities, shared solver points, and deletion closure.
Owner: sketch model; document owner for global identities and transactions.
Tracking: R3 and [#3789](https://github.com/Oblikovati/Oblikovati/issues/3789).

Keep a defining point distinct from a standalone point marker. Both have permanent
owned identities and solver coordinates, but only the marker occupies an entity
enumeration row. Curve factories adopt defining points; explicit point commands
adopt markers. A marker can also be shared by curves. This preserves Go
`Sketch.NewPoint` versus `Sketch.Points().Add` at source revision
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

Keep all owned point IDs in document reservation and collision checks, including
points absent from the visible entity order. Record the distinction explicitly
in planar schema 12. Earlier schemas retain their authored standalone-point
behavior; do not infer a point's role from whether a curve currently uses it.
Such inference would remove intentional markers and change old documents.

Delete through one owned dependency graph for all geometry kinds. Removing a
point removes curves that require it; removing a spline parent removes its
offsets. Retire defining points that no survivor needs, while keeping unselected
markers. Remove relations that lose required ownership through the existing
relation validator and metadata cleanup. Validate the seed and the complete
candidate before changing the owner, and return retired relations to the document
owner for its parameter and history cascade. Never renumber survivors.

This removes the assumption that every solver point must also be a standalone
entity row. It complements ADR-0015's shared solver, ADR-0016's derived-curve
parent boundary, and ADR-0005's global identity boundary. It does not supersede
those decisions or move curve fitting into the application.

Native model and package tests prove ownership, pruning, relation cleanup,
schema refusal/round-trip, and hidden-ID reservation. Public factory adoption,
entity deletion commands, parameter cascade, history/rollback, and live editor
acceptance remain required work; this record does not claim those paths are done.
