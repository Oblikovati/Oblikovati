---
milestone: M12
feature: F02
pbi: PBI-126
title: Joint model (types, DOF, limits)
status: planned
estimate: L
---

# PBI-126 — Joint model (types, DOF, limits)

**Milestone:** M12 Assembly: Constraints, Joints, Motion & Representations  ·  **Feature:** F02 Joints

## Goal

Implement the joint object family (rigid/rotational/slider/cylindrical/planar/ball) with joint origins, DOF, and limits.

## Scope / work

- `AssemblyJointDefinition` per type.
- Joint origin alignment; flip.
- Linear/angular limits.

## API contracts (interfaces / enums / collections)

- `AssemblyJoint(s)`,`AssemblyJointDefinition`,`DSJoint(s)`

## Acceptance criteria

- A rotational joint allows one rotational DOF within limits.

## Depends on

_See feature dependencies._
