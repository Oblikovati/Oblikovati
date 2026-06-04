---
milestone: M23
feature: F04
pbi: PBI-310
title: Technical Illustration (8716) — Gooch + edges
status: planned
estimate: M
---

# PBI-310 — Technical Illustration (8716) — Gooch + edges

**Milestone:** M23 Renderer Display-Mode Parity & Realistic PBR  ·  **Feature:** F04 NPR Stylized Modes

## Goal

The `kTechnicalIllustrationRendering` (8716) mode: Gooch cool-warm shading with
emphasized edges — the classic technical-illustration look where shape reads from a
warm→cool tone shift rather than light/dark.

## Scope / work

- **Gooch shading**: map `N·L` from a cool tone (shadow) to a warm tone (lit) blended
  with object color, instead of conventional diffuse falloff — a deterministic shade
  transform over PBI-308.
- Emphasized **edges**: strong silhouette outlines + lighter interior creases from the
  edge AOV; optional simple hatching in the darkest cool regions.
- Wire the `VisualStyle` through the F01 resolver.

## API contracts (interfaces / enums / collections)

- (internal) Gooch `StylizeConfig`; resolver `PassSet` entry for 8716
- No new public surface.

## Acceptance criteria

- **CPU-reference Gooch oracle** (pure Go): the cool→warm interpolation across a
  `N·L` sweep matches the GPU output within tight tolerance on `offscreen`.
- Metamorphic: rotating the object rotates the tone gradient consistently (the warm
  direction tracks the light), asserted numerically.
- Silhouette edges are present and stronger than interior creases (sampled from the AOV).
- Selecting via `api/client` (PBI-300) and gallery (PBI-302) yields the expected `PassSet`
  and AOV. Validation clean; determinism stable.

## Depends on

PBI-308 (framework), F01 (resolver + selection).
