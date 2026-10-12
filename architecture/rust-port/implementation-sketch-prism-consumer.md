# Checked sketch prism consumer — partial K02/R1/R3/R1V

Tracking: [#3563](https://github.com/Oblikovati/Oblikovati/issues/3563), under
[ADR-0007](../decisions/ADR-0007-martensite-integration.md),
[ADR-0004](../decisions/ADR-0004-session-and-feature-evaluation.md), and
[ADR-0008](../decisions/ADR-0008-vulkan-renderer.md).

## Current document inputs

`retain_selected_sketch_prism` consumes an immutable document snapshot, a stable
sketch identity, explicit directed line boundaries, and a local displacement.
The document resolves current placement and projected inputs and checks the
existing numerical verification. An absent, stale, failed, unconverged, or
under-constrained sketch cannot provide kernel input. The adapter does not
solve as a side effect or use an old successful body when these checks fail.

The boundary selection names authored line identities, not positions in a
catalogue. Reversal changes traversal without changing the source identity.
Each selected line must be a non-construction curve and occur once across all
boundaries. Each loop must close through shared authored endpoint identities.
The adapter reads point coordinates only from those selected sources, then
calls the existing model-plane prism path. It adds no polygon classification,
containment test, winding repair, native geometry algorithm, or second mesher.

Side-face construction roles retain the source line identities. This is still
diagnostic lineage, not the full generated/modified/deleted history or a
persistent feature reference key. Native shape handles remain private.

## Explicit limits

This input contract supports directed line boundaries with shared authored
endpoints. Distinct point identities with equal coordinates are refused before
kernel construction. Equality of coordinates cannot silently establish source
incidence. General profile arrangement, including constrained coincident points,
intersections, curved boundaries, region nesting and selection, belongs in
Martensite. It must consume source identities and achieved solve precision.
Required Go-kernel behavior outside OCCT remains part of that kernel work.

The consumer does not store an extrude feature recipe or a body in document
history. It does not infer a profile index, validate a complete tolerance budget,
publish API feature behavior, or complete save/reopen and undo/replay for bodies.
Those remain required work; this diagnostic consumer does not close R1 or R3.

## Local checks

The shared authored fixture lives in
`test-utilities/rust/selected_sketch_prism.rs`. Width and height are length
parameters. The rectangle has zero free variables and uses stable shared points
and lines. An inner loop uses fixed point constraints.

Tests cover current inputs, stale width edits, solve and recompute, unchanged
previous results, source line roles, reversed source direction, holes, and
model-plane changes. A linked work plane moves with its current parameter
without reauthoring local geometry; suppression refuses the next kernel call.
Malformed boundaries, construction lines, absent sources, distinct coincident
endpoint identities, absent verification, free variables, and failed solves
have named refusal tests. A kernel decline retains the source sketch identity.

The native Linux viewport test uses this same document fixture. It renders the
holed prism, changes width from 4 to 6, solves, and replaces the presentation.
The new frame must have a newer scene generation and more green solid pixels.
Blue pixels remain visible through the hole. Each frame must match actual X11
pixels and its GPU capture, and kernel Vulkan validation must report no errors.
This runs for XY, XZ, YZ, translated tilted, and flipped tilted placements.

```sh
cargo test --offline --locked -p oblikovati-martensite --test sketch_prism
cargo test --offline --locked -p oblikovati-viewport --test native
CARGO_NET_OFFLINE=true make gate
```

All checks stay on this Linux host. This is not a production feature replay or
foreground MCPBridge acceptance run. Native tools, persistent picks, complete
body validity, body history, and general profile extraction remain open.
