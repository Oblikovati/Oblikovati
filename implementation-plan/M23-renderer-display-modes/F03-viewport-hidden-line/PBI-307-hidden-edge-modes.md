---
milestone: M23
feature: F03
pbi: PBI-307
title: Dashed hidden-edge pass + the three HLR modes wired
status: planned
estimate: M
---

# PBI-307 — Dashed hidden-edge pass + the three HLR modes wired

**Milestone:** M23 Renderer Display-Mode Parity & Realistic PBR  ·  **Feature:** F03 Viewport Hidden-Line Visibility

## Goal

Draw occluded edges in a distinct dashed style and assemble the three hidden-edge display
modes from the visible/hidden classification.

## Scope / work

- A **dashed line pass** that renders the hidden spans from PBI-306 in a different line
  style (dashed, dimmed) from the solid visible-edge pass.
- Compose the three modes through the F01 resolver's `PassSet`:
  - **8711 `kWireframeNoHiddenEdges`** — visible edges only (no faces, hidden spans
    dropped).
  - **8712 `kWireframeWithHiddenEdgesRendering`** — visible edges solid + hidden spans
    dashed (no faces).
  - **8707 `kShadedWithHiddenEdgesRendering` / `kHiddenEdgeRendering`** — shaded faces +
    hidden spans dashed over them.
- Each mode writes a capturable AOV so it is diffable in isolation (ADR-0014).

## API contracts (interfaces / enums / collections)

- (internal) dashed hidden-edge pass; resolver `PassSet` entries for 8707/8711/8712
- No new public surface.

## Acceptance criteria

- Per-mode AOV capture of the occluded-box scene matches the analytic edge oracle: 8711
  shows only visible edges, 8712 adds dashed hidden edges, 8707 adds them over shaded
  faces — each verified against the computed visible/hidden segment sets.
- Selecting each mode via `api/client` (PBI-300) and the gallery (PBI-302) produces the
  expected `PassSet` (null-backend) and AOV (offscreen).
- Dashed style is visually distinct from solid and dimmed (asserted by sampling the AOV,
  not by eye).
- Validation layers clean; determinism mode stable.

## Depends on

PBI-306 (visibility classification), F01 (resolver + selection).
