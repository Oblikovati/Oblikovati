---
milestone: M12
feature: F03
pbi: PBI-127
title: iMate definitions & auto-pairing results
status: planned
estimate: M
---

# PBI-127 — iMate definitions & auto-pairing results

**Milestone:** M12 Assembly: Constraints, Joints, Motion & Representations  ·  **Feature:** F03 iMates & Drive

## Goal

Implement iMate definitions (named, typed mate stubs) and the matching that produces iMate results when components are placed.

## Scope / work

- `iMateDefinition` types; composite.
- Auto-pair by name on placement → `iMateResult`.

## API contracts (interfaces / enums / collections)

- `iMateDefinition(s)`,`iMateResult(s)`,`AngleiMate/FlushiMate/InsertiMate/CompositeiMateDefinition`

## Acceptance criteria

- Placing a component with matching-named iMates auto-constrains it.

## Depends on

_See feature dependencies._
