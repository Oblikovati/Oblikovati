---
milestone: M24
feature: F02
pbi: PBI-316
title: Wire the NURBS face mesher (oracle-gated)
status: planned
estimate: M
---

# PBI-316 — Wire the NURBS face mesher (oracle-gated)

**Milestone:** M24 Tolerant NURBS Surface Meshing  ·  **Feature:** F02 Trim-respecting adaptive interior

## Goal

Route B-spline faces through the new on-surface interior-node mesher (pcurve boundary + adaptive
trim-clipped interior + non-folding triangulation), replacing `trimmedPatchMesh`, **only if** it
holds the volume oracle.

## Scope / work

- Assemble F01+F02 into `nurbsPatchMesh(s, outer3D, holes3D)`: march-projected pcurves → adaptive
  trim-clipped interior nodes → CDT → fold-repair → mesh. Boundary nodes lifted **on-surface**
  (`PointAt` of the pcurve), interior likewise — consistent, so no exact-vs-on-surface overlap.
  (F03 then makes the on-surface boundary consistent across faces.)
- Route `geom.BSplineSurface` non-rectangular faces to it in `tessellate_trim.go`
  (`nonRectangularMesh`); analytic faces unchanged.
- **Gate on the oracle**: measure EDF total volume and per-face folds. This PBI may still leave a
  per-face seam (F03 fixes that) but must NOT over-enclose — the volume must be within tolerance
  of OCC (no +20–70%). If a face inflates, fall back to the boundary-only path for that face
  (a guard) so the milestone never regresses volume.

## API contracts (interfaces / enums / collections)

- (internal) `ops.nurbsPatchMesh`; dispatch in `tessellateCurvedFace`/`nonRectangularMesh`.

## Acceptance criteria

- EDF.STEP external freeform faces are **fold-free** (committed detector) and the total volume is
  **within tolerance of OCC** (no inflation) — the per-face inflation guard ensures no regression.
- The OCC oracle suite stays green; the EDF volume does not regress below the current 101.5%
  baseline beyond the stated tolerance.
- Live confirmation (shaded + Normal-Debug, Save-Viewport-PNG): the staircase is gone on the
  external faces.
- `go test ./kernel/...` green; lint clean.

## Depends on

F01 (pcurves), PBI-314 (interior), PBI-315 (no-fold), the OCC oracle.
