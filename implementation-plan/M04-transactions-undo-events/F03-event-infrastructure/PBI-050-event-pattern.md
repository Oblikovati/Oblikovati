---
milestone: M04
feature: F03
pbi: PBI-050
title: Object/sink event composition & subscription
status: planned
estimate: M
---

# PBI-050 — Object/sink event composition & subscription

**Milestone:** M04 Transactions, Undo & Events  ·  **Feature:** F03 Event Infrastructure

## Goal

Implement the object/sink split and subscription model so any subsystem can expose discoverable, subscribable events uniformly.

## Scope / work

- `XEventsObject`+`XEventsSink_Event`→`XEvents`.
- Add/remove handler; multicast.
- Delegate signature template.

## API contracts (interfaces / enums / collections)

- `*EventsObject`,`*EventsSink_Event`,`*Events`

## Acceptance criteria

- A subscriber receives fired events.
- The events object remains usable as a normal object.

## Depends on

_See feature dependencies._
