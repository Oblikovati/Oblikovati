# ADR-0023 — Viewport Display-Mode Parity (software PBR, real-time HLR, NPR framework)

**Status:** proposed (2026-06-04) · **Relates to:**
[ADR-0005](ADR-0005-vulkan13-renderer.md) (Vulkan renderer),
[ADR-0012](ADR-0012-exact-hidden-line.md) (exact drawing HLR),
[ADR-0014](ADR-0014-renderer-testability.md) (pure draw list / oracle hierarchy),
[ADR-0022](ADR-0022-materials-appearances.md) (PBR appearance data). Drives milestone
**M23**.

## Context

The viewport renders only **three** of Autodesk Inventor's display modes — `Shaded`,
`ShadedWithEdges`, `Wireframe` ([renderer/drawlist.go:106](../../renderer/drawlist.go))
— and a comment there records that "the richer Inventor presets (realistic/monochrome/
illustration, hidden-edge removal) need NPR work."

Inventor exposes **11 distinct** modes through `DisplayModeEnum` (authoritative copy at
`Oblikovati.Contracts/Oblikovati.Contracts.CSharp/Enums/DisplayModeEnum.cs`, Inventor
2026 reference, ids 8706–8716; note `kHiddenEdgeRendering` and
`kShadedWithHiddenEdgesRendering` **share** id 8707):

| id | member | family |
|----|--------|--------|
| 8706 | `kWireframeRendering` | wireframe (all edges) |
| 8707 | `kShadedWithHiddenEdgesRendering` / `kHiddenEdgeRendering` | shaded + hidden edges (HLR) |
| 8708 | `kShadedRendering` | shaded |
| 8709 | `kRealisticRendering` | PBR |
| 8710 | `kShadedWithEdgesRendering` | shaded + edges |
| 8711 | `kWireframeNoHiddenEdges` | wireframe, visible edges only (HLR) |
| 8712 | `kWireframeWithHiddenEdgesRendering` | wireframe + dashed hidden edges (HLR) |
| 8713 | `kMonochromeRendering` | NPR |
| 8714 | `kWatercolorRendering` | NPR |
| 8715 | `kIllustrationRendering` | NPR |
| 8716 | `kTechnicalIllustrationRendering` | NPR |

Three families of new work fall out: a **software PBR** path (Realistic), **real-time
hidden-line removal** for the viewport (8707/8711/8712), and a **non-photorealistic**
(NPR) framework for the four stylized modes (8713–8716). The PBR *appearance data*
(albedo/metallic/roughness/emissive/opacity) already flows to `renderer.Surface` from
M19 (ADR-0022 §2), but the shader is Blinn-Lambert and ignores everything but albedo.

The user has scoped this explicitly: implement **all** modes, and Realistic must be
**PBR-capable but software** — **no hardware ray tracing** for now.

## Decision

1. **A display mode is a `VisualStyle` resolved to a set of passes.** The public
   `DisplayModeEnum` mirrors the Inventor members and ids **exactly** (8706–8716,
   including the 8707 alias) — stable, explicit ids that are never renumbered (the
   CONVENTIONS naming rule). The app translates the public enum to a renderer-internal
   `VisualStyle`; a pure **style→passes resolver** decides which passes each mode
   enables (surface / edge / hidden-edge / npr-stylization). The enum lives in
   `api/types` and is aliased in `/source`; the renderer never imports the public
   package — it owns its own `VisualStyle` (ADR-0022 §6).

2. **Realistic = software raster PBR.** A GGX metallic-roughness BRDF replaces
   Blinn-Lambert in the surface shader, consuming the M19 scalars (metallic, roughness,
   emissive, opacity) already plumbed through `renderer.Surface`. Lighting is computed
   in linear space with an sRGB-encoded output and a tone-mapping step (ACES/Reinhard +
   exposure); ambient comes from analytic/prefiltered image-based lighting, not a path
   tracer. **Hardware ray tracing is explicitly out of scope.** Image-texture maps stay
   deferred (ADR-0022 §2) — Realistic uses scalar/color PBR only.

3. **Viewport HLR = real-time tessellated-mesh depth occlusion.** Hidden-edge modes
   classify each model edge segment visible/hidden per frame by testing it against the
   tessellated mesh depth buffer (a depth prepass), then draw hidden segments in a
   distinct dashed style. This is the real-time cousin of — and **deliberately distinct
   from** — ADR-0012's exact analytic B-rep HLR, which targets *drawing views* (M14) and
   is too slow per frame. The tessellated approach is upgradeable to curved silhouettes
   later. It feeds 8707 (shaded + hidden), 8711 (visible edges only), and 8712 (visible
   solid + hidden dashed).

4. **NPR = one screen-space framework, configured per mode.** A shared edge-detection
   AOV (silhouette / crease / material-boundary, derived from the existing depth, normal,
   and ID passes) plus a stylization compositor underpins all four NPR modes. Each mode
   is a *configuration* of that framework: Monochrome (desaturate + posterize +
   outline), Illustration (flat/cel + outline), Technical Illustration (Gooch cool-warm +
   emphasized edges), Watercolor (pigment-dilution fills + paper texture + edge
   darkening). No bespoke per-mode pipeline.

5. **Testability is preserved (ADR-0014).** Style→pass resolution stays pure data,
   unit-tested on the `null` backend. Every new pass writes a **named AOV the offscreen
   backend can capture in isolation**. The deterministic parts — GGX shading, desaturate,
   posterize, Gooch, outline extraction — get **CPU-reference oracles** (pure Go reusing
   `math/`, no external dependency). The painterly modes and full PBR get **Blender
   perceptual goldens**. All of it runs under determinism mode (Mesa llvmpipe, no
   temporal jitter) with the validation-layer gate on.

6. **The renderer stays pure (ADR-0022 §6).** Lights and environment are supplied by the
   app through the existing resolver seam (the same path `SurfaceLookup` takes), not by
   the renderer reaching into `model`. Multi-light Realistic consumes M16 F03 lighting
   styles when present, and falls back to a default headlight + ambient rig otherwise.

## Consequences

- A single enum + resolver is the one place modes are added; the public contract is a
  faithful mirror of Inventor's `DisplayModeEnum`, so add-ins set display modes with the
  exact member names and ids they expect.
- Realistic ships *now* on the existing scalar PBR data; when image textures land
  (deferred, ADR-0022 §2) the GGX shader extends rather than gets replaced.
- Viewport HLR and drawing HLR are two implementations of one idea at different
  fidelity/latency points; ADR-0012 remains the authority for drawings, this ADR for the
  viewport. Sharing may be revisited if the tessellated path proves accurate enough.
- The NPR framework is reusable surface for future stylized output (e.g. presentation
  renders, M16 F04) beyond the four parity modes.
- Each mode is an independently capturable AOV, so a visual regression localizes to one
  pass and one oracle tier rather than "the picture is wrong."
