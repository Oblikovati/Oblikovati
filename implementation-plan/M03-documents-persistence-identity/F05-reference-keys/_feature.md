---
milestone: M03
feature: F05
name: Persistent Identity (Reference Keys)
status: planned
---

# M03 · F05 — Persistent Identity (Reference Keys)

The single most load-bearing kernel mechanism: opaque, serializable reference keys that re-resolve to 'the same' topological entity after the B-rep is rebuilt or the document is reloaded. Topological naming, not pointer identity.

## In scope

- `ReferenceKeyManager` bind/can-bind.
- Key contexts (versioned snapshots) save/load.
- `GetReferenceKey` on entities; key↔string.
- Bind-failure → health-sick handling.

## Out of scope

_None._

## Key API contracts delivered

- `ReferenceKeyManager`
- `entity.GetReferenceKey(ref byte[], KeyContext)`
- key serialization APIs

## Depends on

F03.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-042](PBI-042-reference-key-manager.md) | ReferenceKeyManager: keys, contexts, bind/can-bind |
| [PBI-043](PBI-043-reference-loss-policy.md) | Reference-loss propagation policy |
