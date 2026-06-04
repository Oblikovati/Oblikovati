---
milestone: M23
feature: F04
pbi: PBI-311
title: Watercolor (8714) — pigment fills + paper + edge darkening
status: planned
estimate: L
---

# PBI-311 — Watercolor (8714) — pigment fills + paper + edge darkening

**Milestone:** M23 Renderer Display-Mode Parity & Realistic PBR  ·  **Feature:** F04 NPR Stylized Modes

## Goal

The `kWatercolorRendering` (8714) mode: soft pigment-dilution fills on a paper texture
with darkened edges — the most painterly of the NPR modes.

## Scope / work

- **Pigment fills**: simplify the shaded image to soft, slightly-varying flat washes
  (e.g. abstraction/quantization with gentle in-region noise) rather than crisp cel
  bands.
- **Paper texture**: modulate value by a tiling paper/grain texture (a small built-in
  asset) so washes sit on "paper."
- **Edge darkening (pigment accumulation)**: darken toward silhouettes/creases using the
  PBI-308 edge AOV, mimicking pigment pooling at edges.
- A `StylizeConfig` over PBI-308; wire the `VisualStyle` through the F01 resolver.
- Honor determinism mode: any procedural noise uses fixed seeds, no temporal jitter
  (ADR-0014), so goldens are stable.

## API contracts (interfaces / enums / collections)

- (internal) Watercolor `StylizeConfig`; built-in paper texture; resolver `PassSet` entry
  for 8714
- No new public surface.

## Acceptance criteria

- **Blender / perceptual golden** for the watercolor pass within the pipeline's perceptual
  tolerance (the painterly look is not bit-exact); plus **metamorphic stability** — the
  same scene re-rendered in determinism mode is identical frame-to-frame (no jitter).
- Edge darkening: pixels near silhouettes/creases are measurably darker than region
  interiors (sampled from the AOV), confirming pigment accumulation.
- Paper modulation is present (value variance from the paper texture in flat regions,
  sampled numerically).
- Selecting via `api/client` (PBI-300) and gallery (PBI-302) yields the expected `PassSet`
  and AOV. Validation clean.

## Depends on

PBI-308 (framework), F01 (resolver + selection). Closes M23 display-mode parity.
