---
milestone: M08
feature: F03
name: Sketched Features
status: planned
---

# M08 · F03 — Sketched Features

The primary additive/subtractive features driven by sketch profiles and paths, each delivered as the full Definition→Add→Feature triangle with its extent/termination options and operation (join/cut/intersect/new-body/surface).

## In scope

- Extrude, Revolve, Sweep, Loft, Coil, Rib.
- Extents/terminations (distance/to-face/to-next/through-all/from-to).
- Operations; taper; two-directional; multi-body affected-bodies.

## Out of scope

_None._

## Key API contracts delivered

- `ExtrudeFeature(s)`,`ExtrudeDefinition`,`RevolveFeature(s)`,`RevolveDefinition`,`SweepFeature(s)`,`SweepDefinition`,`LoftFeature(s)`,`LoftDefinition`,`CoilFeature(s)`,`RibFeature(s)`
- `PartFeatureExtent`,`PartFeatureExtentEnum`,`PartFeatureOperationEnum`,`PartFeatureExtentDirectionEnum`

## Depends on

F01,F02,M06.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-092](PBI-092-extrude.md) | Extrude feature (full triangle + extents) |
| [PBI-093](PBI-093-revolve.md) | Revolve feature |
| [PBI-094](PBI-094-sweep.md) | Sweep feature |
| [PBI-095](PBI-095-loft.md) | Loft feature |
| [PBI-096](PBI-096-coil-rib.md) | Coil & rib features |
