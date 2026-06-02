---
milestone: M18
name: Analysis, Measurement & Simulation
status: planned
---

# M18 — Analysis, Measurement & Simulation

Engineering analysis on top of the model: measurement and mass properties, interference/validation, finite-element stress analysis (mesh/loads/constraints/solve/results), dynamic (multibody) simulation, and tolerance/GD&T analysis. These consume topology (M07), assemblies (M11), and material physical properties (M16).

## Goals

- Measurement tools and mass/physical properties.
- Interference and model-validation checks.
- Finite-element stress analysis with results.
- Dynamic simulation of mechanisms.
- Tolerance/GD&T stack analysis.

## In scope

- `MeasureTools`; `MassProperties`; min-distance.
- `InterferenceResults`; validation/model-doctor.
- FEA: mesh/loads/constraints/materials/solve/results.
- `DynamicSimulation`: joints/forces/motion/graphs.
- `ModelTolerance`/`DatumReferenceFrame`; tolerance stacks.

## Out of scope (handled elsewhere)

- Static assembly interference UI (M12 provides the core).
- Material definitions (M16).

## Exit criteria

- Mass/center-of-mass/inertia computed for a part/assembly.
- An FEA study meshes, solves, and reports stress/displacement.
- A mechanism simulates motion and outputs joint force graphs.

## Depends on

M07, M11, M16

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Measurement & Mass Properties](F01-measurement-mass/_feature.md) | 2 | Measure tools and mass/physical properties. |
| **F02** | [Interference & Validation](F02-interference-validation/_feature.md) | 1 | Interference, clearance, and model health checks. |
| **F03** | [Stress Analysis (FEA)](F03-stress-analysis/_feature.md) | 2 | Mesh, loads, constraints, solve, results. |
| **F04** | [Dynamic Simulation](F04-dynamic-simulation/_feature.md) | 1 | Multibody motion simulation of mechanisms. |
| **F05** | [Tolerance & GD&T Analysis](F05-tolerance-analysis/_feature.md) | 1 | Model tolerances, datum frames, stack analysis. |
