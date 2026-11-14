# Analytic body physical properties — partial R1/R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563),
under [ADR-0022](../decisions/ADR-0022-analytic-body-properties.md).
Sources: Go `addin/router/bodies.go`, `model/analysis/massprops.go`, and
`model/analysis/inertia.go`; API-owned `BodyPhysicalPropertiesArgs` and
`MassPropertiesResult` define the wire boundary.

`body.physicalProperties` now resolves the current body index in the same order
as `body.list`, verifies its semantic key, and integrates the retained B-rep.
The query does not edit history or consume a pending display update. A failed
feature contributes no stale body; negative and out-of-range indices are errors.
A file host without native result ownership refuses the query.

The shared native volume helper returns volume and centroid for both probe and
physical-property consumers. Full properties add surface area, centroidal inertia,
and native principal moments/axes. No display mesh is read. The host converts cm
to mm, unit-density inertia to g·mm², and refuses nonfinite converted values.
Accuracy accepts empty, low, medium, or high; all current planar prism cases use
the same analytic integral. The density default follows the existing part-material
contract. A body override remains visible in `body.list` but does not alter this
contract. Missing selected material definitions fail; explicit density bypasses
material lookup. Nonpositive explicit density uses one g/cm³, as in Go analysis.

The OCCT 8.0.0 generator's `--mass` mode records 27 existing prism corpus cases:
rectangle, inner loop, and nonconvex profile; three scales; positive, negative,
and skew sweeps. Tests assert area, volume, centroid, all six independent inertia
entries, and sorted principal moments. Existing per-face area/loop/centroid gates
remain active. Thin solids and a translated, rotated solid add independent checks.

For the rotated 4×3×2 solid, the exact inertia tensor is
`[[874/25,-168/25,0],[-168/25,776/25,0],[0,0,50]]` at unit density.
Jacobian operation `matrix.symmetric.rational_spectrum_claim.check` confirmed the
complete spectrum 26, 40, 50, each with nullity one. Native tests check that tensor,
its spectrum, orthonormal axes, and repeat results; they do not compare arbitrary
axis signs with OCCT. Runtime checks refuse invalid principal certificates.

Reproduce the corpus with the C++ command in
[the prism evidence](implementation-planar-prism.md#independent-occt-corpus-and-local-tests),
passing `--mass`. Run `cargo test -p oblikovati-martensite` and
`cargo test -p oblikovati-head --lib owner::tests::part_scene::physical`, or
`make gate` for the full workspace. Host tests cover units, density lookup,
overflow, edit/undo, local feature failure, and pending presentation.

This does not implement whole-part aggregation, assembly analysis, non-prism
feature families, or full foreground MCPBridge acceptance. Full port acceptance
remains open. Geometry recognizers, fallbacks, and native type assertions have
no increase; the one numerical residual tolerance is justified by ADR-0022.
