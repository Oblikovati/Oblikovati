---
milestone: M18
feature: F05
name: Tolerance & GD&T Analysis
status: planned
---

# M18 · F05 — Tolerance & GD&T Analysis

3D tolerance modeling and analysis: model-based GD&T (tolerance features, datum reference frames) and tolerance-stack analysis that propagates dimensional/geometric tolerances to compute assembly variation.

## In scope

- `ModelToleranceFeature`; `ModelDatumReferenceFrame`.
- Tolerance annotations on the model.
- Tolerance-stack/variation analysis.

## Out of scope

_None._

## Key API contracts delivered

- `ModelToleranceFeature(s)`,`ModelDatumReferenceFrame`,`FeatureControlFrame`(M14)

## Depends on

M14.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-170](PBI-170-tolerance-analysis.md) | Model-based GD&T & tolerance-stack analysis |
