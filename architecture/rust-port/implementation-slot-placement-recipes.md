# Constrained slot placement recipes — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go `model/sketch/recipe_slot.go` and `recipe.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The model now represents the constrained straight and curved slot recipes used
by Go placement tools. Straight slots have six points, four boundary members,
a construction centerline, four tangent relations and one equal-radius relation.
Curved slots have seven points with one shared center and four circular tangent
relations. They omit the redundant construction arc and cap equal-radius relation.
Their initial degrees of freedom are five and six, respectively, with no redundant
equations. The separate raw API slot constructors retain their existing behavior.

A shared indexed path member replaces the raw slot's private member mapper.
Both paths reuse owned point allocation and primitive creation. A recipe commits
geometry, relations, parameter sources and locked dimensions in one candidate and
one session history command. Failure, observer veto and grouped abort publish none
of them; reserved identities remain retired. Undo and redo restore all owned fields.

Locked length/radius and width fields create existing distance and diameter
constraints. The angle/sweep field only controls placement and creates no dimension
without a reference edge. Empty locks are ignored. The document's default policy
makes a newly redundant dimension a reference dimension; its driving policy keeps
it driving. An unavailable locked expression is refused under either policy.
Field indices are checked even when unlocked. Annotation positions are optional,
caller-supplied values in working units. Go's automatic outward annotation placement
and physical 0.6 cm spacing are still pending.

Sixteen fixtures independently captured from the pinned Go implementation compare
points, member roles, constraint order, field bindings, witnesses and DOF. Regeneration
produces identical JSON bytes. Metric comparisons allow an explicit binary64 rounding
bound because raw seed and GUI formulas have different multiplication order. No
production tolerance or geometry repair is added. The pinned Go recipe, locked-field,
steering and redundancy-policy tests pass.

Model tests cover invalid indices and values, allocation exhaustion, late failure,
redundancy policies, persistence and history. Both slot recipes also reach DOF zero
with position/orientation constraints. Three length/radius and width edits retain
convergence with no redundant equations; every defining point is checked against
independent analytic coordinates. Session tests cover commit veto and grouped abort.

The desktop owner exposes the atomic commit operation. A real native test commits
both recipes, checks entity roles and reference keys, tests undo against the original
viewport pixels, and saves/reopens the geometry and locked parameter sources. The
free profile still reports its unavailable solid feature. This test does not prove
a general profile-to-solid path.

The [slot input step](implementation-slot-input.md) adds mouse placement state and
the native owner event path. Menu activation, preview rendering, editable field
widgets, automatic annotation positions and the remaining recipe families are still open. This step provides their
commit path; it does not complete interactive slot placement or R3. General analytic
region-to-solid construction remains a critical kernel blocker under the existing
kernel priority list. Kernel and API pins remain v0.2.5 and v0.158.0. No local source
override is used. Windows/macOS native runtime verification remains open.

The full local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests ran on a
1920×1080×24 Xvfb display with software Vulkan and validation layer 1.4.321.
Every crate exceeded 80% line coverage. All ten new production modules reached
100% line coverage. Duplication was 0.77% (545/70,485 scanned lines). The single
additional detected clone is six lines of typed error-cause dispatch; it does
not duplicate a geometry or commit algorithm.

The cross-platform and comment review found no new absolute geometric tolerance,
map-ordered identity allocation, platform-sized persisted identity or kernel
fallback. Local array indices refer to the fixed recipe graph. Permanent IDs
remain `u64`. Field positions use document working units. The raw slot mapper
was removed when the shared mapper replaced it; its existing regression tests
remain in the full gate. This change is local only, with no push or release.
