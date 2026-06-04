---
milestone: M23
feature: F02
pbi: PBI-303
title: GGX metallic-roughness BRDF + CPU-reference oracle
status: planned
estimate: L
---

# PBI-303 — GGX metallic-roughness BRDF + CPU-reference oracle

**Milestone:** M23 Renderer Display-Mode Parity & Realistic PBR  ·  **Feature:** F02 Realistic PBR Shading (software)

## Goal

Replace the Blinn-Lambert surface shading with a physically based GGX metallic-roughness
BRDF that consumes the M19 appearance scalars already plumbed to `renderer.Surface`.

## Scope / work

- Implement a GGX/Trowbridge-Reitz specular BRDF with Smith geometry and a
  Fresnel-Schlick term, metallic-roughness parameterization (dielectric F0 = 0.04 lerped
  to albedo by metallic), Lambertian diffuse weighted by `(1 − metallic)`, plus emissive
  and opacity — in `head/internal/native/shaders/mesh.frag` (and `mesh.vert` if normals/
  tangents need passing). All lighting math in **linear** space; encode sRGB at output.
- Direct lighting from the existing light inputs (single headlight is fine for this PBI;
  multi-light is PBI-305). Ambient is a flat term here; IBL is PBI-304.
- Write a **pure-Go CPU-reference** implementation of the same BRDF (reusing `math/`,
  no external dependency) — the lit-pass oracle per ADR-0014 item 4.

## API contracts (interfaces / enums / collections)

- (internal) extended surface shader; CPU-reference shading function in the renderer's
  pure layer.

## Acceptance criteria

- On the `offscreen` backend, the GGX lit pass matches the CPU-reference oracle within a
  tight tolerance for a sphere/known surfels across a roughness/metallic sweep
  (dielectric matte, dielectric glossy, metal rough, metal smooth).
- Energy sanity: increasing roughness broadens/dims the highlight monotonically; metallic
  surfaces tint specular by albedo and kill diffuse — asserted numerically, not visually.
- All renderer tests run with validation layers on (no messages) and in determinism mode.
- Existing Shaded/ShadedWithEdges goldens are re-baselined intentionally (the shading
  model changed) and documented as such.

## Depends on

M19 (scalars on `renderer.Surface`), F01 (Realistic `VisualStyle` exists to target).
