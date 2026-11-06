# ADR-0016: Derived sketch curve ownership and analytic evaluation

Date: 2026-10-11. Status: Accepted for the Rust target design.
Boundary: authored parent splines and derived offset curves.
Owner: sketch model for identity/variables; Martensite adapter for curve evaluation.
Tracking: [#3789](https://github.com/Oblikovati/Oblikovati/issues/3789).

## Decision

Keep one named parent spline recipe and a derived offset entity that references
that parent by owned identity. The offset adds exactly one signed working-length
variable. A distance dimension measures its magnitude and preserves the authored
side during edits, including the behavior of Go #1874. Parent defining points
remain shared sketch entities, not duplicated offset control points.

Use Martensite for analytic parent and offset evaluation, with explicit B-spline
poles, weights, knots, multiplicities, degree, and periodicity at the adapter
boundary. Keep kernel handles inside each evaluation. Fit/control construction
and any required Go-only fitting or handle rules belong in Martensite extensions,
not a second application geometry kernel. The shared solver in ADR-0015 still
owns numerical iteration; it does not evaluate sampled curve geometry.

The Go offset convention is positive to the left of the parent direction.
OCCT/Martensite uses the right normal. Translate the sign once at the adapter
boundary; do not reverse the parent or change its parameterization. Return
analytic position and derivatives. Display sampling is a separate derived view.

## What this removes

Completion removes the current `offsetSplineDim` missing-native-geometry decline
through the existing typed ownership, variable, dimension, session and API paths.
Do not port Go `OffsetSpline.Sample`/`offsetPolyline` as modeling geometry and do
not retain it as a fallback. Invalid representations, stationary parent tangents,
unsupported fit rules and nonrepresentable outputs return named errors with the
offending input. Never replace a failed spline with its defining polygon.

## Evidence and remaining acceptance

Go reference: `derived_curves.go`, `curvesample_spline.go` and
`dimension_offset_spline_test.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. Owner-published Martensite v0.1.2
contains `Geom2dBSplineCurve` and `Geom2dOffsetCurve`. Using them does not prove
fit-spline, offset ownership, or application profile acceptance.

Require OCCT analytic position/derivative parity, signed normal invariants,
rational/nonrational and periodic domains, degenerates, owned parent resolution,
DOF and independent Jacobians, live/reference formulas, diagnostics, canonical
API creation/edit/deletion in both owners, permanent identity, rollback,
save/reopen/copy, and the full local Linux gate before completing #3789.
