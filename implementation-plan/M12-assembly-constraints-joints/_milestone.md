---
milestone: M12
name: Assembly: Constraints, Joints, Motion & Representations
status: planned
---

# M12 — Assembly: Constraints, Joints, Motion & Representations

Assembly relationships and states: positional constraints (mate/flush/angle/tangent/insert) and the newer joint model, iMates, drive/animation, contact and interference, and the three representation families (design-view, positional, level-of-detail) plus model states. The assembly constraint solver positions components and reports redundancy/health.

## Goals

- Assembly constraints with a solver and redundancy/health reporting.
- Joints (rigid/rotational/slider/cylindrical/planar/ball) with DOF & limits.
- iMates and constraint/joint drive for motion/animation.
- Design-view, positional, and level-of-detail representations + model states.
- Contact solver, interference, and flexible assemblies.

## In scope

- Mate/flush/angle/tangent/insert/symmetry constraints; limits; solver.
- `AssemblyJoint` types; DOF; limits.
- `iMateDefinition`/`iMateResult`; `DriveConstraint`/joint settings.
- Design-view/positional/LOD representations; model states.
- Contact solver; interference; flexible/motion.

## Out of scope (handled elsewhere)

- Drawing of assemblies (M14).
- Dynamic simulation (M18).

## Exit criteria

- Constraining two parts positions them and the solver reports DOF/redundancy.
- A rotational joint moves within limits; drive animates it.
- Switching positional/LOD representations changes assembly state.

## Depends on

M11

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Assembly Constraints](F01-assembly-constraints/_feature.md) | 1 | Mate, flush, angle, tangent, insert, symmetry + solver. |
| **F02** | [Joints](F02-joints/_feature.md) | 1 | Rigid/rotational/slider/cylindrical/planar/ball joints. |
| **F03** | [iMates & Drive](F03-imates-drive/_feature.md) | 2 | Reusable mate definitions and constraint/joint drive. |
| **F04** | [Representations & Model States](F04-representations/_feature.md) | 1 | Design-view, positional, level-of-detail reps; model states. |
| **F05** | [Contact & Motion](F05-contact-motion/_feature.md) | 1 | Contact solver, interference, flexible assemblies. |
