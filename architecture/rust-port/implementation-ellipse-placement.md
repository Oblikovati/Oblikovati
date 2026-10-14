# Ellipse placement — partial R3/R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go EllipseTool, PendingRecipe and EllipseRecipe at
aa8297d1b0ac33c024f19e54821536d76e8a447a.
Kernel/API pins remain v0.2.5/v0.158.0 with no local source override.

The ellipse tool now accepts centre, major-axis endpoint and a point that sets
the minor radius. Only the perpendicular distance from that point to the major
axis sets the minor radius. Moving along the axis or to its other side does not
change the intended radii. The shared mouse pipeline supplies click/drag input,
preview, failure recovery, Escape, atomic commit and tool restart.

The recipe owns one centre and one ellipse member. It uses the existing ellipse
frame validation and scaled axis normalization. Radii must be finite with
major >= minor > 0; equal radii retain ellipse identity. Zero or nonfinite axes,
zero radii, reversed radii and overflowing witnesses are refused before adoption.
The existing validated model refuses reversed radii; the tool does not silently
swap the frame or install a new tolerance. Scaled normalization can differ from
Go's unit seed by a rounding step. The oracle uses the existing arithmetic bound
for axes and witnesses; no comparison bound was widened.

The Major field binds to the existing ellipse-radius dimension and parameter.
The Minor field is a length steering input, matching the pinned Go recipe; it
creates no live dimension. The separate minor-radius dimension API remains
available. No duplicate ellipse geometry, renderer, persistence format or kernel
algorithm is introduced. Scratch preview and commit use the same recipe path,
document solver settings and released native ellipse sampler.

The fixture generator now records twelve Go ellipse cases: four axis directions
and unequal, equal and thin radii. The full set contains 59 recipes. Tests check
geometry, field witnesses and bindings, five free DOF, invalid input refusal,
major-radius formula edits and unchanged minor radius/frame. A placed ellipse
with fixed centre, axis alignment, major and minor dimensions reaches zero DOF
without redundant equations and remains fully constrained after radius edits.
Adapter tests check a closed preview against the analytic ellipse equation and
increased sample density for a smaller deflection.

Head tests check point order, perpendicular distance, input refusal, click/drag
agreement, preview/commit samples, retry, cancellation, tool restart and history.
The live native test includes the ellipse alongside circles, arcs and rectangles.
It checks preview, commit, Finish, Undo/Redo, save, close and reopen, and compares
native captures with mapped X11 pixels. Native menu selection, field widgets,
snapping and picking remain open. See
[ellipse model and API](implementation-ellipse.md) and
[sketch presentation](implementation-sketch-presentation.md).
This step does not complete R3/R4 or the full port. Windows/macOS runtime
acceptance remains open. Work is local only.

The capture after Finish and Redo was inspected visually:
`/tmp/ellipse-placement-native.png` (1440×900). The ellipse is closed and
continuous, and the prior rectangle remains visible. The blue background is part
of the test fixture. Temporary capture code was removed before the final gate.

The cross-platform and comment review checked finite coordinates, explicit
point order, validated radius order, bounds-checked indices and shared frame
normalization. The Go minor-field rationale is retained. No new tolerance
constant, kernel algorithm, public wire contract or source override was added.
No kernel priority change is needed for this released capability.

The final local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests used Xvfb with software
Vulkan. Every crate exceeded 80% line coverage. Model reached 98.84%, head 96.03%,
the kernel adapter 91.66% and viewport 94.18%. The ellipse recipe, input
conversion, shared geometry resolver and field bindings reached 100%.
Duplication was 0.79% (587/73,842 scanned lines, 68 clone pairs), with no added
duplicated lines or pairs relative to the wave base. Full evidence is in
`/tmp/ellipse-placement-gate.log`. This result text was added after the gate and
checked with `make docs-lint`; no code changed afterward.
