---
milestone: M23
feature: F04
name: NPR Stylized Modes
status: planned
---

# M23 · F04 — NPR Stylized Modes

The four non-photorealistic modes: `kMonochromeRendering` (8713),
`kIllustrationRendering` (8715), `kTechnicalIllustrationRendering` (8716), and
`kWatercolorRendering` (8714). Rather than four bespoke pipelines, this feature builds
**one screen-space NPR framework** — a shared edge-detection AOV plus a stylization
compositor — and expresses each mode as a configuration of it (ADR-0023 §4).

## In scope

- An edge-detection AOV (silhouette / crease / material-boundary from the existing depth,
  normal, and ID passes) and a stylization compositor.
- Monochrome + Illustration (desaturate / flat-cel + outline).
- Technical Illustration (Gooch cool-warm + emphasized edges, optional hatching).
- Watercolor (pigment-dilution fills + paper texture + edge darkening).

## Out of scope

- Hardware ray tracing or photorealism (that is F02 Realistic).
- New public surface (mode selection is F01).

## Key API contracts delivered

- (internal) `renderer` NPR edge-detection AOV; stylization compositor; per-mode configs
- No new public surface.

## Depends on

The depth/normal/ID passes
([architecture/core/08-renderer-vulkan.md](../../../architecture/core/08-renderer-vulkan.md)),
F01 (resolver + the four styles), the oracle pipeline
([architecture/testing/00-renderer-oracle-pipeline.md](../../../architecture/testing/00-renderer-oracle-pipeline.md)).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-308](PBI-308-npr-framework.md) | NPR edge-detection AOV + stylization compositor |
| [PBI-309](PBI-309-monochrome-illustration.md) | Monochrome (8713) + Illustration (8715) |
| [PBI-310](PBI-310-technical-illustration.md) | Technical Illustration (8716) — Gooch + edges |
| [PBI-311](PBI-311-watercolor.md) | Watercolor (8714) — pigment fills + paper + edge darkening |
