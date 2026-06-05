---
milestone: M20
feature: F01
name: Intersecting Booleans
status: in-progress
---

# M20 · F01 — Intersecting Booleans

The general solid/solid boolean that splits faces along the intersection curve —
the Phase-C kernel op M07·PBI-082 deferred. This is the single biggest unblocker:
Cut, Split, Hole, Combine, Emboss, and most sheet-metal/plastic features all reduce
to "intersect two bodies and keep/discard regions."

## In scope

- Surface–surface intersection segments → an imprint of split edges on both bodies.
- Region classification (inside/outside/on) and union/subtract/intersect assembly.
- Lineage on every new face/edge so reference keys survive recompute.
- Completion of `CombineFeature`/`SplitFeature`/`CutFeature` to real geometry.

## Out of scope

- Tolerant near-coincident sew (Phase-D, stays `ops.Sew`).

## Key API contracts delivered

- `ops.Boolean` extended past Phase-A; `CutFeature(s)`/`CutDefinition`; `SplitFeature`
  real geometry.

## Depends on

M07 (`kernel/ops`, `math/predicate`).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-171](PBI-171-boolean-face-split.md) | Face-splitting solid/solid boolean |
| [PBI-172](PBI-172-cut-split-combine-geometry.md) | Cut / Split / Combine real geometry |
| [PBI-199](PBI-199-boolean-partial-penetration.md) | Boolean robustness — partial penetration & concave-wall crossing |

## Known robustness gap (2026-06-05)

The pipeline is correct for **clean through-overlaps** but not for **partial penetrations** or
**concave faceted-wall crossings** — the 2D arrangement only cuts a face when imprint segments
close a loop *on it*, so a tool that pokes part-way in / crosses a re-entrant wall leaves
dangling segments and the overlap is left un-cut (non-manifold / inverted normals). Tracked as
PBI-199; reproduced by the skipped fan e2e. A precondition (twisted-loft warped quads) was fixed
in PBI-174.
