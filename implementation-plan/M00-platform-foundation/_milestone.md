---
milestone: M00
name: Platform Foundation & Interop
status: planned
---

# M00 — Platform Foundation & Interop

The bedrock: the native↔managed interop boundary, the core object-model primitives every other object builds on (RTTI type system, parent/identity back-pointers), the transient container types used throughout method signatures, and the `Application` session root and its lifecycle. Nothing models geometry yet; this milestone makes it *possible* to expose a native kernel as a coherent, navigable, weakly-and-strongly typed object graph.

## Goals

- A single, centralized interop layer that hides all marshaling.
- A stable RTTI scheme (`ObjectTypeEnum`) and universal `Type`/`Parent`/`Application` members.
- Transient containers (`ObjectCollection`, `NameValueMap`, enumerators) usable in any signature.
- An `Application` root object with deterministic startup/shutdown and a headless mode.

## In scope

- Interop host bootstrap and object lifetime/identity across the boundary.
- `ObjectTypeEnum` design and the `Type` discriminator on every object.
- Variant (`object`) handling, `ref`/`out` multi-return conventions, error propagation.
- `Application`/`ApprenticeServer` roots, `SoftwareVersion`, locale, options scaffolding.

## Out of scope (handled elsewhere)

- Geometry math (M01), documents (M03), transactions/events (M04), UI (M05).

## Exit criteria

- A native object can be created, round-tripped, type-queried, and reached from the app root.
- Multiple independent `Application` instances (incl. headless) can coexist in one process.
- Collections/maps/enumerators pass cleanly across the interop boundary with preserved identity.

## Depends on

_Nothing — this is a foundation milestone._

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Native/Managed Interop Layer](F01-interop-layer/_feature.md) | 4 | Centralized marshaling host between the managed API and native kernel. |
| **F02** | [Core Object Model & RTTI](F02-object-model-rtti/_feature.md) | 3 | The type discriminator and universal members on every object. |
| **F03** | [Transient Object Collections](F03-transient-containers/_feature.md) | 3 | The container/value vocabulary used across all API signatures. |
| **F04** | [Application Session & Lifecycle](F04-application-session/_feature.md) | 3 | The process root object, versioning, locale, and headless mode. |
