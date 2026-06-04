---
milestone: M23
feature: F02
pbi: PBI-305
title: Realistic mode wiring + multi-light + Blender golden
status: planned
estimate: L
---

# PBI-305 — Realistic mode wiring + multi-light + Blender golden

**Milestone:** M23 Renderer Display-Mode Parity & Realistic PBR  ·  **Feature:** F02 Realistic PBR Shading (software)

## Goal

Turn the GGX + IBL path into the selectable `kRealisticRendering` mode, lit by the
scene's lighting style, and lock it with a Blender ground-truth golden.

## Scope / work

- Wire the Realistic `VisualStyle` (F01 resolver) to the GGX+IBL surface pass; ensure the
  F01 contract/UI selection lands here end to end.
- Consume **multi-light** lighting styles from M16 F03 (a set of directional/point lights
  with intensities/colors) through the app resolver seam; provide a **default headlight +
  ambient rig** as the fallback when M16 F03 has not landed, so Realistic is usable now.
- Add a Realistic golden to the Blender oracle pipeline
  ([architecture/testing/00-renderer-oracle-pipeline.md](../../../architecture/testing/00-renderer-oracle-pipeline.md)):
  the same scene + appearances + lights rendered in Blender (Eevee/Cycles) as the
  perceptual reference for the lit pass.

## API contracts (interfaces / enums / collections)

- (internal) Realistic pass wired through the F01 resolver; light-rig inputs via resolver.
- No new public surface.

## Acceptance criteria

- Selecting `kRealisticRendering` via `api/client` (PBI-300) and via the gallery
  (PBI-302) renders the GGX+IBL pass — asserted on the offscreen backend.
- A reference scene (a few bodies with distinct appearances — e.g. steel, ABS, oak from
  the M19 built-ins — under a fixed lighting style) matches the **Blender golden** within
  the pipeline's perceptual tolerance.
- With the fallback rig (no M16 F03), Realistic still renders deterministically and passes
  the CPU-reference lit-pass oracle.
- Determinism-mode goldens committed; validation layers clean.

## Depends on

PBI-303, PBI-304, F01 (resolver + selection), M16 F03 (lighting styles; fallback used
otherwise), M19 (appearance scalars).
