---
milestone: M12
feature: F02
name: Joints
status: planned
---

# M12 · F02 — Joints

The simplified joint model (one relationship establishing a degree-of-freedom set) with the standard joint types, DOF, and motion limits.

## In scope

- `AssemblyJoint` types & DOF.
- Joint origins; limits; flip/align.
- DS joints.

## Out of scope

_None._

## Key API contracts delivered

- `AssemblyJoint`,`AssemblyJoints`,`AssemblyJointDefinition`,`DSJoint`,`DSJoints`,`DSJointDefinition`

## Depends on

F01.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-126](PBI-126-joints.md) | Joint model (types, DOF, limits) |
