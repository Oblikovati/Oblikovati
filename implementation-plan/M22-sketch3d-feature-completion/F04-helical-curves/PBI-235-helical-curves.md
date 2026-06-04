---
milestone: M22
feature: F04
pbi: PBI-235
title: Helix3d kernel + HelicalCurve entity (4 modes) + constraint + tool
status: planned
estimate: M
---

# PBI-235 — Helical curves

**Milestone:** M22  ·  **Feature:** F04 Helical Curves

## Goal
Add an analytic helix to the geometry kernel and expose it as a 3D-sketch entity with
all four Inventor definition modes.

## Scope / work
- **kernel:** `kernel/geom/helix.go` — `Helix3d{Axis origin/dir, StartRadius,
  Pitch, Turns, Taper}` implementing `Curve3` (`PointAt`/`TangentAt`/`Length`/`ParamAt`).
  Supports cylindrical (taper 0) and conical/spiral (taper ≠ 0). Length via Gauss
  quadrature; arc-length monotonic.
- **model:** `model/sketch/helix_3d.go` — `HelicalCurve` entity wrapping `Helix3d`,
  built from any two of {pitch, height, revolutions} (+ spiral); `HelicalConstraint3D`.
- **/api:** `Sketch3DEntityHelical`; `AddSketch3DEntityArgs` helix fields (mode,
  pitch, height, revolutions, startRadius, taper, ccw); `client.Sketch3D.AddHelix`.
- **router** case; **UI** helix tool + dialog (mode combo + value fields).

## Acceptance criteria
- Kernel unit + property tests ≥98%: arc-length monotonic; cylindrical helix radius
  constant; `PointAt(0)`/`PointAt(1)` match endpoints; pitch·turns ≈ height.
- Dogfood: add a 10mm-pitch, 5-turn helix; enumerate; round-trip.
- UI: ≥1 e2e test (mode combo → values → commit → helix in model).
- `make ci` green.

## Depends on
PBI-232.
