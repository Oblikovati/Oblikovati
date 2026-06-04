---
milestone: M23
name: Renderer Display-Mode Parity & Realistic PBR
status: planned
---

# M23 — Renderer Display-Mode Parity & Realistic PBR

Bring the **viewport** to full parity with Autodesk Inventor's `DisplayModeEnum`. Today
the renderer draws only three of Inventor's eleven display modes (`Shaded`,
`ShadedWithEdges`, `Wireframe` — [renderer/drawlist.go:106](../../renderer/drawlist.go));
a comment there records that the richer presets "need NPR work." This milestone closes
that gap across three families of new rendering: a **software PBR** path (Realistic),
**real-time hidden-line removal** for the viewport, and a **non-photorealistic (NPR)**
framework for the four stylized modes — plus the public `DisplayMode` contract and the
Visual Style UI that selects them. See [ADR-0023](../../architecture/decisions/ADR-0023-viewport-display-modes.md).

Realistic is **PBR-capable but software** (raster GGX metallic-roughness + analytic IBL +
tone mapping); **hardware ray tracing is out of scope**. The PBR appearance data
(albedo/metallic/roughness/emissive/opacity) already reaches `renderer.Surface` from M19
(ADR-0022) — the missing half is the shading model that consumes it.

## DisplayModeEnum coverage (ids per `DisplayModeEnum.cs`, Inventor 2026)

| id | member | today | M23 |
|----|--------|-------|-----|
| 8706 | `kWireframeRendering` | ✅ `renderer.Wireframe` | keep |
| 8707 | `kShadedWithHiddenEdgesRendering` / `kHiddenEdgeRendering` (aliased) | ❌ | F03 |
| 8708 | `kShadedRendering` | ✅ `renderer.Shaded` | keep |
| 8709 | `kRealisticRendering` | ⚠️ data only | F02 |
| 8710 | `kShadedWithEdgesRendering` | ✅ `renderer.ShadedWithEdges` | keep |
| 8711 | `kWireframeNoHiddenEdges` | ❌ | F03 |
| 8712 | `kWireframeWithHiddenEdgesRendering` | ❌ | F03 |
| 8713 | `kMonochromeRendering` | ❌ | F04 |
| 8714 | `kWatercolorRendering` | ❌ | F04 |
| 8715 | `kIllustrationRendering` | ❌ | F04 |
| 8716 | `kTechnicalIllustrationRendering` | ❌ | F04 |

## Goals

- Every `DisplayModeEnum` member selectable in the viewport and through `/api`.
- A PBR-capable software **Realistic** mode driven by the M19 appearance scalars.
- Real-time viewport **hidden-line removal** feeding the three hidden-edge modes.
- A reusable **NPR framework** backing Monochrome, Watercolor, Illustration, and
  Technical Illustration.
- Every new pass an isolatable, oracle-tested AOV (ADR-0014) — no eyeballing.

## In scope

- Public `DisplayModeEnum` contract + `View`/`ClientView` `DisplayMode` get/set.
- The full `renderer.VisualStyle` set + a pure style→passes resolver.
- GGX metallic-roughness shading, analytic/prefiltered IBL, tone mapping.
- Tessellated depth-occlusion edge visibility + a dashed hidden-edge pass.
- The NPR edge-detection AOV + stylization compositor and the four NPR styles.
- Visual Style ribbon gallery + command + end-to-end UI selection.

## Out of scope (handled elsewhere)

- **Hardware ray tracing** (explicitly deferred; Realistic is raster/software).
- **Image-texture maps & face-level override rendering** — stay deferred (ADR-0022 §2).
- **Exact analytic drawing HLR** — that is [ADR-0012](../../architecture/decisions/ADR-0012-exact-hidden-line.md)/M14; M23's viewport HLR is the real-time tessellated cousin.
- The material/appearance **model** itself (M19) and camera/light/lighting-style
  **model** (M16) — M23 consumes them, it does not build them.

## Exit criteria

- Setting each `DisplayModeEnum` member through `api/client` round-trips and selects the
  corresponding viewport style (dogfood test).
- Realistic renders the M19 appearance scalars through a GGX BRDF, matching a pure-Go CPU
  reference within tolerance and a Blender golden perceptually.
- A box occluded by another box renders correctly in all three hidden-edge modes,
  matching the analytic occlusion oracle.
- Each NPR mode produces its style and matches its oracle tier (CPU-reference for the
  deterministic math; Blender/metamorphic for the painterly parts).
- The Visual Style ribbon gallery selects any mode and an end-to-end test asserts the
  active renderer style.

## Depends on

M07 (tessellation + model edges), M19 (appearance PBR data → `renderer.Surface`),
M16 F03 (cameras/lights/lighting styles for multi-light Realistic; a default rig is the
fallback), M05 (commands/UI). Architecture: ADR-0023, ADR-0005, ADR-0012, ADR-0014,
ADR-0022.

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Display-Mode Contract & Style Plumbing](F01-display-mode-contract/_feature.md) | 3 | `DisplayModeEnum` public contract, the full `VisualStyle` set + style→passes resolver, and the Visual Style ribbon gallery. |
| **F02** | [Realistic PBR Shading (software)](F02-realistic-pbr/_feature.md) | 3 | GGX metallic-roughness BRDF, image-based lighting + tone mapping, and the wired Realistic mode with a Blender golden. |
| **F03** | [Viewport Hidden-Line Visibility](F03-viewport-hidden-line/_feature.md) | 2 | Per-frame tessellated edge-occlusion classification and the dashed hidden-edge pass feeding modes 8707/8711/8712. |
| **F04** | [NPR Stylized Modes](F04-npr-stylized/_feature.md) | 4 | The NPR edge-detection + stylization framework and the four stylized modes (Monochrome, Illustration, Technical Illustration, Watercolor). |
