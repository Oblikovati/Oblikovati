# Owned sketch rotation — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Go reference: `model/sketch/edit_ops.go`, `addin/router/sketch_edit.go`, and
`addin/router/param_expr.go` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The released API 0.158.0 already defines `sketch.transform` with `op: "rotate"`,
a centimetre center, and an angle expression. The Rust file host now implements
that operation through the existing document command. It returns no new entity
IDs. No public wire type, method string, client binding, or dependency pin changes.

`SketchSelectionTransform` is an internal application model value. Translation
and rotation share selection resolution, unique defining-point collection,
finite-output preparation, candidate adoption, and current-source recompute.
The former separate translation arithmetic is removed. Existing translation
wrappers retain their behavior, including the Go affine expression's signed-zero
result. Diagnostics now identify the transform and affected point.

Rotation applies the Go affine expression to points, line endpoints, circle
centers, arc defining points, ellipse centers, and spline defining points.
Shared points move once. Selected ellipse axes rotate once per owned entity;
a selected center alone does not rotate its ellipse axis. Axes retain the model's
unit-direction invariant under repeated rotations. Radii, IDs, incidence, curve
roles, ordering, constraints, and reference keys remain owned by the same objects.

Two intentional corrections to Go behavior are explicit. Repeating an ellipse
in the selection rotates its axis once, consistently with its unique center;
Go's `transformEntityDir` runs once per selection occurrence. A length quantity
such as `"1 cm"` is refused as an angle. Go's literal fallback can return a
length because a suffix overrides the parser's default category. The Rust route
checks the final category and finiteness before the command runs. Healthy named
angle parameters and formulas resolve at command time; bare numeric angles use
the document's preferred angle unit. This seed edit creates no live angle driver.

Selecting an offset spline directly declines with its identity and a request to
select its parent. Rotating the parent retains the offset relation and distance.
This avoids the Go default branch's silent no-op for an unsupported entity kind.
Copy is now covered by the [copy evidence](implementation-sketch-copy.md).
Mirror is covered by the [mirror evidence](implementation-sketch-mirror.md).
Elliptical arcs, trimming, extending, and native
interactive rotation remain separate port work.

Invalid centers, angles, selections, converted coordinates, or output coordinates
publish no partial edit. Reply rejection and observer veto discard the candidate.
The shared command supplies grouping, abort, undo/redo, saved state, and reload.
Constraints and grounded projections still govern the solved result. Numerical
nonconvergence retains the existing sketch health behavior.

Model tests cover independent quarter-turn coordinates, shared points, metadata,
ellipse directions, control/fit spline points, offset refusal, finite overflow,
empty and zero edits, and 256 ellipse rotation/JSON-reload cycles. API tests cover
working-unit scales, angle expressions and named parameters, bare input, bad
categories, history and storage, response rejection, veto, read-only hosts, and
part/assembly grounded recompute. Existing translation tests remain active.
The baseline Go move/shared-point and ninety-degree rotation tests pass.

A native window test opens the stored parameter-driven prism and rotates its
profile seed by 15 degrees through the typed API. Recompute restores the fully
constrained profile. The presented RGBA buffer remains byte-identical, and the
existing X11 pixel check verifies the visible result. This uses isolated Xvfb at
1920×1080×24, software Vulkan, and validation layer 1.4.321. It does not establish
hardware presentation, general profile-region acceptance, or a completed R3.

The full local `make gate` passes over all 13 crates in the discovered workspace:
all targets/features, lint, format, instrumented tests, native window checks,
doc tests, Markdown/link/SPDX checks, workflow lint, and duplication checks.
Model coverage is 98.80%, session 99.03%, and automation 96.25%; every crate
exceeds 80%. The shared point-edit implementation, new affine transform module,
and wire transform route each have 100% line coverage. Workspace duplication is
0.72% (478 of 66,297 Rust lines). This evidence uses the released API and kernel
pins; it does not substitute for the remaining port milestone acceptance.

## Constructor-owned defining points — 2026-10-13

A constructor-boundary regression found a gap in the selection traversal at
`cc61b496`. Primitive constructors can hide defining points from `entity_ids()`.
The transform collected their IDs from the curve, then filtered against that
visible list. As a result, a newly created free line could return a successful
move while its endpoints stayed unchanged. Rotation could turn an ellipse axis
without turning its hidden center. Earlier tests used visible points or a
constrained center that was expected to stay in place; those premises did not
cover this case.

Both model regression cases failed before the fix: move and rotate returned an
empty affected-point list instead of the three shared defining-point IDs. A
separate typed API test created a line through the primitive command, requested
a `[3,4]` move, and observed unchanged `[2,0]` and `[4,0]` endpoints instead of
`[5,4]` and `[7,4]`. These tests exercise the actual point-ownership representation.

The shared transform now filters the complete `owned_entity_ids()` enumeration.
Visible entity order remains unchanged; hidden defining points follow in their
owned order. There is no separate branch for a constructor or curve type.
Regression coverage includes hidden line/arc endpoints, circle/ellipse centers,
control/fit spline points, shared points, duplicate selections, mixed visible
and hidden points, atomic overflow refusal, and unchanged visibility metadata.
The typed API case also checks rotation, keys, undo/redo, saved reload, and another
move after reload. This fixed a prerequisite for the subsequent
[copy implementation](implementation-sketch-copy.md).

The defining-point fix passes the full local gate over all 13 crates, including
native display tests. Model line coverage is 98.80% (17,459/17,671); every crate
exceeds 80%, and duplication remains 0.72%. The production change replaces the
visible enumeration with the existing complete ownership enumeration. It adds no
geometry formula, tolerance, entity allocation, wire method, or dependency pin.

Line split is now covered by the [split evidence](implementation-sketch-split.md).
