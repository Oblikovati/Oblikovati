---
milestone: M12
feature: F03
pbi: PBI-128
title: Constraint/joint drive & animation
status: planned
estimate: M
---

# PBI-128 — Constraint/joint drive & animation

**Milestone:** M12 Assembly: Constraints, Joints, Motion & Representations  ·  **Feature:** F03 iMates & Drive

## Goal

Implement driving a constraint/joint through a value range with steps, repeats, and optional collision detection for motion study.

## Scope / work

- `DriveConstraintSettings` (start/end/step/repeat).
- Joint drive.
- Collision stop.

## API contracts (interfaces / enums / collections)

- `DriveConstraintSettings`,`DriveConstraintSettingsObject`

## Acceptance criteria

- Driving a rotational joint animates assembly motion; collision halts the drive.

## Depends on

_See feature dependencies._
