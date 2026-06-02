---
milestone: M14
feature: F03
name: Dimensions & Annotations
status: planned
---

# M14 · F03 — Dimensions & Annotations

The annotation layer: retrieved model dimensions and standalone drawing dimensions (linear/angular/radial/ordinate/baseline/chain), centerlines/center marks, and GD&T (feature control frames, datums, surface texture) placed on annotation planes.

## In scope

- `GeneralDimension`/`DrawingDimension` types; baseline/ordinate/chain.
- Retrieve model dimensions; `AnnotationPlanes`.
- Centerlines/center marks.
- GD&T: feature control frames, datum frames, surface texture.

## Out of scope

_None._

## Key API contracts delivered

- `DrawingDimension(s)`,`GeneralDimension`,`LinearGeneralDimension`,`AngularGeneralDimension`,`RadiusGeneralDimension`,`OrdinateDimension`,`BaselineDimensionSet(s)`
- `Centerline(s)`,`Centermark(s)`,`FeatureControlFrame(s)`,`DatumReferenceFrame`,`ModelDatumReferenceFrame`,`SurfaceTextureSymbol(s)`,`AnnotationPlane(s)`

## Depends on

F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-141](PBI-141-dimensions.md) | Drawing & model dimensions (all types) |
| [PBI-142](PBI-142-gdt-annotations.md) | Centerlines, GD&T & datum frames |
