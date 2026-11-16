# Fillet history probe

Base `9e7f0fddc`, pinned Martensite
`bef25f47548431c8f128f08f66a1f719bcee4982`, Rust 1.99.0. These are
headless kernel compatibility probes, not a desktop feature or BDD acceptance.

A 10-unit box with one 2-unit edge fillet produced a valid body and the
closed-form analytic volume. The test also checked seven result faces: six
planar areas, one curved area and one loop on each face. The curved face may
be represented as a cylinder or rational spline; the area check alone does
not certify its exact surface. Generated, modified and retained face history
covered every result face, but this does not prove unique durable names.

The box had 12 input edges and 15 output edges; direct retained or modified
history attributed 11 output edges. The remaining edges need an identity path,
for example from named neighbors, with ambiguity refused. A second probe copied
the host's prism and its selected edge, then checked a valid result, source
volume and face coverage. It did not resolve a document-scoped edge key.
The host feature, full provenance, replay, UI and scenarios remain open (#4045).
