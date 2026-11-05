# Arc placement — partial R3/R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go arc tools, ArcRecipe and commitRecipeEntities at
aa8297d1b0ac33c024f19e54821536d76e8a447a.
Kernel/API pins remain v0.2.5/v0.158.0 with no local source override.

Centre-point and three-point arcs now use the shared mouse placement path.
The centre-point tool accepts centre, start and end. The three-point tool
accepts start, end and then a point on the arc, as in Go. It passes start,
through and end to the released native circumcircle service. Both tools support
click and drag input, preview, retry, cancellation, commit and restart.

The centre-point tool uses the released exact orientation predicate to select
winding. An exact semicircle uses clockwise winding, as in Go. For collinear
inputs, ordered coordinates distinguish opposite rays without an epsilon or a
rounded cross product. Zero radii, equal endpoints and zero sweeps are refused.
The app does not introduce a kernel construction algorithm.

The arc recipe owns a centre and two endpoints. It retains intrinsic circularity
without adding a duplicate equation. It has a radius field and a sweep steering
field, with the same witnesses and signed angle difference as Go. A locked radius
uses the existing parameter and dimension path. Fixing the centre and the two
radial directions with a radius dimension gives zero DOF. Parameter expressions
can change that radius through the existing solver.

Preview now solves the scratch recipe before native sampling. This is required
when the centre-point tool receives endpoints at different radii. Preview carries
the document's solver settings; changing those settings invalidates cached
preview geometry. A failed solve reports an error and preserves the prior valid
view. Scratch history and identifiers remain separate from document history.
The existing default-policy sampling entry point delegates to this same path.

Three-point recipes retain a temporary construction check. After solve, both
preview and commit verify the actual circle or arc against the original points
with the caller's physical residual bound. This also applies to three-point
circles. Commit verifies before adoption; a refusal leaves history unchanged,
keeps a caller-owned command group open and retires allocated identifiers.
The shared read index supplies solved circular geometry to both placement and
the existing typed three-point API. These checks are creation-time obligations,
not saved point-on-curve constraints. Editable field widgets remain open;
changing a three-point radius at creation must not silently discard the check.

The Go fixture set now has 47 recipes, including ten arcs. Free arcs have five
DOF. The tests compare seeds, fields, rank and circularity. They compare solved
coordinates for already-circular seeds, which need no movement. For unequal
radii, a free arc has no unique solved coordinates. Rust and Go can reach different
valid free solutions because of the documented
[solver scaling](implementation-solver.md). Eight fully constrained fixtures
supply unique solutions and compare radius edits from 5 to 7 to 1.5 against Go
at the original coordinate bound. No parity tolerance was widened.

Adapter tests cover exact winding at small and large scales, invalid inputs,
post-solve verification, failed solves and non-default iteration limits. Owner
tests cover point order, click/drag agreement, retry, cancellation, solver policy,
preview/commit sample agreement and history. A native viewport test checks cache
invalidation when only solver policy changes. Session tests check successful
verification, Undo/Redo and rollback inside a caller-owned group.

The live native test drives both arc tools through preview, commit, Finish,
Undo/Redo, save, close and reopen. It compares solved preview samples with the
committed curves and checks captures against mapped X11 pixels. It also retains
the existing circle and rectangle cases. Native menu selection, field widgets,
snapping and picking remain open. See
[circle placement](implementation-circle-placement.md) and
[sketch presentation](implementation-sketch-presentation.md).
This step does not complete R3/R4 or the full port. Windows/macOS runtime
acceptance remains open. Work is local only.

Both arc captures after Finish and Redo were inspected visually:
`/tmp/arc-center-native.png` and `/tmp/arc-three-point-native.png` (1440×900).
The curves are continuous and the prior rectangle remains visible. The blue
background is part of the test fixture. Temporary capture code was removed
before the gate.

The cross-platform and comment review checked finite coordinates, exact
orientation, deterministic point order, checked indices, explicit solve policy
and retained Go references. No kernel algorithm, public wire contract, tolerance
constant or source override was added. No kernel priority change is required.

The final local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests used Xvfb with software
Vulkan. Every crate exceeded 80% line coverage. Model reached 98.83%, session
98.65%, the kernel adapter 91.66%, viewport 94.18% and head 96.00%. The arc recipe,
circular verification, exact direction adapter and tool dispatch reached 100%.
Duplication was 0.80% (587/73,685 scanned lines, 68 clone pairs), with no added
duplicated lines or pairs relative to the wave base. Full evidence is in
`/tmp/arc-placement-gate.log`. This result text was added after the gate and
checked with `make docs-lint`; no code changed afterward.
