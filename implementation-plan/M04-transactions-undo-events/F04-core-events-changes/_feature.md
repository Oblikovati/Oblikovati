---
milestone: M04
feature: F04
name: Core Events & Change Manager
status: planned
---

# M04 · F04 — Core Events & Change Manager

The foundational event sets fired by the platform — application, document, and modeling events — plus the `ChangeManager`/`ChangeProcessor` that records and reacts to model changes.

## In scope

- `ApplicationEvents` (new/open/save/close/activate/quit…).
- `DocumentEvents`, `ModelingEvents`.
- `ChangeManager`/`ChangeProcessor`, `ChangeDefinition`.

## Out of scope

_None._

## Key API contracts delivered

- `ApplicationEvents`,`DocumentEvents`,`ModelingEvents`
- `ChangeManager`,`ChangeProcessor`,`ChangeDefinition`,`ChangeManagerProcessControl`

## Depends on

F03.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-052](PBI-052-application-document-events.md) | Application & Document event sets |
| [PBI-053](PBI-053-modeling-change-manager.md) | ModelingEvents & ChangeManager/ChangeProcessor |
