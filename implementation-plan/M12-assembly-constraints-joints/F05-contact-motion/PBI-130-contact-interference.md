---
milestone: M12
feature: F05
pbi: PBI-130
title: Contact solver & interference detection
status: planned
estimate: L
---

# PBI-130 — Contact solver & interference detection

**Milestone:** M12 Assembly: Constraints, Joints, Motion & Representations  ·  **Feature:** F05 Contact & Motion

## Goal

Implement the contact solver (components in contact sets resist interpenetration when dragged) and static interference analysis between occurrences.

## Scope / work

- Contact set membership; real-time contact during drag.
- `InterferenceResults` volume/where.
- Flexible subassembly DOF exposure.

## API contracts (interfaces / enums / collections)

- `ContactSolver`,`ContactSet(s)`,`InterferenceResults`,`InterferenceResult`

## Acceptance criteria

- Dragging a contacting part stops at contact; interference reports overlapping volumes.

## Depends on

_See feature dependencies._
