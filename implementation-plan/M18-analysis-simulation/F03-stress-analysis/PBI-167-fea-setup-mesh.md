---
milestone: M18
feature: F03
pbi: PBI-167
title: FEA study setup, meshing, loads & constraints
status: planned
estimate: XL
---

# PBI-167 — FEA study setup, meshing, loads & constraints

**Milestone:** M18 Analysis, Measurement & Simulation  ·  **Feature:** F03 Stress Analysis (FEA)

## Goal

Implement the FEA study model: geometry simplification, FE meshing with controls, and definition of materials, loads, constraints, and contacts.

## Scope / work

- Study/material assignment.
- Mesh generation + local controls.
- Loads/constraints/contacts.

## API contracts (interfaces / enums / collections)

- FEA study/mesh/load/constraint API,`Material`

## Acceptance criteria

- A part meshes with refinement controls and accepts loads/constraints.

## Depends on

_See feature dependencies._
