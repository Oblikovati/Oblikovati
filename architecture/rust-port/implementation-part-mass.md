# Whole-part analytic mass properties — partial R1/R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563),
under [ADR-0023](../decisions/ADR-0023-analytic-part-aggregation.md).
Sources: Go `addin/router/analysis.go`, `model/analysis/massprops.go`, and
`model/analysis/inertia.go`; the request and result remain API-owned.

`analysis.massProperties` now uses the same native result owner as body queries.
The owner validates every current body's semantic name, then integrates bodies in
feature replay order. Native `GProps::add` moves each inertia tensor to the system
reference location. The final tensor is about the combined volume centroid.
The existing native eigensolver and certificates supply sorted principal moments
and axes. A body query enters the same nonempty integration pipeline with one
body; a regression checks exact equality with the one-body aggregate.

Density lookup and accuracy validation are shared with `body.physicalProperties`.
The part assignment supplies zero-request density; explicit nonpositive values
normalize to one g/cm³. Missing selected assets remain errors. Unit conversion
and overflow checks are unchanged. Empty parts report zero integrals, resolved
density, and identity axes. Multiple separate bodies are summed, including their
areas when they overlap; this is analysis of the body collection, not a union.
Queries leave history and pending display updates unchanged.

Nine new OCCT 8.0.0 fixtures cover three scales and three body offsets, including
coincident bodies and translation in all three axes. Tests compare volume, area,
centroid, all six independent tensor entries, and principal moments. Additional
checks cover a known parallel-axis result, reversed input order within numerical
precision, empty systems, exact one-body equivalence, density defaults, rejected
requests, parameter edits, undo, and local feature failure. Existing per-face
construction tests remain active.

Rebuild the new corpus with the compiler/link flags from
[the prism evidence](implementation-planar-prism.md#independent-occt-corpus-and-local-tests),
using `crates/martensite/tests/analytic_system/occt_oracle.cpp` as the source.
Run `cargo test -p oblikovati-martensite --lib analytic_body` and
`cargo test -p oblikovati-head --lib owner::tests::part_scene`, or `make gate`.

The current body set still covers retained sketch prisms. Assembly aggregation,
other feature families, the separate `model.physicalProperties` summary, and
foreground MCPBridge acceptance remain open. This increment does not complete
R1, R7, or the full port. It adds no geometry algorithm, tolerance, fallback,
recognizer, or native type assertion.
