---
milestone: M12
feature: F01
name: Assembly Constraints
status: planned
---

# M12 · F01 — Assembly Constraints

The classic constraint set that positions occurrences relative to each other, backed by the assembly constraint solver with limits, redundancy detection, and health reporting on over-constrained assemblies.

## In scope

- Mate/flush/angle/tangent/insert/symmetry/rotate.
- `ConstraintLimits`.
- Assembly solver; redundancy; health.

## Out of scope

_None._

## Key API contracts delivered

- `AssemblyConstraint`,`AssemblyConstraints`,`MateConstraint`,`FlushConstraint`,`AngleConstraint`,`TangentConstraint`,`InsertConstraint`,`AssemblySymmetryConstraint`
- `ConstraintLimits`,`HealthStatusEnum`

## Depends on

M11.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-125](PBI-125-assembly-constraints.md) | Assembly constraint set & solver |
