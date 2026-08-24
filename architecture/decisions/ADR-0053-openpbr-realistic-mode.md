<!-- SPDX-License-Identifier: GPL-2.0-only -->

# ADR-0053 — OpenPBR Surface Realistic mode (dual-backend path tracer)

**Status:** Accepted (2026-08-24). Supersedes [ADR-0023](ADR-0023-viewport-display-modes.md)
§2's "Realistic = software raster PBR, hardware ray tracing explicitly out of scope" call.
Extends [ADR-0005](ADR-0005-vulkan13-renderer.md) (Vulkan renderer), preserves
[ADR-0014](ADR-0014-renderer-testability.md) (pure draw list / CPU-reference oracles), reuses
[ADR-0044](ADR-0044-recompute-dependency-invalidation-seam.md)'s dirty-tracking seam, and
deprecates (without removing) the `Appearance` contract from
[ADR-0022](ADR-0022-materials-appearances.md) §2. Drives milestone **M45**; supersedes the
remaining raster-wiring scope of M23-F02 issue #565; is the foundation M30 (Rendering &
Animation Studio) builds on for offline photoreal render/animation output.

## Context

ADR-0023 shipped `Realistic` as a raster **glTF metallic-roughness** shader: one
GGX/Cook-Torrance BRDF (`head/internal/native/shaders/mesh.frag`), five material scalars
(albedo, metallic, roughness, emissive, opacity — `Oblikovati.API/contract/appearance.go`),
and analytic split-sum image-based lighting. It scoped that way deliberately: hardware ray
tracing was called out as explicitly out of scope, and "software" was a requirement, not a
fallback.

metallic-roughness is a single specular+diffuse lobe. It cannot represent — no matter how the
raster shader is extended — coat/fuzz inter-reflection, thin-film interference, volumetric
subsurface scattering, or physically dispersive transmission through a nested-dielectric
stack, because each of those requires multi-bounce light transport a single-pass rasterizer
does not evaluate. The OpenPBR Surface specification
(github.com/AcademySoftwareFoundation/OpenPBR, Apache-2.0, stable since v1.0, current v1.1.1)
formally layers exactly those lobes (Base diffuse/subsurface/transmission, Specular, Coat,
Fuzz, Thin-film, Emission, Geometry) on top of a metallic-roughness-compatible base. Being
**fully compliant** with it — every lobe, not a cut-down real-time subset — is the explicit
requirement for this migration; it is not satisfiable by extending `mesh.frag`.

Full compliance needs a path tracer. Near-real-time performance for a path tracer needs
hardware-accelerated ray intersection. The user has authorized using the Vulkan ray-tracing
API for this, on one explicit condition: hardware RT must be an **optional** checkbox that
only costs performance/convergence time when unchecked, never correctness — a software
(compute-shader) ray tracer stays fully OpenPBR-compliant on any GPU, just slower to converge.

## Decision

1. **Realistic becomes a progressive, physically based path tracer**, fully compliant with
   OpenPBR Surface 1.1.1 — all lobes, no reduced-fidelity mode. Every other `VisualStyle`
   value (Shaded, ShadedWithEdges, Wireframe, the HLR and NPR modes) keeps the existing raster
   metallic-roughness pipeline untouched; only `Realistic`/`kRealisticRendering` migrates.

2. **Two ray-intersection backends behind one shared BSDF/path-integrator, selected by a
   persisted checkbox** (default on iff the device reports
   `VK_KHR_acceleration_structure` + `VK_KHR_ray_tracing_pipeline`/ray query support):
   - **Hardware backend** — BLAS per body from the existing tessellated drawlist triangles,
     TLAS from the existing per-instance assembly transforms (ADR-0038); refit/rebuild
     triggers off the same tessellation-dirty signal the viewport tessellation cache already
     uses (ADR-0044), not a new dirty-tracking mechanism.
   - **Software backend** — a compute-shader BVH build + traversal over the same triangle
     data. Always available, no RT extensions required, no fidelity reduction — only slower
     convergence to the same spec-correct image. This is the path unchecking the box selects.
   - Both sit behind a small `Intersector` interface so path-integration and BSDF evaluation
     stay unit-testable on the CPU (ADR-0014) independent of which GPU backend is live.

3. **Progressive accumulation, not per-frame full convergence.** A persistent accumulation
   buffer collects samples frame over frame while the viewport is idle; any camera or scene
   edit resets it. A 1-bounce raster/preview pass covers active orbiting so the viewport never
   blocks on convergence. No ML denoiser in this milestone — accumulation-only convergence for
   v1, denoising tracked as a follow-up issue, not a blocker.

4. **Public API: new `OpenPBRAppearance` type, additive alongside the existing `Appearance`.**
   Parameter groups/names/defaults/ranges transcribe OpenPBR's `parametrization.md.html` 1:1
   (Base/Specular/Transmission/Subsurface/Coat/Fuzz/Thin-film/Emission/Geometry) rather than a
   bespoke redesign. `Appearance` is marked deprecated (docs/comments only) and keeps working;
   its removal is a separate future MAJOR-version decision, out of scope here. Ships as a MINOR
   `Oblikovati.API` bump.

5. **BSDF math is ported from Adobe's `openpbr-bsdf`** (github.com/adobe/openpbr-bsdf,
   Apache-2.0), which already targets path tracing and implements the full 1.1.1 lobe set
   including multi-scatter energy-compensation LUTs — license-compatible with both the
   Apache-2.0 API module and the GPL-v2 application module.

6. **Testing extends, not replaces, the existing conventions.** Every new lobe gets a
   deterministic CPU-reference-oracle unit test (fixed-angle BSDF evaluations, ADR-0014's
   pattern). The Blender-golden pipeline
   (`architecture/testing/00-renderer-oracle-pipeline.md`) extends to a converged OpenPBR
   reference image. `mcplive` + MCP screenshot verification exercises hardware-RT-on,
   hardware-RT-off, and several lobes (coat, fuzz, subsurface) before any PR, per this repo's
   live-test convention.

## Consequences

- **M23-F02 issue #565** ("Realistic mode wiring + multi-light + Blender golden") is
  superseded: its remaining acceptance criteria (multi-light consumption, Blender golden,
  selector wiring) fold into M45-F05 PBI-350 (issue #2137) instead of being built twice for a
  shader about to be replaced. #565 is a candidate to close once #2137 lands — closing it is a
  separate, explicitly user-confirmed action, not automatic from this ADR.
- **M30** (Rendering & Animation Studio), which already documents itself as building on "the
  M23 PBR/IBL renderer" for offline photoreal output, instead gets a path-tracing foundation:
  the same progressive integrator this ADR builds, driven at higher sample counts and without
  the idle/interactive constraint. M30's own scope (render-to-image, keyframe animation) is
  unaffected and stays out of this ADR.
- **No regression risk to shipped styles.** Because only `Realistic` migrates, `mesh.frag` and
  every other `VisualStyle`'s pass resolution (ADR-0023 §1) are untouched by this decision.
- **Hardware capability is no longer a hard requirement for Realistic mode**, reversing
  ADR-0023 §2's software-only stance in the other direction: HW RT is now permitted, but,
  per the user's explicit condition, never mandatory — every acceptance test for Realistic
  mode must pass with the checkbox in either state.
- **A new spec dependency** (OpenPBR 1.1.1, tracked externally by ASWF) enters the renderer.
  Compliance is validated against the spec's own reference values and the Adobe `openpbr-bsdf`
  port, not re-derived from first principles per lobe.
