---
milestone: M18
feature: F04
name: Dynamic Simulation
status: planned
---

# M18 · F04 — Dynamic Simulation

Rigid-body dynamic simulation of assemblies: converting joints/constraints to a mechanism, applying forces/torques/motions, integrating motion over time, and outputting graphs (position/velocity/force) that can drive FEA loads.

## In scope

- `DynamicSimulation` mechanism from joints.
- Forces/torques/motions; gravity.
- Time integration; output grapher.
- Export reaction loads to FEA.

## Out of scope

_None._

## Key API contracts delivered

- `DynamicSimulation`,`DynamicSimulations`,`AssemblyJoint`(M12)

## Depends on

M12.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-169](PBI-169-dynamic-simulation.md) | Dynamic (multibody) simulation |
