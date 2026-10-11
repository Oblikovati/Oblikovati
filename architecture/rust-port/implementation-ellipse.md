# Native ellipse port — partial R3

Tracking: [#3788](https://github.com/Oblikovati/Oblikovati/issues/3788).
Decision: [ADR-0015](../decisions/ADR-0015-parameters-and-constraints.md),
which supersedes the shared parameter/solver boundary in ADR-0006.
Go reference: `model/sketch/entities.go`, `ellipse_orientation.go` (#1879),
`dimension_advanced.go`, and `dimension_advanced_test.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

## Analytic value

`SketchEllipse2D` is an immutable validated value with distinct curve and shared
center identities, two radii, and a normalized major-axis direction. It owns no
B-rep or document. Preparation does not prove that its center exists in a sketch;
that check belongs to adoption. Radius edits preserve the frame, and unchanged
rotation preserves axis bits. One orientation angle is intended for solver
integration; two free axis components would introduce a false magnitude freedom.

The radii must be finite with major >= minor > 0. Equal radii are valid and retain
the ellipse identity. A radius crossing is an error; it does not swap the radii
or change the axis. `Geom_Ellipse` in the local OCCT source checks the same radius
order, while allowing a zero minor radius that the sketch value refuses.
Finite axis directions use scaled normalization to avoid underflow and overflow.
Analytic evaluation returns position and tangent in working units; non-finite
inputs or nonrepresentable outputs return named errors. No polyline or circle
approximation enters this computation.

## Oracle and regression evidence

The Linux OCCT 8.0.0 installation at `occt-install` supplied 108 position and
first-derivative samples through `Geom_Ellipse::EvalD1`. They cover three working
scales, three axis directions, unequal/equal/thin radii, and four parameters.
The current source checkout used to inspect radius rules is
`3d097a0328e71b826377d4814ab05ec3c3d23871`; this is not proof that the installed
libraries were built at that commit. The fixture records the installed version.

The generator is `crates/model/tests/sketch_ellipse/occt_oracle.cpp`.
Its output is `occt-8.0.0.csv` in the same directory. Each row contains center x/y,
input axis x/y, major/minor radius, parameter, position x/y, and tangent x/y.
The recorded generator was compiled against the installed OCCT headers and
linked to TKG3d, TKMath, and TKernel, with the installation's library directory
on `LD_LIBRARY_PATH`. Its version output was `8.0.0`.

`cargo test --locked -p oblikovati-model --test sketch_ellipse` checks these
samples with a scale-relative floating arithmetic bound. Separate tests check
symmetric-difference derivatives, invalid identities/radii/axes/angles, output
overflow, unchanged orientation bits, equal radii, and subnormal/maximum finite
axis directions. The CSV is compiled into the tests; the normal gate does not
need to load external OCCT libraries.

## Remaining ownership and solving work

The value now enters `Sketch2D` through atomic shared-center adoption. Cross-kind
identity collisions are refused. Ordered inspection, construction filtering,
and shared-center translation include ellipses. The solver layout adds two
radii and one orientation after the existing point/circle columns; a fresh
ellipse has five freedoms, and grounding its center leaves three. Candidate
radius/rotation edits validate before publication, while unchanged orientation
keeps frame bits.

Planar schema 7 records validated ellipse values and explicit role/order/settings
metadata. Schemas 1-6 remain readable and cannot contain ellipse values. Saved
axes must be unit directions within a normalization roundoff bound and retain
their exact bits. Invalid radii, axes, missing schema fields, centers, and
cross-kind identities are refused. History comparison sees axis signed-zero
changes. Native-package tests preserve frames, insertion order, construction
roles, and identity marks through encode/decode/restore and refuse lower marks.
Generic inspection reports zero for the circular-only radius field, as required
by the released API; major-radius measurement has its own dimension path.

## Public entity creation

The released `sketch.addEntity` contract creates an ellipse from one center,
a two-component axis direction, and major/minor length expressions. Literal
centers use centimetres; expression centers and radii use the document quantity
context. The axis is unitless. Expressions seed geometry without inventing live
dimension bindings. The constructor ignores `variant`, as the pinned Go route
does. Construction roles use the normal primitive command.

Frame and point validation precede identity reservation. Typed creation supports
an existing owned center, including the grounded origin. Tests cover part and
assembly owners, scaled working units, malformed frames and expressions,
unchanged state on refusal, rejected replies with retired IDs, undo/redo,
save/reopen, and copy with new reference keys. A publicly created ellipse accepts
the major-radius dimension below.

## Major-radius dimension

The native `EllipseRadius` relation uses the existing scalar radius equation
with the ellipse's major-radius variable. It does not project the ellipse to a
circle. Its exact derivative is one for the major radius and zero for the other
four variables; the rank test removes one freedom. Reference mode excludes its
target and still measures current geometry. The relation survives planar storage.

The canonical `ellipseRadius` API route now accepts one owned ellipse and creates
a fresh live Model formula through the existing atomic owner/session path. Tests
cover part and assembly creation, source propagation, reference metadata,
undo/redo, save/reopen/copy, edit, deletion/recreation, and malformed operands.
Radius crossings fail without adopting a partial recipe or reusing names.
The former missing-native-ellipse constructor decline is removed.

Remaining #3788 work includes minor-radius and orientation/axis constraints, fully constrained and degenerate
cases, ellipse-specific failed formula and transaction acceptance, and edited
save/reopen/copy. This step does not establish profile-kernel acceptance. B-rep
ellipse operations stay in Martensite; the sketch value does not replace kernel
geometry or establish kernel operation validity.
