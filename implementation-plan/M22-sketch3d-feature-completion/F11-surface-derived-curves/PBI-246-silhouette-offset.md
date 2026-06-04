---
milestone: M22
feature: F11
pbi: PBI-246
title: SilhouetteCurve + OffsetCurve3
status: partial (model+kernel layer done; /api face-ref binding + serialize + recompute TODO)
estimate: M
---

# PBI-246 — Silhouette + offset curves

**Milestone:** M22  ·  **Feature:** F11 Surface-Derived Curves

## Goal
Add the remaining two surface-derived curves.

## Scope / work
- `model/sketch/surface_curves_3d.go` (extend): `SilhouetteCurve` (face ref + direction
  → F10 silhouette), `OffsetCurve3` (3D curve ref + distance + plane/normal). Reference-
  key bound; recompute-driven.
- `/api`: entity kinds + args; collections (`SilhouetteCurves`, `OffsetCurve3` via
  entities); `client` helpers.
- router cases; UI tools + ribbon buttons.

## Acceptance criteria
- Unit ≥98%: cylinder silhouette for axis-perpendicular view = two lines; offset of a
  circle = concentric circle; identity tests.
- Dogfood; round-trip; ≥1 UI e2e test; `make ci` green.

## Depends on
PBI-245.
