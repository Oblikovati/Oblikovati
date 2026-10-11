# Current sketch entity inspection — partial R3

This step implements `sketch.entities` from `Oblikovati.API` 0.157.0 for the Rust
model's points, lines, circles, and circular arcs. It implements
[R3 issue #3767](https://github.com/Oblikovati/Oblikovati/issues/3767).

The API returns the authored entity order and current local sketch coordinates.
Points have one defining point, lines have two shared endpoints, circles have a
center and independent radius, and arcs have a center, start, end, and analytic
radius derived from the start. Coordinates and radii are converted from document
working units to centimetres. An unrepresentable conversion refuses the response
with the offending value and scale. Every returned reference key is checked by
the current document catalogue before encoding.

`Sketch2D::inspect_entities` prepares one current grounded system and evaluates
its Jacobian once. Coordinate freedom uses the checked shared QR diagnostics.
`GroundedLayout` restores the reduced freedom flags into the original coordinate
order and assigns `false` to every grounded constant. The same layout restores
current anchor coordinates for shape presentation. Inspection neither iterates
the nonlinear solver nor adopts a candidate or changes a saved seed.

Entity variables use the existing solver layout: point coordinates, shared line
endpoints, circle center coordinates and independent radius, and the three arc
points. An entity is `freeToMove` when any of these variables can move locally.
If all are determined, an owned fix point or grounded defining point makes the
entity `fixed`; otherwise it is `byDimensionChange`. This retains the Go
classifier's fix-point precedence. It does not infer freedom from which column
became a Jacobian pivot and does not establish finite drag motion.

`SketchMovementOptions` keeps rank and squared-freedom thresholds separate. The
session supplies its injected rank tolerance and retains Go's `1e-6` squared
projection threshold. A bad policy refuses inspection, including empty inputs.
The document boundary resolves current associative point sources first. A
suppressed host or unhealthy source refuses inspection instead of displaying a
last-good projection. Dimension compilation also requires healthy current
parameter values.

The read-only file host and mutable automation host use the same query path and
canonical API DTOs, method constants, and typed client. Part and assembly sketches
are supported; other document kinds and absent owners decline. Empty sketches
return an empty entity array. Owned curves report their saved construction role;
plain points have no construction flag. Reference-curve metadata remains absent,
so that flag is false. Point associations retain
their exact grounded behavior; the reference flag does not invent a projected
curve representation. Formatting and spline fit metadata remain absent for these
primitive kinds, as in the Go response.

Tests cover every current shape, mixed authored order, shared points, analytic
radii, fixes and dimension control, coupled freedom, grounded variable mapping,
current anchors, failed policies and inputs, centimetres and overflow, current
reference round trips, save/reopen/copy, both host types, document ownership, and
history/geometry/dirty immutability. Corresponding pinned Go movement and router
reference tests pass independently at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

Native construction selection and armed modes, included curves and their metadata, additional geometry
kinds, interactive drag, native pick/render acceptance, and feature consumers
remain required for the complete application port.
