# Polygon placement — partial R3/R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go PolygonTool, PolygonRecipe and polygonConstraints at
aa8297d1b0ac33c024f19e54821536d76e8a447a.
Kernel/API pins remain v0.2.5/v0.158.0 with no local source override.

The polygon tool now accepts a centre and a vertex through the shared mouse
placement path. The caller supplies the side count; six selects Go's default
hexagon. Counts below three are refused when the tool starts. The interactive
tool makes an inscribed polygon, as in Go. The recipe also supports the existing
circumscribed seed used by the typed API. Native side-count widgets remain open.

The shared recipe owns the ordered vertices, centre, closed boundary lines and
construction circumcircle. Each vertex lies on that circle; consecutive edges
have equal length. The n point-on-circle and n-1 equal-length relations leave
four DOF: centre, radius and orientation. A locked Diameter field creates an
owned parameter and dimension on the construction circle. Angle is a placement
steering field. The circle remains construction geometry in preview and commit.

The typed polygon path and placement recipe share the existing vertex generator,
one construction-radius calculation and one ordered regularity relation builder.
The old separate relation builder is removed. Typed API identity allocation and
reply shape are unchanged. Rectangle and polygon recipes also share the closed
boundary member builder. Preview and commit continue through the common recipe,
solver and native sampler; there is no new geometry owner or renderer.

Twenty independent Go recipe fixtures cover inscribed and circumscribed polygons
with 3, 4, 5, 8 and 17 sides, including translated and rotated inputs. The complete
recipe set now has 79 cases. Tests compare vertices, members, construction roles,
relations, field witnesses, bindings and four DOF. Trigonometric seeds and their
derived circle radii use the existing binary64 arithmetic bound; no bound was
widened and no bitwise cross-language trigonometric parity is claimed.

A placed five-sided polygon with fixed centre, orientation and formula-driven
diameter reaches zero DOF without redundant equations. Diameter edits retain
that rank, equal edges and point-on-circle incidence. Invalid side counts,
nonfinite points, zero/unrepresentable radii and unallocatable vertex counts are
refused. Existing API tests still check exact owned identities and constraint
references, recipe round trips, atomic reservation exhaustion and radius edits.

Head tests cover three side counts through click/drag input, unchanged history
during preview, sample agreement after commit, tool restart and Undo/Redo.
A collapsed radius retains the stage for retry; Escape cancels without a history
change. Invalid counts preserve the previously active tool. The native shared
placement test now covers polygons through preview, commit, Finish, Undo/Redo,
save, close and reopen. Captures are checked against mapped X11 pixels.

Native menu/ribbon selection, field widgets, snapping and picking remain open.
A regular polygon is not proof of a certified profile or general extrusion.
See [owned polygon creation](implementation-sketch-polygon.md) and
[sketch presentation](implementation-sketch-presentation.md).
This step does not complete R3/R4 or the full port. Windows/macOS runtime
acceptance remains open. Work is local only.

The capture after Finish and Redo was inspected visually:
`/tmp/polygon-placement-native.png` (1440×900). It shows the closed hexagon,
dashed construction circle, centre marker and prior rectangle. The blue
background is part of the test fixture. Temporary capture code was removed
before the final gate.

The cross-platform and comment review checked explicit vertex/relation order,
checked local indices, representable allocations and the shared radius arithmetic.
The Go radius and regularity rationale is retained. No new tolerance constant,
kernel algorithm, public wire contract or source override was added. No kernel
priority change is needed for this released capability.

The final local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests used Xvfb with software
Vulkan. Every crate exceeded 80% line coverage. Model reached 98.84%, head 96.05%,
the kernel adapter 91.66% and viewport 94.18%. The polygon recipe, shared seed,
regularity adapter, field/relation bindings and tool dispatch reached 100%.
Duplication was 0.79% (587/73,942 scanned lines, 68 clone pairs), with no added
duplicated lines or pairs relative to the wave base. Full evidence is in
`/tmp/polygon-placement-gate.log`. This result text was added after the gate and
checked with `make docs-lint`; no code changed afterward.
