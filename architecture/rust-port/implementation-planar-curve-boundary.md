# Owned analytic curve intersections — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Dependency: Martensite v0.2.1,
`193572dc3665598c4948777c87375bad7dc8114a`.
The [released intersection probes](implementation-planar-intersections.md)
provide the native prerequisite evidence.

The internal Martensite adapter now accepts owned line, circle, ellipse, and
explicit B-spline definitions. Conic frames include center, local axis, and
travel direction. Optional bounds use the native parameterization. Finite
strictly increasing bounds must stay inside a nonperiodic parent's domain or
span at most one period. Periodic bounds retain their supplied phase, including
a circular arc across the seam. No endpoint extension, period adjustment,
coordinate perturbation, or tolerance retry occurs implicitly.

Each call constructs fresh kernel objects, uses the released general `GInter`
intersector, checks completion, and reads its fallible counts. No mutable kernel
handle escapes. Failure cannot publish a previous successful report. TKGeomAlgo
moves from a test-only dependency to the internal adapter's normal dependencies;
its exact version and revision do not change. No public wire method or API
contract changes in this slice.

Results retain both native parameters, operand-owned transitions, touch side,
relative orientation, and bounded, half-bounded, or unbounded overlap endpoints.
Unknown and undecided transition states remain explicit. Every reported finite
incidence is evaluated on both source curves. Its parameter must be finite and
inside that curve's domain. The adapter measures both spatial residuals and
refuses any non-finite value or distance above the caller's explicit bound.
Native confusion tolerance, intersection tolerance, and spatial residual bound
are separate required positive finite inputs. There is no hidden adapter epsilon.
Points sort by operand parameters then position; overlap bounds sort explicitly.
Report indices are not permanent entity identities.

`PlanarCurveFailure` separates invalid input, native exception, incomplete
computation, and invalid result. It retains the native exception through
`Error::source`; input context names the operand and its supplied definition.
Shared B-spline construction now preserves its native exception until each
caller chooses its error boundary. The existing spline evaluator keeps its
string interface and messages. Parent construction and left-to-right offset
sign conversion each have one implementation shared by evaluation and
intersection, as required by ADR-0016.

The offset-line test initially failed for an infinite parameter domain with
`Standard_ConstructionError: Domaine sur courbe incorrect`. The same geometry
and tolerances pass when the offset operand has finite bounds. The pinned
`int_imp_par_gen/intersector.rs` explicitly requires two bounds on a parametric
operand. The adapter retains that native failure. It does not substitute a
line-specific engine or automatically trim and retry an unbounded input.
This remains a released-kernel capability limit, not a completed extension.

Tests cover immutable inputs/results, independent calls after failure, native
cause retention, every option field, invalid coordinates/radii/domains,
clockwise/counterclockwise arcs, periodic seam parameters, finite left offsets,
rational and nonrational splines, elliptic crossings, tangency, endpoint
metadata, operand exchange, empty completed results, and both overlap directions.
Internal boundary tests supply incomplete and invalid native reports to verify
refusal, the exact requested residual bound, transition conversion, half-bounded
incidences, and deterministic ordering. These are boundary tests, not evidence
that the native solver emitted all injected invalid cases. The existing analytic
spline tests, including 135 OCCT reference samples, remain active.

Completion status and small residuals do not certify all roots, exact topology,
or the whole interior of an overlap. No application trim/extend command consumes
this boundary yet. Authored fit/control spline construction, edit selection,
constraint and feature updates, stable identities, undo, persistence, and live
native presentation of trim/extend remain open. There is no new sketch-edit UI
or full R3 acceptance claim.

Run the focused checks with:

```sh
cargo test --locked -p oblikovati-martensite --lib \
  --test planar_curve_boundary --test planar_intersections \
  --test planar_spline --test go_spline_recipe
```

The full local `make gate` passed for all 13 crates in the discovered workspace,
including all targets/features, formatting, Clippy, build, instrumented native
Linux tests under isolated Xvfb with Vulkan validation, doctests, Markdown/link/
SPDX checks, workflow lint, and duplication. Every crate exceeds 80% line
coverage. Martensite adapter coverage is 90.16% (1,833/2,033); the new boundary's
five modules cover 352/360 lines (97.78%). Production-code duplication is 0.75%
(510/67,647 lines); the configured scan excludes tests. Cross-platform and
comment review found no new runtime epsilon, platform-sized stored ID, geometry
recognizer, fallback, or type assertion. This is local Linux evidence; full
cross-platform and application trim/extend acceptance remain open.
