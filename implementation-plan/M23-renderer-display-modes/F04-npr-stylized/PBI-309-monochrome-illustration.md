---
milestone: M23
feature: F04
pbi: PBI-309
title: Monochrome (8713) + Illustration (8715)
status: planned
estimate: M
---

# PBI-309 — Monochrome (8713) + Illustration (8715)

**Milestone:** M23 Renderer Display-Mode Parity & Realistic PBR  ·  **Feature:** F04 NPR Stylized Modes

## Goal

The first two NPR modes, both built from the PBI-308 framework with deterministic shade
transforms — making them ideal CPU-reference oracle targets.

## Scope / work

- **Monochrome (8713)**: desaturate the shaded image to luminance, posterize to a small
  number of tone bands, composite the edge AOV as outlines. A single hue/neutral palette.
- **Illustration (8715)**: flat / cel shading (quantized diffuse, minimal specular) in
  full color with crisp composited outlines — the "cartoon/illustration" look.
- Both are `StylizeConfig`s over PBI-308; no new pipeline.
- Wire both `VisualStyle`s through the F01 resolver.

## API contracts (interfaces / enums / collections)

- (internal) `StylizeConfig` for Monochrome and Illustration; resolver `PassSet` entries
- No new public surface.

## Acceptance criteria

- **CPU-reference oracle** (pure Go) for the deterministic math — luminance desaturation,
  posterize/cel quantization, and outline composite — matches the GPU output within tight
  tolerance on `offscreen`.
- Monochrome output is neutral/single-hue (saturation ≈ 0 in the fills), asserted by
  sampling the AOV.
- Illustration output is banded (a bounded count of distinct shade levels per face),
  asserted numerically.
- Selecting each via `api/client` (PBI-300) and gallery (PBI-302) yields the expected
  `PassSet` and AOV. Validation clean; determinism stable.

## Depends on

PBI-308 (framework), F01 (resolver + selection).
