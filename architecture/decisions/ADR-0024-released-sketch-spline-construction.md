# ADR-0024: Released construction of authored sketch splines

Date: 2026-10-14. Status: Accepted for the Rust target design.
Boundary: owned model spline recipes to analytic Martensite curves.
Tracking: [#3789](https://github.com/Oblikovati/Oblikovati/issues/3789).

## Decision

Consume the owned sketch constructor in tested Martensite v0.2.7. Delete the
application's `unreleased_spline` failure and the whole-sketch spline precheck.
Existing control and fit recipes, including closed curves and named offsets,
resolve through the same analytic curve path as other sketch geometry.

Resolve defining points from the current read index. Map API fit-method constants
to the released kernel policy. The kernel constructs and checks the curve; the
application does not copy its knot, basis, collocation or interpolation algorithms.
Copy the accepted native poles, knots and multiplicities into the existing
application-owned B-spline definition. No mutable native handle escapes.

Retain source identity, fit stations, measured fit and seam residuals, and the
actual construction limits in the resolved result. The native version-one defaults
set separate fit, seam and derivative limits. The caller's endpoint bound is
checked separately where the source defines analytic endpoints. Presentation
deflection remains a sampling policy and does not change construction limits.
Preserve typed kernel failures with the source entity. Invalid entities remain
local display declines or typed query failures; none are omitted.

Offsets retain their parent identity and use the single sign translation already
required by ADR-0016. Sampling, display and intersection queries share the same
constructed recipe. No sampled line representation enters analytic queries.

This adopts the spline types that the model currently owns. Derivative handles
are not yet represented in that model; their authoring, persistence and API work
remain required. Native intersection completion is still not certified root
completeness. This decision does not enable trim/extension edits or general
sketch-region extrusion without their separate acceptance gates.

## Evidence and deletion

The application pins RustyCADKernel v0.2.7 at
`d40ed71b704efd08310a4c56a1c1c3a64d4e3e9b`. The retained OCCT 8.0.0
position and tangent oracles are exact copies of that revision's
`crates/tkgeomalgo/tests/sketch_spline/occt-8.0.0.csv` and
`occt-control-8.0.0.csv`. The retained Go fit recipes are in
`crates/martensite/tests/planar_spline/go-fit-oracle.json`. Tests cover all six
fit recipes, 169 OCCT construction samples, source identity, signed offsets,
typed failures, display, and native pixels on software Vulkan and AMD RADV.

This application change adds zero recognizers, zero tolerance constants, zero
fallback sites, and zero geometry type assertions. It deletes the old
`unreleased_spline` decline and the whole-sketch precheck.
