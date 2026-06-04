---
milestone: M23
feature: F04
pbi: PBI-308
title: NPR edge-detection AOV + stylization compositor
status: planned
estimate: M
---

# PBI-308 — NPR edge-detection AOV + stylization compositor

**Milestone:** M23 Renderer Display-Mode Parity & Realistic PBR  ·  **Feature:** F04 NPR Stylized Modes

## Goal

Build the shared NPR substrate every stylized mode configures: a screen-space
edge-detection AOV and a stylization compositor that takes a shaded image + the edge AOV
and emits the final styled image.

## Scope / work

- An **edge-detection pass** deriving outlines from the existing AOVs: silhouettes from
  depth discontinuities, creases from normal discontinuities, and material/object
  boundaries from the ID buffer. Output an edge-strength AOV (capturable in isolation).
- A **stylization compositor** with pluggable per-mode configuration (shade transform +
  outline composite + optional paper/hatch texture), so each NPR mode is a config, not a
  new pipeline.
- Keep the edge extraction a **pure function** of the input AOVs where feasible
  (CPU-testable), per ADR-0014.

## API contracts (interfaces / enums / collections)

- (internal) `renderer` NPR edge AOV; `StylizeConfig` consumed by the compositor

## Acceptance criteria

- **Analytic silhouette oracle**: for known geometry (a cube against empty background)
  the edge AOV marks exactly the silhouette/crease pixels expected from the depth/normal
  discontinuities — bit-comparable on the deterministic extraction.
- The compositor with an identity config reproduces the input shaded image (no-op
  correctness).
- Edge extraction's pure core is unit-tested on hand-built depth/normal/ID inputs, no GPU.
- Validation layers clean; determinism mode stable (no temporal jitter in the edges).

## Depends on

The depth/normal/ID passes, F01 (the NPR styles that consume this).
