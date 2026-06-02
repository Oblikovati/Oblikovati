---
milestone: M11
feature: F03
name: Context Proxies
status: planned
---

# M11 · F03 — Context Proxies

The proxy mechanism that views a part-definition entity (face/edge/work-feature) through an occurrence context so it reports assembly-space geometry while still identifying the underlying native entity — the bridge required for assembly features, mates, and measurement.

## In scope

- `CreateGeometryProxy`.
- `FaceProxy`/`EdgeProxy`/`WorkPointProxy`/etc.
- Native vs proxy distinction; `ContextDefinition`.

## Out of scope

_None._

## Key API contracts delivered

- `ComponentOccurrence.CreateGeometryProxy`
- `FaceProxy`,`EdgeProxy`,`VertexProxy`,`WorkPlaneProxy`,`WorkAxisProxy`,`WorkPointProxy`,`SurfaceBodyProxy` (275 `*Proxy` types)

## Depends on

F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-120](PBI-120-geometry-proxy.md) | Geometry proxies & context transforms |
