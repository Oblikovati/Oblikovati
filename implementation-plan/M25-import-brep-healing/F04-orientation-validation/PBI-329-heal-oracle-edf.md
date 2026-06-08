---
milestone: M25
feature: F04
pbi: PBI-329
title: Healing oracle + EDF end-to-end regression through M24
status: planned
estimate: M
---

# PBI-329 — Healing oracle + EDF end-to-end regression through M24

**Milestone:** M25 Imported B-Rep Healing  ·  **Feature:** F04 Orientation, validation & oracle gate

## Goal

Lock the milestone: prove healing is safe on good input and that healed EDF meshes fold-free +
volume-correct through the (already-built) M24 mesher — the M24 exit criterion, now reachable.

## Scope / work

- Wire healing into the STEP import path (heal-after-import), behind the model tolerance.
- Extend the OCC oracle: healing a clean fixture is a **no-op** (volume + face count unchanged).
- Enable the M24 NURBS mesher on the healed geometry; assert EDF freeform faces are fold-free
  (committed fold detector → 0) and total volume within tolerance of OCC `getMass`.
- Final live confirmation (shaded + Normal-Debug, Save-Viewport-PNG): the EDF staircase is gone.

## Acceptance criteria

- Healing a clean OCC fixture changes neither volume nor face count.
- Healed EDF.STEP: 0 fold-edges on freeform faces; total volume within tolerance of OCC (no
  inflation) — closing M24's exit criterion.
- OCC oracle green; lint clean; live staircase gone.

## Depends on

F01–F03, PBI-328, the M24 mesher (F01 + PBI-314/315), the OCC oracle.
