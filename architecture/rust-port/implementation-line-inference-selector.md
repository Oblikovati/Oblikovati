# Shared line-placement inference selector — partial R3

This step implements
[issue #3774](https://github.com/Oblikovati/Oblikovati/issues/3774).
`Sketch2D::infer_line_placement` proposes endpoint coincidences and the winning
horizontal, vertical, parallel, or perpendicular relation for a fresh segment.
It reads existing owned points and lines, returns their real local identities,
and changes no recipe state, solver state, or identity reservation. Construction
lines remain valid direction references. These are interaction proposals, never
modelling predicates or proof that two entities are incident.

The injected `SketchInferencePolicy` has a checked angular reach in `(0, pi/4)`
and positive finite working-unit snap reach. Disjoint angular families prevent
classification from depending on a first-fit order. Nonfinite endpoints decline
before selection, even when inference is disabled. The default factory preserves
Go's 3-degree angular reach and 0.001-cm physical snap reach. It converts that
length through the document working scale, correcting the scale dependence
recorded in legacy [#2619](https://github.com/Oblikovati/Oblikovati/issues/2619).
The future cursor owner can inject a view-dependent reach through this same policy.

Point candidates are fixed before selection. Fresh endpoints cannot snap onto
each other, but both can propose the same existing target. Nearest distance wins;
equal distances select the last authored point, as in Go. Distance uses `hypot`
to prevent overflow or underflow from creating false coincidences. The snap
boundary is closed. Infer-without-constrain reports suggestions while retaining
the input endpoints. When constraining is enabled, virtual snapped endpoints
feed direction selection, matching the Go snap-before-direction sequence.

Direction suggestions keep the Go integer quality score and preference rules.
Horizontal/vertical priority selects that family when both exist;
parallel/perpendicular priority selects its family regardless of score. With no
family preference, greater quality wins and equal integer scores select the axial
family. Equal relative scores select the first authored line. Unknown stored
priority IDs retain the Go horizontal/vertical preference behavior.

The conditioned relative-angle calculation retains baseline
`aa8297d1b0ac33c024f19e54821536d76e8a447a` arithmetic and score truncation.
When squared lengths or their product leave the normal floating-point range,
it reuses the existing robust direction normalization. This includes positive
subnormal products: rounded dot/length ratios must not fabricate a parallel
relation. A failing 6.34-degree separation at `2e-162` is a regression fixture. Coordinate subtraction also reuses the shared overflowing-point
difference helper. This is one selector with conditioned arithmetic, not another
inference engine. A zero-length probe or reference has no direction. This
corrects Go `math.Vector2::AngleTo` returning zero for such vectors and thus
reporting them as parallel. Large, subnormal, and opposite-extreme finite
coordinates have hard direction-family assertions.

For example, a placement owner can inspect an immutable proposal:

```rust
use oblikovati_model::sketch::SketchInferencePolicy;
let policy = SketchInferencePolicy::for_units(snapshot.parameters.units());
let proposal = geometry.infer_line_placement(
    [[0.0, 0.0], [4.0, 0.01]],
    &snapshot.effective_sketch_settings(),
    policy,
)?;
```

Tests cover all direction families, orientation reversal, nearest and tied
candidates, quantized score precedence, closed boundaries, explicit policy
validation, both inference toggles, virtual snap order, recipe immutability,
construction references, floating-point range limits, 144 generated relative
rotation cases, and five document scales. Go tests and a direct Go selector
probe check the well-conditioned family and integer-score fixtures. Analytic
rotation, scale, and range tests independently check the intended geometry.

[Issue #3775](https://github.com/Oblikovati/Oblikovati/issues/3775) owns production
integration: permanent constraint reservations, atomic line creation with inferred
relations, and canonical applied-inference responses. It must replace the current
API line decline. This selector does not allocate constraints, apply relations,
run the solver, add API line creation, or supply a live cursor overlay. Those
remain required work, and R3 and all kernel integration checks remain open.
