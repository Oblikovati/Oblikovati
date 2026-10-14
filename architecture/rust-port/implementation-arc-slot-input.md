# Curved slot mouse input — partial R3/R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go CenterPointArcSlotTool, ThreePointArcSlotTool and ArcSlotRecipe at
aa8297d1b0ac33c024f19e54821536d76e8a447a.
Kernel/API pins remain v0.2.5/v0.158.0 with no local source override.

Both curved-slot tools now use the shared mouse placement path. The centre-point
tool accepts centre, start and end direction. The three-point tool accepts start,
end and then through, matching Go's corrected order (#2028). The caller supplies
the width. Click and drag, preview, retry, Escape, commit, restart and history
use the existing owner path. Invalid widths, circular-construction bounds,
collinear three-point input and a nonpositive inner radius are refused.

Centre-point winding uses the existing released exact orientation adapter.
Three-point construction uses the same released circumcircle builder as circles
and arcs. All curved slots use the existing four-arc recipe and four independent
pinned tangent relations. No extra centreline entity or cap equality relation is
added. Existing tests retain six free DOF and fully constrained parameter-driven
radius and width edits.

Three-point verification now reads the actual solved centreline from the outer
arc's centre and the two cap centres, with the outer arc's winding. The existing
arc radius rule derives its radius. Preview and commit pass that value to the
injected native verifier using the caller's physical residual bound. This checks
input incidence, endpoint displacement and winding before adoption. A wrong cap
reference, changed centreline or verifier refusal leaves the document unchanged.
These are temporary construction obligations, not saved constraints.

## Off-radius seed defect

The new centre-point mouse test exposed a cap collapse for centre `(0,0)`, start
`(10,0)`, end cursor `(5,5)` and width `2`. The minimal reproduction is
`cargo test -p oblikovati-martensite --test placement_preview off_radius_arc_slot`.
Before the fix, native sampling refused the collapsed end cap as a full-turn
span. The initial cap centre was the raw cursor, while both cap endpoints had
already been placed around the radius set by the start. That inconsistent seed
selected an internal inner-arc/cap tangent branch instead of external contact.

A direct pinned Go Apply/Solve probe also converged to collapsed width for that
input. Evidence is in `/tmp/arc-slot-go-probe.log`; Rust seed, branch and solved
coordinates are in `/tmp/arc-slot-rust-probe.log`. Changing only the authored end
cap centre to the start-radius ray selected the proper branch and left the seed
unchanged by solve (`/tmp/arc-slot-projected-probe.log`). This rejected a sampler
fault and isolated the input rule from solver behavior.

The shared arc-slot seed now uses one centreline radius for both caps. The last
cursor selects the end direction; the seed constructs its cap centre at the
start radius before allocating geometry or selecting tangent branches. This is
an authoring rule, not a repair of solved geometry. It applies to both the raw
slot constructor and the constrained recipe. Already-equal radii retain the
original endpoint bits, including signed zero. No widened solve tolerance,
retry, perturbation or alternate solver was added. The collapsed Go result is
not retained as intended behavior.

Regression tests cover four cursor radii in four directions and both windings,
cap radius equal to half-width, original coordinate bits for equal-radius input,
and the original native sampling failure. The 20 Go slot seed fixtures and all
79 placement recipe fixtures still pass. The test seam reaches the actual
shared seed and native preview; temporary debug code was removed. Testing only
equal-radius fixtures had missed this invariant.

## Native and remaining acceptance

Owner tests cover both tools through click/drag, preview/commit sample agreement,
restart and Undo/Redo. Session tests verify actual cap-centre reads and atomic
refusal. The native shared test covers preview, commit, Finish, Undo/Redo, save,
close and reopen. Captures are compared with mapped X11 pixels.

Both captures after Finish and Redo were inspected visually:
`/tmp/arc-slot-center-native.png` and `/tmp/arc-slot-three-point-native.png`
(1440×900). Both show closed, continuous slot boundaries and the prior rectangle.
The blue background is part of the test fixture. Temporary capture code was
removed before the final gate.

Native menu/ribbon selection, width widgets, snapping and picking remain open.
These outlines do not prove general profile-to-solid construction. See
[slot recipes](implementation-slot-placement-recipes.md) and
[arc placement](implementation-arc-placement.md).
This step does not complete R3/R4 or the full port. Windows/macOS runtime
acceptance remains open. Work is local only.

The cross-platform and comment review checked finite and representable radii,
explicit member order, bounds-checked cap indices, stable winding and the
unchanged equal-radius arithmetic. Go provenance comments remain. No kernel
algorithm, public wire contract, tolerance constant or source override was
added. No kernel priority change is needed for this released capability.

The final local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests used Xvfb with software
Vulkan. Every crate exceeded 80% line coverage. Model reached 98.84%, head 96.12%,
the kernel adapter 91.66% and viewport 94.18%. The corrected curved-slot seed,
centreline reader, circular recipe dispatch and tool dispatch reached 100%.
Duplication was 0.79% (587/74,165 scanned lines, 68 clone pairs), with no added
duplicated lines or pairs relative to the wave base. Full evidence is in
`/tmp/arc-slot-input-gate.log`. This result text was added after the gate and
checked with `make docs-lint`; no code changed afterward.
