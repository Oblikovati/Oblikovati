---
milestone: M24
feature: F04
pbi: PBI-320
title: EDF end-to-end volume + fold regression
status: planned
estimate: S
---

# PBI-320 — EDF end-to-end volume + fold regression

**Milestone:** M24 Tolerant NURBS Surface Meshing  ·  **Feature:** F04 Oracle gating & EDF regression

## Goal

Pin the driving case: EDF.STEP imports fold-free and volume-correct, as a committed regression.

## Scope / work

- A local/CI regression that imports `EDF.STEP`, asserts total volume within tolerance of the OCC
  ground truth (207,002) and **0 fold-edges** across all faces.
- Keep EDF out of the public/clean-room path (it is a third-party model) — a local fixture path /
  CI artifact, skipped when absent, per the repo's clean-room rule.
- Final live confirmation on the running head (shaded + Normal-Debug, all bodies): no staircase,
  no slivers, external surfaces smooth.

## Acceptance criteria

- EDF regression: volume within tolerance, 0 fold-edges; skipped cleanly when the model is absent.
- Live: the external freeform faces read smooth in shaded and Normal-Debug.
- Milestone exit criteria all met; OCC oracle green; lint clean.

## Depends on

F02, F03, F04 PBI-319.
