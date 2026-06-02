---
milestone: M18
feature: F05
pbi: PBI-170
title: Model-based GD&T & tolerance-stack analysis
status: planned
estimate: L
---

# PBI-170 — Model-based GD&T & tolerance-stack analysis

**Milestone:** M18 Analysis, Measurement & Simulation  ·  **Feature:** F05 Tolerance & GD&T Analysis

## Goal

Implement model-based tolerance features/datum frames and a tolerance-stack analysis computing min/max/statistical assembly variation.

## Scope / work

- `ModelToleranceFeature`/`ModelDatumReferenceFrame`.
- Worst-case & statistical stack analysis.
- Report contributors.

## API contracts (interfaces / enums / collections)

- `ModelToleranceFeature(s)`,`ModelDatumReferenceFrame`

## Acceptance criteria

- A 1D stack reports worst-case and RSS variation with contributor breakdown.

## Depends on

_See feature dependencies._
