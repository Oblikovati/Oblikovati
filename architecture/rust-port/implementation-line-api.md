# Canonical inferred line constructor

Tracking: [#3775](https://github.com/Oblikovati/Oblikovati/issues/3775).
Prerequisites: permanent constraint reservations (#3776), centerline roles
(#3777), line alignment and filtered geometric indices (#3778), and the
[atomic document command](implementation-inferred-lines.md) (#3779).

## Contract and command

The existing Apache API `sketch.addEntity` contract now accepts `kind: line`.
One exhaustive constructor classification selects the inferred line command or
an existing primitive command. All use the shared coordinate/expression parser
and the normal session command. Reply preparation occurs before adoption.
No DTO, method constant, or numerical solver is redefined in the application.

Two literal points use centimetres and are converted to document working units.
Nonempty `pointExprs` take precedence and evaluate healthy length expressions
with current parameters and display units. Expressions seed coordinates; they
do not invent permanent dimensions. Wrong counts, null/invalid coordinates,
failed expressions, and invalid owners decline before allocation.

The Go constructor ignores line `variant`; Rust preserves that behavior. The
construction and centerline flags remain independent. The document's inference
preferences and physical snap reach control selection and application.

```rust
let input = oblikovati_api::wire::AddSketchEntityArgs {
    kind: "line".into(),
    points: Some(vec![Some(vec![0.0, 8.0]), Some(vec![4.0, 8.02])]),
    ..Default::default()
};
let created = client.sketch().add_entity(&input)?;
```

## Wire evidence

The reply imports `AddSketchEntityResult`, `AppliedConstraintInference`, and
`AppliedPointInference` from the API crate. Direction kinds use canonical Go
spellings, rather than formatting the API's frozen numeric enum values.
Horizontal and vertical records name the new line. Parallel and perpendicular
records name the new line and the existing reference line. The constraint index
is the actual geometric enumeration position, checked against the wire's i64
range. Dimensional relations are excluded. Point records name each new endpoint
and its existing snap target, in constructor order.

When constraint application is disabled, point proposals remain visible without
moving new coordinates. No direction record or fictitious index is returned.
Disabled inference returns neither record group. A zero-length line has distinct
fresh endpoints and no inferred direction. This preserves the documented Rust
corrections to Go phantom audit records and degenerate direction selection.

Reservations survive undo, abort, rejected replies, commit veto, save, reopen,
and copy. A vetoed constructor publishes no line or constraints. New attempts
use new IDs. File copies retain authored identities with new reference namespaces.

## Validation and remaining scope

Typed-client tests cover all four direction kinds, snap order and operands,
Go's horizontal wire fixture, variants, both role flags, preference modes,
zero-length inputs, unit scales, expressions and repair, current dimension
sources, filtered indices, malformed inputs, owners, history, veto, and storage.
Private projection tests cover unknown kinds and index range failures.
The pinned Go `TestAddEntityReportsAppliedInference` and
`TestInferenceOptionsOverWire` pass independently at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The full Linux gate covers every Rust workspace member, lint, documentation,
SPDX, coverage, and duplication. The generated and degenerate command corpus
remains covered by #3779. The [constraint-list query](implementation-constraint-list.md)
now makes returned geometric indices inspectable. Remaining constructors,
interactive drawing tools, profiles, native viewport acceptance, and kernel
integration remain separate port work. This step does not close R3 or the full
application port. Windows and macOS remain deferred.
