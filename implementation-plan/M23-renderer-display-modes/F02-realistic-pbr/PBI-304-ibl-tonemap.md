---
milestone: M23
feature: F02
pbi: PBI-304
title: Image-based lighting + tone mapping
status: planned
estimate: M
---

# PBI-304 — Image-based lighting + tone mapping

**Milestone:** M23 Renderer Display-Mode Parity & Realistic PBR  ·  **Feature:** F02 Realistic PBR Shading (software)

## Goal

Give Realistic a believable ambient response and a display-referred output: image-based
lighting for the GGX BRDF plus a tone-mapping stage — all analytic/precomputed, no path
tracer.

## Scope / work

- Provide an environment as **diffuse irradiance** + **prefiltered specular** (precomputed
  from a small built-in environment, or an analytic sky/ambient model if that is simpler
  and deterministic). The renderer receives it as bindless textures/uniforms through the
  app resolver seam — the renderer does not import `model` (ADR-0022 §6).
- Feed irradiance into the diffuse ambient and the prefiltered specular + a split-sum BRDF
  LUT into the specular ambient of the PBI-303 BRDF.
- Add a tone-map stage (ACES or Reinhard) + exposure control before sRGB encode.
- Keep everything reproducible under determinism mode (fixed environment, no temporal
  accumulation).

## API contracts (interfaces / enums / collections)

- (internal) environment/IBL inputs to the surface pass; BRDF LUT; tone-map stage params.

## Acceptance criteria

- Analytic ambient oracle: with a constant (uniform) environment, ambient response equals
  the closed-form irradiance/specular value within tolerance on `offscreen`.
- Metamorphic invariance: rotating both the environment and the object together leaves the
  shaded result invariant (within tolerance); rotating only the light changes highlights
  predictably.
- Tone-map stage is idempotent w.r.t. exposure=0 and monotonic in exposure; output stays
  in `[0,1]` (no NaNs/negatives), asserted on the captured AOV.
- Validation layers clean; determinism-mode output stable across runs.

## Depends on

PBI-303 (the BRDF the IBL feeds).
