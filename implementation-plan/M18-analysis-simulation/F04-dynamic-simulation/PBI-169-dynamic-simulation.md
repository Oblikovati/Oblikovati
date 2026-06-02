---
milestone: M18
feature: F04
pbi: PBI-169
title: Dynamic (multibody) simulation
status: planned
estimate: XL
---

# PBI-169 — Dynamic (multibody) simulation

**Milestone:** M18 Analysis, Measurement & Simulation  ·  **Feature:** F04 Dynamic Simulation

## Goal

Implement rigid-body dynamic simulation: build a mechanism from joints/constraints, apply forces/motions, integrate over time, and output result graphs.

## Scope / work

- Mechanism extraction from joints/constraints.
- Force/torque/motion drivers; gravity.
- Time-step integration; output grapher; reaction export.

## API contracts (interfaces / enums / collections)

- `DynamicSimulation(s)`,`AssemblyJoint`(M12)

## Acceptance criteria

- A four-bar mechanism simulates motion and outputs joint-force vs time graphs.

## Depends on

_See feature dependencies._
