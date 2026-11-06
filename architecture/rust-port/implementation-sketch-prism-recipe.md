# Authored sketch prism inputs — partial R1/R2/R3

Tracking: [#3563](https://github.com/Oblikovati/Oblikovati/issues/3563), under
[ADR-0004](../decisions/ADR-0004-session-and-feature-evaluation.md),
[ADR-0007](../decisions/ADR-0007-martensite-integration.md), and
[ADR-0015](../decisions/ADR-0015-parameters-and-constraints.md).

## Model-owned sources

The model now owns `SketchPrismDefinition`, the directed line selection, and a
version-1 `SketchPrismRecipe`. The kernel adapter re-exports the selection types
and consumes the same model definition. It no longer declares those authored
types or repeats their line-count and unique-identity validation.

The definition stores a stable sketch identity, ordered outer and inner line
boundaries, and three scalar length sources for local X, Y and normal
displacement. Scalar sources reuse `DatumLength`: either an authored working-unit
constant or a stable parameter identity. There are no stored parameter callbacks,
derived coordinates, frames, solve reports, native handles, or body caches.

Construction validates recipe version, nonzero identities, unique selected
lines, boundary sizes, finite constants and nonzero constant normal displacement.
Current parameter values resolve as finite lengths in document working units.
A failed, missing, wrong-dimension or zero normal source produces a typed local
error. Its source sketch, displacement axis and underlying failure remain
available. Failed parameter evaluation cannot use a retained last-good value.

Exact authored comparison includes scalar source kind, parameter identity,
constant bits including signed zero, and directed boundary order. Recipe capture
and restore preserve those values without storing evaluation results.

## Current reads and kernel consumption

The definition resolves the current document sketch through the existing
placement and zero-free-variable check. It attributes the source sketch, current
placement and sketch dimension dependencies, plus each displacement parameter.
The common read footprint retains deterministic ordering and conservative read
status. The owner must refresh those reads when it adopts changed sources.

`retain_sketch_prism_definition` passes these current inputs to the same selected
line and model-plane kernel path. The earlier constant-input diagnostic calls
this path through a temporary validated definition. Neither path introduces an
alternate geometry engine or a new mesher.

## Local evidence

Model tests cover strict recipe round-trip, version and unknown-field refusal,
invalid identities, duplicate boundaries, scalar syntax, exact comparison,
current unit conversion and negative normal displacement. They check read
footprints, stale and missing sketches, linked placement changes, typed error
sources, failed depth, wrong dimensions, and zero current normal displacement.

The adapter restores the recipe from JSON and builds the actual kernel prism.
Changing depth from 2 cm to 30 mm changes analytic volume from 24 to 36 in the
fixture's working units. A later failed depth refuses construction while the
previous immutable result stays unchanged. The authored recipe remains equal.

The native viewport uses this parameter-backed definition for the existing
holed, fully constrained document fixture. Width edits still produce newer scene
generations and more solid pixels. Completed GPU captures agree with actual X11
pixels across the five checked model-plane placements.

```sh
cargo test --offline --locked -p oblikovati-model --test sketch_prism_recipe
cargo test --offline --locked -p oblikovati-martensite --test sketch_prism
cargo test --offline --locked -p oblikovati-viewport --test native
CARGO_NET_OFFLINE=true make gate
```

## Remaining owner integration

This is an authored kernel-input recipe, not the complete Go extrude definition.
It is not yet part of the document package or undo state. Body features must
join the common ordered feature program and permanent identity owner; they must
not create another independent feature graph. Join, cut, intersect, surface
operations, full extent modes, taper, region detection, curved profiles,
persistent body references and complete kernel result contracts remain required.
This evidence closes none of R1, R3 or the complete application port.

All checks remain on this Linux host. No kernel source, release package, remote
branch or platform pipeline is changed by this step.
