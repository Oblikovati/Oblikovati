---
milestone: M23
feature: F02
name: Realistic PBR Shading (software)
status: planned
---

# M23 · F02 — Realistic PBR Shading (software)

The `kRealisticRendering` (8709) mode: a **PBR-capable software** shading path. The
appearance scalars (albedo, metallic, roughness, emissive, opacity) already reach
`renderer.Surface` from M19 (ADR-0022 §2) but the shader is Blinn-Lambert and uses only
albedo. This feature adds the GGX metallic-roughness BRDF, image-based ambient, and tone
mapping that consume the rest — **raster, no hardware ray tracing** (ADR-0023 §2).

## In scope

- A GGX metallic-roughness BRDF replacing Blinn-Lambert in the surface shader, in linear
  space with sRGB output.
- Analytic / prefiltered image-based lighting (irradiance + specular) for ambient.
- Tone mapping (ACES/Reinhard) + exposure.
- Multi-light from M16 F03 lighting styles, with a default headlight + ambient fallback.
- The wired Realistic `VisualStyle`, validated by a CPU-reference GGX oracle and a
  Blender perceptual golden.

## Out of scope

- **Image-texture maps** (albedo/normal/roughness maps) — deferred (ADR-0022 §2);
  Realistic uses scalar/color PBR only.
- Hardware ray tracing, real GI/path tracing.
- The lighting/camera **model** (M16) — consumed here, not built.

## Key API contracts delivered

- (internal) `renderer` GGX surface pass; environment/IBL inputs; tone-map stage
- No new public surface (display-mode selection is F01).

## Depends on

M19 (appearance scalars on `renderer.Surface`, [app/materials.go](../../../app/materials.go)),
M16 F03 (lighting styles; fallback rig otherwise), the surface/normal passes
([architecture/core/08-renderer-vulkan.md](../../../architecture/core/08-renderer-vulkan.md)).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-303](PBI-303-ggx-brdf.md) | GGX metallic-roughness BRDF + CPU-reference oracle |
| [PBI-304](PBI-304-ibl-tonemap.md) | Image-based lighting + tone mapping |
| [PBI-305](PBI-305-realistic-mode-wiring.md) | Realistic mode wiring + multi-light + Blender golden |
