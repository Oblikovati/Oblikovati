# Point and circular constructors over the API — partial R3

This step connects canonical `sketch.addEntity` to the shared owned primitive
command. It implements
[issue #3771](https://github.com/Oblikovati/Oblikovati/issues/3771).
Supported constructors are standalone point, center-radius circle, and
center-start-end circular arc. Empty variants select these defaults; circles
also accept `centerRadius` and arcs accept `centerStartEnd`.

The host imports API 0.157.0 DTOs, method constants, kind constants, and typed
client methods. It returns actual entity and defining-point IDs from the document
command. It does not create a separate geometry owner or allocation path. Part
and assembly documents share the implementation; absent sketch indices, other
document kinds, and read-only hosts refuse creation.

Literal coordinates are centimetres and use the same checked pair conversion as
selection translation. Nonempty `pointExprs` overrides `points`, including when
the unused literal input is malformed. Each expression pair must have two length
expressions. Circle radius uses the existing contextual length evaluator, which
supports healthy current nominal parameters, formulas, explicit units, and bare
numbers in document display units. Results use document working units. Empty
expression arrays retain the literal path.

Expressions supply constructor seeds, as in Go. No parameter or live dimension
binding is created by these constructors. A later parameter edit affects the
next constructor evaluation; existing geometry needs an authored dimension to
stay driven. For example, after creating a sketch, a typed client can create a
circle with these inputs:

```rust
client.sketch().add_entity(&AddSketchEntityArgs {
    kind: "circle".into(),
    points: Some(vec![Some(vec![1.0, 2.0])]), // Centimetres, independent of display units.
    radius: "10 mm".into(),
    ..Default::default()
})?;
```

Shape, count, variant, expression, and conversion checks run before reservations.
The model's positive radius check also precedes allocation. Circles and arcs
retain construction roles; plain points have no curve construction or centerline
flag, matching Go. Arc direction remains authored. Automatic recompute, response
preparation before adoption, groups, undo/redo, observer veto, and native file
storage use the existing command boundary. Veto discards geometry but does not
rewind consumed IDs: the changed reservation marks still require a save.

Tests cover constructors and defaults, canonical variants, both arc directions,
roles, real IDs and keys, parameter formulas and failure/repair, expression
precedence, input and display units, malformed/null inputs, dimension and radius
errors, overflow, atomic refusal, reservations, groups, undo, veto, save/reopen,
copy namespaces, and document ownership. Independent pinned Go expression,
point-expression, and constructor-refusal tests pass at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

Go applies inference to new lines by default. That path remains required, so the
wire line constructor declines by name. Three-point circle/arc overloads, other
geometry kinds, centerline formatting, dimensions, native tools, profiles, and
kernel consumers also remain required. Unknown kinds and unported variants
decline before allocation. This step does not claim a fully constrained sketch,
the full constructor family, or native rendering acceptance.
