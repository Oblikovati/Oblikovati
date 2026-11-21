# Three-point circles and arcs — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go `model/sketch/construct.go` and `addin/router/sketch_add_entity.go`
at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The typed `sketch.addEntity` call now accepts `variant: "threePoint"` for circles
and arcs. The desktop owner, queued add-in calls, and CLI Lua host inject the
released Martensite construction service. The model owns the small construction
interface and value types. Automation has no kernel dependency. A host without
that service returns an explicit missing-capability error before reserving IDs.
No public wire schema or API version changes are needed.

Construction uses the existing
[owned circumcircle adapter](implementation-circumcircle.md). The kernel and API
pins remain v0.2.5 and v0.158.0. There is no copied circumcircle algorithm, local
kernel source override, retry with wider bounds, or coordinate perturbation.
Inputs use the existing expression/literal resolver and document working units.
Exactly three positions are required. The circle stores only its new center;
the arc stores its new center, first point, and last point. The middle point is
an input to construction and verification, not an extra stored sketch point.
Arc winding follows the ordered input triple. The radius and `ccw` request fields
do not override this constructor, as in the pinned Go implementation.

The composition roots select version 1 of an explicit authoring acceptance policy:
maximum measured point residual `1e-9` database centimetres. Its physical value
comes from pinned Go `math.DefaultTolerance`; this new postcondition is not a
claim that Go's circumcircle constructor performed the same verification.
The bound is converted once to document working units. Invalid versions, scales,
negative/nonfinite bounds, overflow, and positive bounds that underflow to zero
are refused. An explicitly requested zero bound remains zero. This policy is
separate from the sketch solver's numerical tolerances and topology predicates.

Creation uses the existing owned primitive operation and shared transaction
pipeline. A new projected-command callback runs after exposed-property sync and
automatic sketch solve, but before candidate adoption. The service checks the
actual final center/radius, all three original point residuals, and actual arc
endpoint radial residuals. It also checks endpoint displacement and exact winding
through the middle input. A refusal discards the complete candidate while keeping
its reserved IDs retired. Earlier commands in an open group survive. There is
no second solve/adoption path and no response prepared from an unverified seed.
Existing commands use the same pipeline with an identity projection.

The checks measure binary64 radial residuals; they are not interval bounds or a
center-error certificate. An accepted authored curve is not proof of a valid
closed profile or a solid. General analytic regions and certified trim/extension
incidences remain kernel blockers in the kernel's application priority list.
Three-point arc-slot composite creation and full R3 remain open.

Tests cover injected service calls, owned point order, construction roles, unit
conversion, reference keys, undo/redo, save/reopen, missing capability, malformed
input, and final-verification rollback with retired IDs. Session tests use a
constrained line whose authored endpoint moves during solve, proving that the
projection sees solved coordinates. Rejected projections preserve both automatic
and caller-owned transaction state. The real adapter tests retain their OCCT
fixtures and generated cases and add final-result acceptance/refusal checks.

A native desktop test rejects collinear input, creates each curve through the
typed API with the real service, checks reference resolution, observes the free
profile's feature-health failure, and verifies exact original viewport pixels
after undo. It runs on Xvfb with software Vulkan and validation enabled. A real
CLI Lua process creates both curves, checks returned IDs and radii, and saves a
native document that a separate CLI invocation can read. The initial CLI test
used the wrong JSON field case and omitted the default projected origin point;
those test assumptions were corrected without changing production behavior.

Windows and macOS runtime checks remain open. This change is local only; no push
or release is included.

The final local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint, and duplication. Native tests ran serially on a
1920×1080×24 Xvfb display with software Vulkan and validation layer 1.4.321.
Every crate exceeded 80% line coverage. The four new production modules reached
100%: construction policy 27/27, injected host access 11/11, three-point authoring
111/111, and kernel service/final verification 78/78. Model coverage was 98.80%,
session 98.82%, automation 96.39%, and Martensite adapter 91.45%.
Duplication was 0.78% (539/69,302 scanned lines). The two new matches are generic
projected-command signatures; edit, solve, rollback and adoption remain shared.

The cross-platform and comment review preserves the moved adapter type comments
and finds no new geometry engine, topology epsilon, unordered identity allocation,
or platform-sized persisted ID. The physical acceptance bound has an explicit
origin and unit. Both positive and negative subnormal conversion cases have
regression tests; neither can silently become an accepted zero bound.
