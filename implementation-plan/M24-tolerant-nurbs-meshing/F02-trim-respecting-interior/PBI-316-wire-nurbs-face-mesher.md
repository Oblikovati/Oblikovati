---
milestone: M24
feature: F02
pbi: PBI-316
title: Wire the NURBS face mesher (oracle-gated)
status: blocked
---

# PBI-316 — Wire the NURBS face mesher (oracle-gated)

**Milestone:** M24 Tolerant NURBS Surface Meshing  ·  **Feature:** F02 Trim-respecting adaptive interior

## Goal

Route B-spline faces through an on-surface interior-node mesher (pcurve boundary + adaptive
trim-clipped interior + non-folding triangulation), replacing `trimmedPatchMesh`, **only if** it
holds the volume oracle.

## BLOCKED — finding (2026-06-08): the (u,v)-interior approach is a dead end for imported faces

Assembled `nurbsPatchMesh` (F01 marchUV pcurve + F02 PBI-314 interior grid + PBI-315 fold repair)
and wired it; measured on EDF.STEP. It **inflated the body volume +33%** (281k vs the 210k baseline,
OCC truth 207k). Diagnosis (all measured, then reverted):

- The marchUV pcurves are **clean** (0 self-intersections on every EDF face) and per-face areas
  match the boundary-only mesh (~1.0 ratio) — F01 works.
- But the interior `(u,v)` nodes' `PointAt` lands **~15% of the face size off** the true trim
  surface (13–25 mm on faces 61–147 mm across), **uniformly**, on every imported face. The
  imported rational-NURBS **parameterization is non-conformal**: a `(u,v)` point inside the trim's
  `(u,v)` boundary does NOT map to a 3D point inside the trim. So a `(u,v)` interior grid cannot
  refine these faces — it samples the wrong part of the surface and over-encloses.
- A `ParamAt`-vs-pcurve **branch mismatch** also flipped boundary normals against the interior
  (fixed by `nurbsBoundaryLoops`), but that was secondary; the parameterization is the wall.
- A **deviation guard** (fall back if interior nodes stray from the boundary-only mesh) cannot
  distinguish a non-conformal error from a **legitimate bulge** — both deviate from the flat
  boundary-only mesh — so it falls back even a correct conformal dome. No usable guard on this path.

**Conclusion:** F02's `(u,v)`-grid interior is the wrong primitive for imported non-conformal
NURBS. The mesher must refine in **3D space**, not `(u,v)`. The F01 pcurve, PBI-314 adaptive
density, and PBI-315 fold repair stay valid + tested; only the interior *sampling* changes. Reverted
the wiring; develop stays at the correct boundary-only baseline (210k). See PBI-321.

## Superseded by

[PBI-321 — 3D-space interior refinement](PBI-321-3d-space-refinement.md).
