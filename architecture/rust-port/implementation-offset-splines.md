# Native offset-spline port — partial R3

Tracking: [#3789](https://github.com/Oblikovati/Oblikovati/issues/3789).
Decision: [ADR-0016](../decisions/ADR-0016-derived-sketch-curves.md).

The application adapter consumes owner-published Martensite v0.1.2 at
`19108f6650cdfca7050febc759d91bdfec001769`. Its new tkg2d, tkernel and tkmath
imports use that same exact version/revision; there are no sibling path imports.
It creates a fresh B-spline from explicit poles, optional weights, knots,
multiplicities, degree and periodicity for each evaluation. Handles remain local.
It returns analytic position and tangent in working units with respect to the
parent parameter. It uses neither a display mesh nor a sampled offset polygon.

Positive Go offset distance means left of the parent direction. Martensite
`Geom2dOffsetCurve` uses the right normal. The adapter negates the signed distance
once, retaining parent parameterization. Zero offset evaluates the parent.
Parameters must lie in the owned domain, even for periodic parents. Malformed
representations, non-finite input/output, stationary nonzero-offset parents and
exact offset cusps return named errors. No point is moved to repair a failure.

## Oracle evidence

The installed OCCT 8.0.0 library independently generates 135 position/tangent
samples with `Geom2d_BSplineCurve` and `Geom2d_OffsetCurve`. The retained generator
is `crates/martensite/tests/planar_spline/occt_oracle.cpp`; its CSV is alongside it.
It covers nonrational and rational open quadratic curves, a periodic quadratic
curve, three working scales, both offset sides and zero, and five parameters
including domain endpoints. This is the installed library's version, not a
claim about its source commit. The Rust gate compiles the CSV into its tests.

Additional tests check the independent signed normal, finite-difference tangent,
immutable input, invalid representations and domains, and stationary/cusp cases.
Run `cargo test --locked -p oblikovati-martensite --test planar_spline` or the full
`make gate`. This proves the analytic adapter for these representations; it does
not prove sketch spline ownership or profile-to-B-rep evaluation.

## Native ownership and storage

The sketch now owns `SketchSpline2D` parent records with shared defining-point
references, fit/control intent, fit method, and closure. Offset records resolve
one exact owned spline parent and retain a finite signed working distance.
Adoption validates all references and cross-kind identity collisions before
changing the owner. Control and fit parents use their distinct released API
kinds; offsets have no defining points of their own, matching Go `ShapePoints`.

Defining points use the existing coordinate variables. Another parent with
those same point references adds no duplicate coordinates. Each offset adds one
signed-distance column after existing ellipse columns. Candidate writeback
validates a finite distance and preserves the input owner on failure. Grounding
parent defining points leaves the offset's one freedom. This is ownership/DOF
acceptance, not offset-distance driving or curve fitting acceptance.

Ordered inspection, construction/profile candidate selection, and translation
include parents and offsets. Translation changes each selected defining point
once; selecting an offset does not translate its parent implicitly. Planar
schema 10 records explicit parent and offset lists and the existing metadata.
Older schemas cannot carry these new fields. Save/restore resolves parents
before offsets. History compares signed-distance bits, including signed zero.

Model tests cover sharing, DOF, grounding, kinds, translation, invalid ownership,
writeback and schema refusal/round-trip. Native package tests preserve IDs,
roles and distance in part and assembly packages and refuse identity marks below
any parent, offset or defining point. No API creation constructor or general
fit/handle evaluation is claimed by these ownership tests.

## Remaining port

Preserve the Go fit/control distinction, periodic closure, public fit methods and
active handle rules. The Go parent recipe stores shared defining points. Required
construction/fitting extensions belong in Martensite. The next sketch step must bind the owned signed-distance column to the existing
scalar magnitude dimension path and preserve the selected side during driving.

Keep #3789 open until typed ownership, solver/Jacobian and degenerates, canonical
API creation/edit/deletion, both document owners, live/reference formulas and
health, identity/annotation/history/rollback, and save/reopen/copy pass. The
current missing-native-geometry decline is not removed by this adapter alone.
