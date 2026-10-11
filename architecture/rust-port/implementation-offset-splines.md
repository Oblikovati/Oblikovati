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

## Go construction corpus and kernel extension

The retained `crates/martensite/tests/planar_spline/go_oracle.go` generator calls
the Go constructors at `aa8297d1b0ac33c024f19e54821536d76e8a447a`. Its
`go-fit-oracle.json` contains six five-point recipes: Smooth/centripetal,
Chord/chord-length and Sweet/uniform, each open and closed. These have 135
analytic position/derivative samples, including all returned fit stations.
The Rust `go_spline_recipe` test evaluates their explicit poles and knots in the
published kernel, checks independent authored point coordinates at stations,
and checks closed endpoint tangent agreement. Input definitions remain unchanged.
These checks prove recipe evaluation. They do not prove native construction,
curvature continuity, handles, control curves or degenerate fitting inputs.

To regenerate, use Go 1.27.0 with a `go.work` that uses the pinned Go application
and replaces `oblikovati.org/api` with the API repository's `go` module. From this
Rust repository, run with that work file selected in `GOWORK`:

```sh
go run ./crates/martensite/tests/planar_spline/go_oracle.go > /tmp/go-fit-oracle.json
cmp /tmp/go-fit-oracle.json crates/martensite/tests/planar_spline/go-fit-oracle.json
cargo test --locked -p oblikovati-martensite --test go_spline_recipe
```

The generated file was reproduced byte-for-byte locally. Kernel extension
[#9](https://github.com/Oblikovati/RustyCADKernel/issues/9) records the exact Go
construction and active handle rules, source references, corpus and acceptance
requirements. Published `Geom2dAPIInterpolate` uses two extra poles for cubic open
interpolation without tangent requests; Go uses one pole per station with averaged
knots. Equal input parameters alone do not prove construction equivalence.
`PointsToBSpline` approximation does not replace interpolation. This extension
belongs in Martensite and must pass the owner's tests and release workflow before
the application adopts it.

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

## Public spline inspection

The canonical typed `sketch.entities` query now reports the stored fit method as
`smooth`, `chord`, or `sweet` for fit parents. It uses the released API enum and
existing `SketchEntityInfo.fit_method` contract. Control parents, offsets and
other geometry retain the Go API's empty fit-method field. Unknown enum values
are refused rather than reported as a valid method. This ports the Go
`splineFitSpelling` behavior without adding a second curve evaluator.

API acceptance tests install native recipes through the document command owner,
then use the typed public client to query part and assembly documents. They check
all three fit methods, distinct control/offset kinds, shared defining points,
authored order and persistent key resolution. Save/reopen retains the full rows;
copy retains local identities and methods while changing the document key
namespace. These tests cover public inspection of stored recipes, not public
entity creation, spline edits or deletion.

## Native distance dimensions

The typed seeded constructor now binds one offset entity to a length source and
captures its nonzero left/right side. Ownership checks consume no IDs and do not
select the branch. A zero seed requires an explicit side relation; automatic
selection refuses it before appending. An existing retained relation can drive
to zero and back without losing its side or numerical rank.

One scalar length residual replaces the previous radius-only equation. Its
positive domain retains the original radius checks and residual/gradient bits;
its signed domain maps the retained side to the magnitude row. The offset row
has one derivative, +1 on the left or -1 on the right. It matches the pinned Go
absolute-value residual on nonzero seeds, with a nonstationary signed derivative
at zero. Radius and offset constraints use this same scalar implementation.

Model tests ground parent points and drive both sides to DOF = 0. They cover
source edits through zero, current reference measurement with an unhealthy
excluded source, rejected driving formulas and units, deletion, explicit zero
side, generated working-distance scales, and independent parent preservation.
The scaled tests request an explicit numerical tolerance for small offsets.
Planar schema 11 retains the offset relation and side; earlier versions cannot
contain it. Parent-only recipes remain schema 10. Native package tests restore
source expressions, IDs and side in both part and assembly documents, then solve
the restored negative-side offset to its independently checked magnitude.

This proves native dimension behavior. Canonical API dimension creation remains
at its explicit decline until public entity creation and fitting acceptance are
available. No end-to-end add-in, application window or B-rep claim follows from
these native tests.

## Remaining port

Preserve the Go fit/control distinction, periodic closure, public fit methods and
active handle rules. The Go parent recipe stores shared defining points. Required
construction/fitting extensions belong in Martensite. The native distance relation is in place. Public creation and fit/handle
evaluation must use the owner-published kernel construction capabilities.

Keep #3789 open until typed ownership, solver/Jacobian and degenerates, canonical
API creation/edit/deletion, both document owners, live/reference formulas and
health, identity/annotation/history/rollback, and save/reopen/copy pass. The
current missing-native-geometry decline is not removed by this adapter alone.
