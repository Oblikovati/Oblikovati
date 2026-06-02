# Oblikovati — Parametric CAD Implementation Plan

A dependency-ordered roadmap to build a parametric, feature-based, history-driven MCAD application (Inventor-class) on a greenfield codebase, and the public automation API that drives it. Scope is derived from the Oblikovati.Contracts surface (2200+ interfaces) which mirrors the Autodesk Inventor 2026 programming interface. Each PBI names the concrete contracts it must deliver so the plan can be consumed directly during implementation.

## How this plan is organized

- **Milestones** (`Mxx-*`) are large, dependency-ordered capability blocks. Each has a
  `_milestone.md` defining scope, goals, exit criteria, and the features it contains.
- **Features** (`Fxx-*`) are coherent sub-capabilities inside a milestone. Each has a
  `_feature.md` defining scope-in / scope-out and the PBIs that realize it.
- **PBIs** (`PBI-xxx-*.md`) are individual, implementable backlog items. Each names the
  concrete API contracts (interfaces/enums/collections) it must deliver, acceptance
  criteria, and dependencies.

Read [CONVENTIONS.md](CONVENTIONS.md) first — it defines the file template, the
"Definition→Add→Feature" pattern every modeling PBI follows, and the cross-cutting
rules (units, identity, transactions, events) that apply to every milestone.

## Milestone roadmap (build in this order)

| ID | Milestone | Features | PBIs | Depends on |
|----|-----------|:-------:|:----:|------------|
| **M00** | [Platform Foundation & Interop](M00-platform-foundation/_milestone.md) | 4 | 13 | — |
| **M01** | [Math & Transient Geometry](M01-math-transient-geometry/_milestone.md) | 4 | 9 | M00 |
| **M02** | [Units, Parameters & Expressions](M02-units-parameters-expressions/_milestone.md) | 4 | 10 | M00 |
| **M03** | [Documents, Persistence & Identity](M03-documents-persistence-identity/_milestone.md) | 6 | 13 | M00, M02 |
| **M04** | [Transactions, Undo & Events](M04-transactions-undo-events/_milestone.md) | 4 | 8 | M03 |
| **M05** | [Application UI, Commands, Interaction & Add-in Platform](M05-ui-commands-addins/_milestone.md) | 5 | 12 | M04 |
| **M06** | [2D/3D Sketching & Constraint Solver](M06-sketching-constraints/_milestone.md) | 6 | 13 | M01, M02, M05 |
| **M07** | [B-Rep Modeling Kernel & Topology](M07-brep-kernel-topology/_milestone.md) | 4 | 8 | M01, M03 |
| **M08** | [Part Modeling: Sketched & Work Features](M08-part-sketched-work-features/_milestone.md) | 4 | 12 | M06, M07 |
| **M09** | [Part Modeling: Dress-up & Pattern Features](M09-part-dressup-pattern-features/_milestone.md) | 4 | 10 | M08 |
| **M10** | [Surfacing & Freeform Modeling](M10-surfacing-freeform/_milestone.md) | 4 | 8 | M08 |
| **M11** | [Assembly Modeling & Instancing](M11-assembly-modeling-instancing/_milestone.md) | 5 | 8 | M07, M08 |
| **M12** | [Assembly: Constraints, Joints, Motion & Representations](M12-assembly-constraints-joints/_milestone.md) | 5 | 6 | M11 |
| **M13** | [Sheet Metal](M13-sheet-metal/_milestone.md) | 4 | 6 | M08, M09 |
| **M14** | [Drawing & Documentation](M14-drawing-documentation/_milestone.md) | 5 | 9 | M07, M11 |
| **M15** | [Design Automation: iPart/iAssembly, Tables & iLogic](M15-design-automation/_milestone.md) | 4 | 5 | M08, M11 |
| **M16** | [Visualization, Appearances, Styles & Presentations](M16-visualization-presentation/_milestone.md) | 4 | 7 | M07, M11 |
| **M17** | [Interoperability & Translation](M17-interoperability-translation/_milestone.md) | 4 | 6 | M07 |
| **M18** | [Analysis, Measurement & Simulation](M18-analysis-simulation/_milestone.md) | 5 | 7 | M07, M11, M16 |

## Dependency spine

The non-negotiable foundation layers (ruinous to retrofit) come first:

```
M00 Platform/Interop ─▶ M01 Math/Geometry ─▶ M02 Units/Parameters ─▶ M03 Documents/Identity
                                                                          │
                                              M04 Transactions/Events ◀───┤
                                                      │                   │
                            M05 UI/Commands/Add-ins ◀─┘                   │
                                                                          ▼
                                            M06 Sketching ─▶ M07 B-Rep Kernel
                                                                  │
                            M08 Sketched+Work Features ◀──────────┤
                            M09 Dress-up+Pattern Features ◀────────┘
                                      │
              ┌───────────────────────┼───────────────────────────┐
              ▼                       ▼                            ▼
        M10 Surfacing         M11/M12 Assembly                M13 Sheet Metal
              └───────────────────────┼───────────────────────────┘
                                      ▼
                          M14 Drawing/Documentation
                                      ▼
        M15 Design Automation · M16 Visualization · M17 Interop · M18 Analysis/Sim
```

Milestones M05 (UI/Add-ins) and M16 (Visualization) are *platform* milestones that can be
developed in parallel once their dependencies land; command/ribbon population and view
generation for each modeling capability are delivered incrementally alongside M06–M13.

## Totals

- Milestones: **19**
- Features: **85**
- PBIs: **170**
