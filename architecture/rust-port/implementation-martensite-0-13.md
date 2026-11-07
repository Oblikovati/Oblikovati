# Released Martensite v0.13 adoption

## Release boundary

The authored sketch and native renderer dependencies target
[v0.13.0](https://github.com/Oblikovati/Martensite/releases/tag/v0.13.0), published
2026-11-07. Annotated tag `84adff83ae1e687599bbc09bc3f941f8160ed277` peels to
commit `d051ff699982faba4ec2df3ee9a47e14f38dedbd`.

At the first pin checkpoint, all 15 consumed Martensite packages use that immutable source. The lockfile
changes only these 15 package identities; no API or third-party registry package
changes. Exact direct versions are read from the released package manifests.
The renderer source gate rejects the prior v0.11.0 source for both visualization
and sketch packages.

At that checkpoint, the separate RustyCAD v0.2.7 solid adapter remains in use. It does not
claim that its B-rep or extrusion operations now use Martensite v0.13. The
`release.toml` modeling pin and its existing tests continue to describe that
adapter. Replacing it requires its own geometry, naming, and replay evidence.

## Consumed changes and design reference

The changed consumed runtime includes deterministic collection traversal,
periodic-wire healing, mesh triangle ordering, sketch-incidence and extrusion
predicates, validation-layer admission diagnostics, and the default-view-axis
accessor. New IGES, OBJ, and PLY providers remain unlinked; their release does not
establish an application exchange command.

The validation-layer minimum remains 1.4.313. Validation stays enabled during
native qualification. No renderer fallback or duplicated adapter was added.
The Go application remains the architecture and implementation reference, and
MCAD-Scenarios remains the behavior benchmark.

## Verification

The pin checkpoint passes all 76 build-tool tests and all 209 geometry adapter
unit/integration tests, including the per-face OCCT corpus, lineage/copy/replay,
analytic physical properties, and mesh-role checks. Formatting and Markdown
lint pass. The lockfile diff contains only the 15 Martensite packages.

Native qualification passes the 150 viewport unit tests, display-lifetime test,
all six GUI-render tests, and 24 of 25 native viewport integration tests. The
remaining test compares a regular point's preview and committed marker. The
preview currently forces a cross while committed regular points use a dot.
The case remains red; it is not waived and this checkpoint is not merge-ready.
The separate host correction must pass both regular and center-point cases.

The two focused native head tests pass: GUI save/restore of model pixels and
physical-keyboard Save routing. All owned windows are closed. Native commands
use Xvfb `-noreset` and the pinned 1.4.313 validation runtime. An initial outer
Xvfb invocation omitted `-noreset` and failed its lifetime check. Two later
private-server startup timeouts passed on isolated retry without code or
timeout changes; the successful first unit run remains separate evidence.

The full local gate, final integrated native acceptance, coverage, duplication,
and hosted checks remain required before integration or push. This record does
not claim that the old solid backend has been replaced.

A direct v0.13 ellipse query also confirms why measurements must request
convergence control: the same 8 mm/4 mm ellipse returns 38.804690497910002 mm
from `AbscissaPoint::length`, versus 38.753792882190709 mm from
`length_with_tolerance(1e-9)`. The general adapter call correction is separate
sketch work; no ellipse formula or numerical fallback was added here.

## Full model backend candidate

The second, unmerged candidate implements
[ADR-0036](../decisions/ADR-0036-martensite-model-backend.md). Its lockfile uses
17 Martensite packages from the same released revision and removes the entire
RustyCAD dependency graph. The project-owned adapter interfaces remain stable.
Primitive construction, analytic measurements, validation, copying, naming,
and meshing now compile against the released SDK types.

Release verification checks the official schema-2 package catalog, its SHA-256
against the GitHub asset digest, and the complete source-workspace archive's
revision and digest. It validates each independently versioned consumed crate.
The 0.13 release does not republish every unchanged component archive, so the
source-workspace catalog is the correct release boundary.

The candidate passes 81 build-tool tests, all 29 selected-sketch prism tests,
and two unique per-face oracle matching regressions. All-target Clippy passes
for the build tool, kernel adapter and viewport. The real preflight passes SDK
release verification, then stops at the missing packaging icon resource
`packaging/resources/icons/oblikovati.png`; this is not a successful full gate.

The full adapter run passes 211 tests across 15 binaries. Two strict geometry
checks remain red. The rotated-ellipse closest-point case
returns a boundary because the SDK marks every stationary ellipse root as a
non-minimum, tracked in
[Martensite issue 406](https://github.com/Oblikovati/Martensite/issues/406).
Surface integration on an extruded half ellipse reports achieved
relative error near machine precision while its area error is about 3.86e-5.
The surface integration defect is inherited from OCCT 8.0, which reproduces
the same incorrect result and error estimate at the same requested tolerance.
The independent high-precision analytic oracle exposes the defect in both.
The latter is tracked in
[Martensite issue 407](https://github.com/Oblikovati/Martensite/issues/407).
Both require upstream general fixes. The host adds no special-case formula,
fallback engine, altered geometry, or relaxed per-face oracle.

The per-face corpus now matches each independent oracle row exactly once by
its existing metric bounds. Equal-area faces must not be paired by incidental
floating-point sort order. Loop and edge assertions remain exact. Metric
centroid checks reuse the existing analytic scalar bound. A typed SDK alert
check replaces dependence on an OCCT diagnostic's textual spelling.

The point-role correction is included as a separate commit. The complete
model-backend candidate passes 150 viewport unit tests, the display-lifetime
test, six GUI-render tests, and all 26 native viewport tests, including both
point roles. The native suite uses serial execution on its isolated Xvfb display.
An accidental concurrent run failed nine overlapping-window pixel comparisons;
the serial rerun passes without product or timeout changes. The candidate is
not merge-ready and does not claim a completed backend migration.

The broad native head run was intentionally stopped during application BDD.
The external source contains steps this lane does not implement, including the
populated drawing-template step at `document-creation.feature:333`. No clean-base
causality claim is made. Application acceptance belongs to the active application
lane and must run in the final integrated gate. Focused non-BDD native head
compatibility checks are separate evidence, not a waiver of that requirement.

All seven non-BDD native head GUI regressions pass on the complete backend
candidate. These include physical-keyboard Save, save/restore model pixels,
parameter-driven rectangle authoring with edit/history/save, resize, camera
routing, browser timing, and save-before-first-frame behavior. Six text/pixel
diagnostic tests also pass. The broader filtered command reports one additional
sketch-BDD fixture-admission failure because this lane's retained benchmark
snapshot lacks `features/sketch`; no sketch scenario ran in that command.
These scoped results do not establish full application or sketch acceptance.
