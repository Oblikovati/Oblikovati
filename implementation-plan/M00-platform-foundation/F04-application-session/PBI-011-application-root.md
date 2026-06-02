---
milestone: M00
feature: F04
pbi: PBI-011
title: Application root object & service directory
status: planned
estimate: M
---

# PBI-011 — Application root object & service directory

**Milestone:** M00 Platform Foundation & Interop  ·  **Feature:** F04 Application Session & Lifecycle

## Goal

Implement the `Application` object as the owner/exposer of global services (factories, managers, collections) — a directory, not a brain.

## Scope / work

- Expose `Documents`, `TransientGeometry`, `TransientObjects`, `TransactionManager`, managers.
- Pointer discipline: one root reaches everything.
- Multi-instance support.

## API contracts (interfaces / enums / collections)

- `Application`,`_Application`

## Acceptance criteria

- From the root, every global service is reachable.
- Two roots in one process are independent.

## Depends on

_See feature dependencies._
