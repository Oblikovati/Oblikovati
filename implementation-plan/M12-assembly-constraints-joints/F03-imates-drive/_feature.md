---
milestone: M12
feature: F03
name: iMates & Drive
status: planned
---

# M12 · F03 — iMates & Drive

iMates (pre-defined, named mate definitions on a component that auto-pair on placement) and the drive mechanism that animates a constraint/joint through a value range for motion study.

## In scope

- `iMateDefinition`/`iMateResult`; composite iMates.
- `DriveConstraintSettings`/joint drive.
- Collision detection during drive.

## Out of scope

_None._

## Key API contracts delivered

- `iMateDefinition`,`iMateDefinitions`,`iMateResult(s)`,`AngleiMateDefinition`,`FlushiMateDefinition`,`InsertiMateDefinition`,`CompositeiMateDefinition`
- `DriveConstraintSettings`

## Depends on

F01,F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-127](PBI-127-imates.md) | iMate definitions & auto-pairing results |
| [PBI-128](PBI-128-drive.md) | Constraint/joint drive & animation |
