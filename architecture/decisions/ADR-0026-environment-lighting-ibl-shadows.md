# ADR-0026 — Environment Lighting, IBL Reflections & Shadows

**Status:** proposed (2026-06-04) · **Relates to:**
[ADR-0005](ADR-0005-vulkan13-renderer.md) (Vulkan renderer),
[ADR-0014](ADR-0014-renderer-testability.md) (pure draw list / oracle hierarchy),
[ADR-0018](ADR-0018-apache-api-contract-module.md) (API contract split),
[ADR-0022](ADR-0022-materials-appearances.md) (PBR appearance data),
[ADR-0023](ADR-0023-viewport-display-modes.md) (display modes / Realistic PBR §6).
Drives milestone **M16 F03** (PBI-155 Lights & lighting styles) and the IBL
refinement the surface shader reserved as **PBI-304**.

## Context

The Realistic display mode (ADR-0023 §2) runs a GGX metallic-roughness BRDF in
`head/internal/native/shaders/mesh.frag`, but it is lit by a single **hardcoded
headlight** (`const vec3 LIGHT_DIR`) with a **constant analytic ambient**
(`amb = lin*0.18 + f0*0.08`) the shader itself labels the *"image-based-lighting
stand-in (PBI-304 will refine)"*. Consequences:

- metallic/smooth bodies reflect nothing — there is no environment to reflect;
- there is no real ambient/sky term, so cavities read flat;
- there are no shadows; and
- nothing — neither the UI nor an add-in — can control lighting.

ADR-0023 §6 already committed the direction: *"Lights and environment are supplied
by the app through the existing resolver seam … Multi-light Realistic consumes M16
F03 lighting styles when present, and falls back to a default headlight + ambient
rig otherwise."* Inventor's reference object model (`Light`, `Lights`,
`LightingStyle`, and `View`'s shadow toggles) is the parity target.

### The architectural cost

The offscreen viewport (`head/internal/native/viewport.cpp`) today binds **only
push constants — zero descriptor sets** for the scene, runs **one** color+depth
pass, and clears to a flat themed color. Image-based lighting, a visible sky, and
shadows each need GPU machinery the viewport has never had: sampled textures
(hence descriptor sets), precompute passes, a background pass, and a depth pass
from the light's point of view.

## Decision

1. **The renderer emits lighting as pure data; the native layer owns the GPU.**
   A new `renderer.SceneLighting` (lights, ambient/brightness/exposure, an
   `Environment` IBL reference, and `ShadowSettings`) travels the same app→renderer
   seam as `SurfaceLookup` (ADR-0022 §6, ADR-0023 §6). It is plain `[N]float32`
   data, unit-tested on the null/CPU path (ADR-0014). The renderer never loads a
   file or touches Vulkan; lighting styles are a **preset table + gallery**, exactly
   like `VisualStyle` (`renderer/visualstyle.go`).

2. **The public API mirrors Inventor (ADR-0018).** `Light`/`LightingStyle`
   contracts, the `LightType`/`LightDefinitionType`/`LightingStyleType`/
   `ShadowDirection`/`GroundShadow` enums (Inventor's exact frozen ids), the wire
   methods/DTOs, and the typed client all live in the Apache-2.0 `api` module; the
   GPL app aliases the enums and satisfies the contracts. A public
   `LightingStyleEnum` ⇄ internal preset bijection lives in the app, exactly like
   `app/display_mode.go`.

3. **IBL is the split-sum approximation, precomputed once per environment.** At
   environment load the native layer builds, from an equirectangular source: an
   environment **cubemap**, a diffuse **irradiance** cubemap, a roughness-mipped
   **prefiltered** specular cubemap, and a one-time environment-independent **BRDF
   integration LUT**. The Realistic shader replaces its `amb` stand-in with
   `diffuse = irradiance(N)·albedo·kd` + `specular = prefiltered(R,rough)·(F·lut.x +
   lut.y)`. These bind through a descriptor set added to the mesh pipeline.

4. **The environment is either a built-in preset or a user file.** Presets are
   generated procedurally (hemisphere/studio/outdoor gradients) — no file I/O, so
   the default path stays headless and dependency-free. A user `.hdr`
   (Radiance RGBE) file is decoded behind a thin wrapped loader
   (`head/internal/native/envimage`, the "wrap third-party libs" rule); `.exr` is a
   later impl of the same interface. Both yield float pixels the native layer
   uploads and precomputes identically.

5. **A skybox pass draws the environment as the background.** A cube-sampled
   fullscreen pass renders the environment (rotated/scaled per `Environment`) when
   one is active, superseding the flat `obk_viewport_set_clear` color so reflections
   match the visible sky. With no environment, the themed clear color is unchanged.

6. **Shadows are a sun shadow map + ground receiver + screen-space AO.** A
   depth-only pass from the primary directional light feeds PCF **object shadows**
   in the surface shader (light-space matrix in the scene UBO). A ground plane
   receives **ground shadows** (honoring `GroundShadowEnum` None/Ground/XRay) and
   optional ground reflections. **Ambient shadows** are screen-space AO from the
   depth buffer modulating the IBL ambient term. The `View` toggles
   (`ShowGroundShadows`/`ShowObjectShadows`/`ShowAmbientShadows`) and
   `ShadowDensity`/`Softness`/`Direction` drive them. Hardware ray tracing stays out
   of scope (ADR-0023 §2).

7. **Backward compatibility: the Default style reproduces today's look.** The
   default `SceneLighting` is one directional headlight (dir ≈ (0.4, 0.6, 0.8),
   intensity 3) with ambience 0.18 and no environment/shadows — the current
   hardcoded values — so an un-themed, un-configured Realistic frame is unchanged
   until the user selects a style/environment or enables shadows.

## Consequences

- The viewport gains its first descriptor set; the four existing pipelines and the
  new skybox/shadow pipelines share one set layout (scene UBO + IBL textures +
  shadow map). This is a one-time complexity step the architecture had deferred.
- Realistic ships richer **incrementally**: lights + exposure land first on the
  existing PBR (no textures), then IBL, then shadows — each independently testable
  and shippable.
- Lighting styles are one preset table, the single place a style is defined —
  matching the display-mode resolver, with a totality test guarding additions.
- The `Environment` loader is the reusable seam for presentation renders (M16 F04)
  and any future HDR-driven output.
- Determinism (ADR-0023 §5): the split-sum precompute and shadow PCF are
  deterministic under llvmpipe; IBL/shadow correctness gets CPU-reference oracles
  for the analytic parts and Blender perceptual goldens for the full image.
