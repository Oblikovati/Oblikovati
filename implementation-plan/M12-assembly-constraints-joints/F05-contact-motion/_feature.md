---
milestone: M12
feature: F05
name: Contact & Motion
status: planned
---

# M12 · F05 — Contact & Motion

Real-time motion behaviors: a contact solver that prevents interpenetration during dragging, static interference detection, and flexible subassemblies.

## In scope

- Contact solver (contact sets).
- Static interference detection.
- Flexible subassembly behavior.

## Out of scope

_None._

## Key API contracts delivered

- `ContactSolver`,`ContactSet(s)`,`InterferenceResults`,`InterferenceResult`

## Depends on

F02,F04.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-130](PBI-130-contact-interference.md) | Contact solver & interference detection |
