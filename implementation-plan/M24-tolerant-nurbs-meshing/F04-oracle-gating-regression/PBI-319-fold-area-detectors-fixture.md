---
milestone: M24
feature: F04
pbi: PBI-319
title: Fold + over-enclosure detectors and a trimmed-NURBS OCC fixture
status: planned
estimate: M
---

# PBI-319 — Fold + over-enclosure detectors and a trimmed-NURBS OCC fixture

**Milestone:** M24 Tolerant NURBS Surface Meshing  ·  **Feature:** F04 Oracle gating & EDF regression

## Goal

Make the diagnostic metrics permanent tests, and add a freeform-NURBS fixture the oracle can gate.

## Scope / work

- Commit a **fold detector** (interior edge whose two triangles' 3D normals oppose) and a
  **per-face-area / over-enclosure** check as test helpers in `kernel/ops`/`kernel/exchange/step`.
- Generate a synthetic **trimmed B-spline patch with a hole** via `test-utilities/step-oracle`
  (OpenCASCADE) → `testdata/occ/`, with its `getMass` volume in the oracle JSON.
- Extend `occ_oracle_test.go` to assert, for the freeform fixture: volume within tolerance AND
  0 fold-edges AND no over-enclosure.

## Acceptance criteria

- The freeform-NURBS fixture passes volume + fold + over-enclosure assertions.
- The detectors are deterministic and unit-tested on hand-built meshes (a known fold, a known
  clean mesh).
- OCC oracle suite green; lint clean.

## Depends on

F02 (the mesher), `test-utilities/step-oracle`, the gmsh SDK.
