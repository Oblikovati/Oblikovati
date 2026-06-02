---
milestone: M10
name: Surfacing & Freeform Modeling
status: planned
---

# M10 — Surfacing & Freeform Modeling

Surface and free-form modeling: creating and editing surface bodies (boundary patch, ruled, sculpt, stitch), surface editing (trim/extend/offset/mid-surface/thicken), sub-d free-form modeling, and mesh/imported geometry features including core/cavity (mold) tooling. Keep the 2D solver and analytic kernel decoupled from free-form sub-d behind clean boundaries.

## Goals

- Surface creation features (patch, ruled, sculpt, knit).
- Surface editing features (trim, extend, offset, mid-surface).
- Sub-D free-form bodies with edit operations.
- Mesh & imported geometry features and mold core/cavity.

## In scope

- BoundaryPatch/RuledSurface/Sculpt/Knit(Stitch)/Patch.
- Trim/Extend/FaceOffset/MidSurface/Thicken.
- FreeformFeature/AliasFreeform bodies/edit.
- MeshFeature; NonParametricBase; CoreCavity.

## Out of scope (handled elsewhere)

- Solid sketched/dress-up features (M08/M09).

## Exit criteria

- A bounded region of curves/surfaces creates a trimmed surface body.
- A free-form body edits via control-point/cage manipulation.
- A mid-surface is extracted for FEA (M18).

## Depends on

M08

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Surface Creation](F01-surface-creation/_feature.md) | 2 | Boundary patch, ruled, sculpt, knit/stitch. |
| **F02** | [Surface Editing](F02-surface-editing/_feature.md) | 2 | Trim, extend, offset, mid-surface, thicken. |
| **F03** | [Freeform Modeling](F03-freeform/_feature.md) | 2 | Sub-D free-form bodies and edit operations. |
| **F04** | [Mesh, Imported Geometry & Mold](F04-mesh-imported-mold/_feature.md) | 2 | Mesh features, non-parametric base, core/cavity tooling. |
