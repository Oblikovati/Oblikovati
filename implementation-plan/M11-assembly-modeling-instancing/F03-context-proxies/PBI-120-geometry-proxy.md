---
milestone: M11
feature: F03
pbi: PBI-120
title: Geometry proxies & context transforms
status: planned
estimate: L
---

# PBI-120 — Geometry proxies & context transforms

**Milestone:** M11 Assembly Modeling & Instancing  ·  **Feature:** F03 Context Proxies

## Goal

Implement a generic proxy mechanism (native entity + occurrence context + path transform) exposed via `CreateGeometryProxy`, so any entity can be viewed in assembly space, with all `*Proxy` types generated over it.

## Scope / work

- Generic proxy = entity + context + transform.
- `CreateGeometryProxy(native)→proxy` and reverse.
- `Type` matches underlying; geometry transformed by path.
- Generated `*Proxy` surface.

## API contracts (interfaces / enums / collections)

- `CreateGeometryProxy`,`FaceProxy`,`EdgeProxy`,`*Proxy`

## Acceptance criteria

- A part face viewed via an occurrence reports assembly-space geometry.
- Proxy and native are type-distinct so part-space geometry can't be misused in assembly space.

## Depends on

_See feature dependencies._

## Notes

Implement proxying once generically; do not hand-author 275 proxy types. The architectural win is the explicit native≠proxy type distinction.
