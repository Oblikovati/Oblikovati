---
milestone: M03
feature: F05
pbi: PBI-042
title: ReferenceKeyManager: keys, contexts, bind/can-bind
status: planned
estimate: XL
---

# PBI-042 — ReferenceKeyManager: keys, contexts, bind/can-bind

**Milestone:** M03 Documents, Persistence & Identity  ·  **Feature:** F05 Persistent Identity (Reference Keys)

## Goal

Implement opaque reference keys encoding a derivation path to topology, key contexts (persistable snapshots), and bind/can-bind resolution that may legitimately fail.

## Scope / work

- `CreateKeyContext`/`Release`; `Save/LoadContextToArray`.
- `GetReferenceKey` on Face/Edge/Vertex/Feature/Parameter.
- `BindKeyToObject`/`CanBindKeyToObject` with match-type.
- `KeyToString`/`StringToKey`.

## API contracts (interfaces / enums / collections)

- `ReferenceKeyManager`

## Acceptance criteria

- A key to a face rebinds after a recompute that recreates the face.
- `CanBindKeyToObject` returns false when topology genuinely vanished.
- Keys persist across save/close/reopen via context save/load.

## Depends on

_See feature dependencies._

## Notes

This is the hardest, most schedule-critical PBI in the plan. Topological naming quality determines whether feature edits 'lose' references. Budget research time; design before features depend on it (M08+).
