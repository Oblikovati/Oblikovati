---
milestone: M08
feature: F03
pbi: PBI-092
title: Extrude feature (full triangle + extents)
status: planned
estimate: L
---

# PBI-092 — Extrude feature (full triangle + extents)

**Milestone:** M08 Part Modeling: Sketched & Work Features  ·  **Feature:** F03 Sketched Features

## Goal

Implement extrude as the reference feature: `ExtrudeDefinition`, `ExtrudeFeatures.Add`/`AddByDistanceExtent`/`AddByToFace`/`AddByFromToExtent`/`ToNext`/`ThroughAll`, and `ExtrudeFeature` with round-trippable `Definition`.

## Scope / work

- Definition with profile/operation/extent/taper/two-directional.
- All extent constructors & `Set*Extent` mutators.
- Result faces (start/end/side); proxy.
- Recompute & health.

## API contracts (interfaces / enums / collections)

- `ExtrudeFeature`,`ExtrudeFeatures`,`ExtrudeDefinition`,`ExtrudeFeatureProxy`,`PartFeatureExtent*`

## Acceptance criteria

- All extent types build valid solids.
- Editing `Definition` or a driving parameter recomputes.
- Start/end/side faces are queryable and reference-keyed.

## Depends on

_See feature dependencies._

## Notes

Establishes the canonical feature pattern every later feature copies. Get this one exemplary.
