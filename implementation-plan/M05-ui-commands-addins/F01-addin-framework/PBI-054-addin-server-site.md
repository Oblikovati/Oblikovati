---
milestone: M05
feature: F01
pbi: PBI-054
title: Add-in server/site lifecycle & registration
status: planned
estimate: M
---

# PBI-054 — Add-in server/site lifecycle & registration

**Milestone:** M05 Application UI, Commands, Interaction & Add-in Platform  ·  **Feature:** F01 Add-in Framework

## Goal

Implement add-in discovery/registration and the activate/deactivate lifecycle with a host site providing services.

## Scope / work

- `ApplicationAddInServer.Activate/Deactivate`.
- `ApplicationAddInSite` (app handle, services).
- Manifest-based discovery; load/unload.

## API contracts (interfaces / enums / collections)

- `ApplicationAddInServer`,`ApplicationAddInSite`,`ApplicationAddIns`,`ApplicationAddIn`

## Acceptance criteria

- A registered add-in activates on startup and cleans up on deactivate.
- The registry lists installed add-ins.

## Depends on

_See feature dependencies._
