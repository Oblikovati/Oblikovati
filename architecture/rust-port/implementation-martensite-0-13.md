# Released Martensite v0.13 adoption

## Release boundary

The authored sketch and native renderer dependencies target
[v0.13.0](https://github.com/Oblikovati/Martensite/releases/tag/v0.13.0), published
2026-11-07. Annotated tag `84adff83ae1e687599bbc09bc3f941f8160ed277` peels to
commit `d051ff699982faba4ec2df3ee9a47e14f38dedbd`.

All 15 consumed Martensite packages use that immutable source. The lockfile
changes only these 15 package identities; no API or third-party registry package
changes. Exact direct versions are read from the released package manifests.
The renderer source gate rejects the prior v0.11.0 source for both visualization
and sketch packages.

The separate RustyCAD v0.2.7 solid adapter remains in use. This update does not
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
