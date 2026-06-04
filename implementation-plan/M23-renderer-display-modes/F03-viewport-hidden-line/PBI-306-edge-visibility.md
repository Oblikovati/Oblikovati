---
milestone: M23
feature: F03
pbi: PBI-306
title: Per-frame tessellated edge-occlusion classification + analytic oracle
status: planned
estimate: L
---

# PBI-306 — Per-frame tessellated edge-occlusion classification + analytic oracle

**Milestone:** M23 Renderer Display-Mode Parity & Realistic PBR  ·  **Feature:** F03 Viewport Hidden-Line Visibility

## Goal

Classify each model edge segment as visible or hidden per frame by testing it against the
tessellated mesh depth, fast enough for real-time viewport use.

## Scope / work

- Render a **depth prepass** of the tessellated bodies (reuse the existing depth pass).
- For each model edge (from `ops.TessellateBody`'s edge output), sample/compare segment
  depth against the depth buffer with a bias to classify spans visible vs occluded;
  produce per-segment visibility (visible span list + hidden span list). Handle the
  self-occlusion bias so an edge on a front face is not falsely hidden by its own face.
- Keep the classifier **above the GPU line** where possible: the comparison logic is a
  pure function of (edge samples, depth samples) — unit-testable against a supplied depth
  buffer with the `null`/CPU path (ADR-0014).
- Cache per body/camera so a static frame does not re-classify (invalidate on
  camera/tessellation change).

## API contracts (interfaces / enums / collections)

- (internal) `renderer` edge-visibility classifier: `classify(edges, depth, cam) →
  visibleSpans, hiddenSpans`

## Acceptance criteria

- **Analytic occlusion oracle**: for known geometry (a small box fully behind a larger
  box; two overlapping boxes), the visible/hidden classification matches the analytically
  computed expectation exactly (the geometry makes the answer computable).
- Self-occlusion: edges of a single convex body facing the camera are all classified
  visible (no false hides from depth bias); back edges are hidden.
- The pure comparison function is unit-tested on a hand-built depth buffer with no GPU.
- Validation layers clean; determinism mode stable.

## Depends on

M07 (edges/tessellation), the depth pass, F01 (the modes that consume this).
