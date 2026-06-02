---
milestone: M04
feature: F03
name: Event Infrastructure
status: planned
---

# M04 · F03 — Event Infrastructure

The uniform eventing pattern reused everywhere: an `XEventsObject` (normal members) plus an `XEventsSink_Event` (event declarations) composed into `XEvents`, delivering before/after timing and letting before-handlers veto via a `HandlingCode` and a `NameValueMap` context.

## In scope

- Object/sink interface composition.
- `EventTimingEnum` before/after.
- `HandlingCodeEnum` veto; context map.
- Delegate signature conventions.

## Out of scope

_None._

## Key API contracts delivered

- `*EventsObject`,`*EventsSink_Event`,`*Events`
- `EventTimingEnum`,`HandlingCodeEnum`
- event-handler delegates

## Depends on

M00.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-050](PBI-050-event-pattern.md) | Object/sink event composition & subscription |
| [PBI-051](PBI-051-before-after-veto.md) | Before/after timing & HandlingCode veto |
