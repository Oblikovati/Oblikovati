---
milestone: M25
name: Imported B-Rep Healing (pcurves, snapping, sewing)
status: planned
---

# M25 — Imported B-Rep Healing (pcurves, snapping, sewing)

Heal imported B-reps so their geometry is **self-consistent and meshable/operable** — the missing
foundation that blocks [M24](../M24-tolerant-nurbs-meshing/_milestone.md) (tolerant NURBS meshing).
This is the equivalent of OpenCASCADE's **`ShapeHealing` / `ShapeFix`** (reference checkout at
[`OCCT/src/ModelingData/TKShHealing`](../../../OCCT/src) and `ModelingAlgorithms`): the pass every
robust kernel runs on imported STEP/IGES before meshing or modelling.

## Why (the M24 finding)

EDF.STEP (SolidWorks export) imports with two structural defects that no mesher can paper over,
proven during M24 (see the `m24-tolerant-nurbs-mesher` / `step-import-status` memories):

1. **No pcurves.** SolidWorks writes `0 SURFACE_CURVE` — edges carry only 3D curves, not their 2D
   `(u,v)` curve on each face's surface. So the mesher cannot know a face's trim region in `(u,v)`;
   it must guess it by projection, and the guess's interior over-encloses (+33% volume on EDF).
2. **Edges lie ~1.9 mm off their surfaces.** A STEP authoring tolerance: an edge's 3D curve does
   not lie exactly on the adjacent faces' surfaces. So the faces are not truly shared along edges,
   and the surface is **self-proximal** — a point inside a trim is closest to a *different* part of
   the same surface, so closest-point projection (verified sound against brute-force in M24) lands
   on the wrong part. No projector fixes this; the geometry itself must be healed.

OCCT meshes these models because `ShapeFix` **rebuilds the pcurves** (exactly bounding each trim in
`(u,v)`) and **snaps edges onto their surfaces** within tolerance, *then* sews and orients — so by
the time `BRepMesh` runs, the `(u,v)` regions are exact and `PointAt` over them is single-valued.
M25 builds that pass; M24's mesher (F01 pcurves + PBI-314/315, already merged + tested) then works.

## Goals

- Every imported face has **accurate pcurves** for its edges — the trim's `(u,v)` boundary, exact
  enough that sampling its interior stays on the trim (no over-enclosure).
- Edge 3D curves lie **on** their faces' surfaces within a controlled tolerance (the ~mm gap closed).
- Faces are **sewn** into watertight shells along genuinely-shared edges (the shared-edge topology
  the raw import lacks).
- Shells are **coherently oriented** (outward) and **validated** (manifold/closed) — or the defect
  is reported, never silently shipped.
- Healing is **measured** against the OpenCASCADE volume oracle and gated so it never degrades a
  model that was already fine.

## In scope

- A robust **surface point-inversion** API (formalising M24's verified `ParamNear`: multi-seed +
  branch/periodicity handling) and a **curve-on-surface projection** built on it.
- **Pcurve reconstruction** (`BRepLib::BuildCurves3d` analogue): march-project each edge onto each
  adjacent surface, handling seams, periodicity, and the correct `(u,v)` sheet, and attach the
  resulting pcurve to the topology so the mesher consumes it.
- **Edge/vertex tolerance snapping**: pull edge curves onto their surfaces and merge
  near-coincident vertices/edges within the model tolerance.
- **Face sewing**: identify shared edges by 3D proximity and stitch faces into shells, recording the
  shared-edge adjacency (so two faces of an edge mesh the SAME polyline).
- **Orientation + validation**: coherent outward shell orientation and a manifold/closed check, with
  a healing report.
- Oracle gating + an EDF end-to-end regression: healed EDF → M24 mesher → fold-free + volume-correct.

## Out of scope

- The **mesher** itself (M24 — it consumes the healed geometry).
- **Self-intersection / overlap removal** of grossly invalid solids (a deeper repair; M25 targets
  tolerance-level defects: missing pcurves, off-surface edges, unshared edges).
- Reading STEP-supplied pcurves when present (an optimisation; the driving case has none, so M25
  reconstructs them).

## Exit criteria

- Healed EDF.STEP: every face has pcurves whose interior `(u,v)`, sampled, stays on the trim (a
  point-on-trim check), edges lie on their surfaces within tolerance, and the body is a watertight
  oriented shell — verified by committed tests.
- With healing on, the **M24 mesher** produces the EDF freeform faces **fold-free** and the total
  volume within tolerance of OpenCASCADE `getMass` (the M24 exit criterion, now reachable).
- Healing a synthetic clean OCC fixture is a **no-op** (volume + face count unchanged) — never
  degrades good input.
- OCC oracle green; lint clean; live confirmation (shaded + Normal-Debug) that the EDF staircase is gone.

## Depends on

M07 (B-rep topology, `kernel/topo`), M17 (STEP import, `kernel/exchange/step`), M24's verified
projector (`geom.BSplineSurface.ParamNear`) + pcurve marching (`ops.marchUV`) + mesher blocks. A new
**ADR-0031 (imported B-rep healing)** records the heal-before-mesh pipeline and tolerance model.
Reference: OpenCASCADE `ShapeHealing` (`OCCT/src`).

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Robust projection & pcurve reconstruction](F01-pcurves-projection/_feature.md) | 2 | Formalise the verified point-inversion projector; reconstruct + attach per-edge pcurves that exactly bound each trim in `(u,v)`. |
| **F02** | [Edge/surface tolerance snapping](F02-edge-surface-snapping/_feature.md) | 2 | Pull edge curves onto their surfaces within tolerance; merge near-coincident vertices/edges. |
| **F03** | [Face sewing into watertight shells](F03-face-sewing/_feature.md) | 2 | Identify shared edges by proximity and stitch faces into shells with shared-edge adjacency. |
| **F04** | [Orientation, validation & oracle gate](F04-orientation-validation/_feature.md) | 2 | Coherent outward orientation + manifold/closed validation; healing oracle + EDF end-to-end regression through the M24 mesher. |
