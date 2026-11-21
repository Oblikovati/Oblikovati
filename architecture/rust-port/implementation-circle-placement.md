# Circle placement — partial R3/R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go circle tools, preview and CircleRecipe at
aa8297d1b0ac33c024f19e54821536d76e8a447a.
Kernel/API pins remain v0.2.5/v0.158.0 with no local source override.

Centre-radius and three-point circles now use the same mouse placement state as
rectangles and slots. Click and drag input, preview, failure recovery, Escape,
one-step commit and tool restart share the existing path. Preview does not change
document history. Finish retains the committed geometry; Undo and Redo replace
the displayed sketch.

The shared recipe member now describes either a line/arc path member or a closed
circle. Circles do not acquire artificial path endpoints. Commit resolves local
point indices to canonical primitive inputs; scratch preview uses that same
commit path in its private namespace. The existing kernel sampler draws the
circle. There is no second circle renderer, persistence format or geometry owner.

A circle recipe owns one centre point, one radius and a diameter field. Its
intrinsic rank is three DOF, with no extra relations. The field witness and
binding match Go. A locked diameter creates an owned parameter through the
existing atomic field path. Fixing the centre and locking the diameter yields
zero DOF without redundant equations. The mouse path remains unconstrained until
the user adds constraints; editable diameter widgets remain open.

Three-point placement calls the injected SketchCircleConstruction service,
backed by the released native circumcircle builder. Its options require an
explicit finite nonnegative point-residual bound in document working units.
Callers can convert the existing SketchConstructionPolicy to those units.
No view-dependent tolerance or widened retry is introduced. Seed coordinates and
radius pass unchanged into the recipe. Point order can change native rounding;
tests compare each recipe exactly with its native seed and check measured
geometric agreement across orders. The clicked points seed the circle; they do
not become permanent point-on-circle constraints, matching Go.

Independent Go fixtures now contain nine circle cases, plus the existing 28
slot and rectangle cases. Tests check geometry, diameter witnesses and bindings,
rank, formula-driven diameter changes, persistence recipe round trips, history,
invalid indices/radii and late-field rollback. Adapter tests check a closed
native preview and chord error. Head tests cover click/drag parity, no history
change during preview, failed point recovery, cancellation, accuracy policy,
native construction failures and ordered seed preservation.

The live native test drives both circle tools through preview, commit, Finish,
Undo and Redo. It checks the radius and centre identity against the preview,
retains prior geometry, and compares native captures with mapped X11 pixels.
The three-point circle capture after Finish and Redo was inspected visually:
`/tmp/circle-placement-native.png` (1440×900). A temporary test capture was removed
before the final gate. The blue background belongs to the existing test fixture.

The cross-platform review checked finite binary64 values, overflow refusal,
explicit recipe order, bounds-checked local indices and the unchanged physical
pixel drag threshold. Comments retain the Go rank rationale. No kernel algorithm,
public wire contract, new tolerance constant or local kernel override was added.
No kernel priority change is needed for this supported release capability.

Native menu/ribbon selection, field widgets, snapping, picking and other sketch
tools remain open. See [committed sketch presentation](implementation-sketch-presentation.md).
This step does not complete R3/R4 or the full port. Windows/macOS runtime
acceptance remains open. Work is local only.

The final local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests used Xvfb with software
Vulkan. Every crate exceeded 80% line coverage. Model reached 98.83%, the kernel
adapter 91.48%, viewport 94.17% and head 95.94%. The circle recipe, placement
geometry resolver and shared input state reached 100%; tool dispatch reached
98.96%. Duplication was 0.80% (587/73,266 scanned lines, 68 clone pairs), with no
additional duplicated lines or clone pairs relative to the wave base. Full
evidence is in `/tmp/circle-placement-gate-reviewed.log`. The result text was added
after that gate and checked with `make docs-lint`; no code changed afterward.
