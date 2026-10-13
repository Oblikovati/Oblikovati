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
Elliptical arcs, copying, mirroring, trimming, splitting, extending, and native
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
