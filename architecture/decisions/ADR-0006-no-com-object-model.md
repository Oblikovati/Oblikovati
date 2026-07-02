# ADR-0006 — Drop the COM object model (RTTI / variants / dual interfaces)

**Status:** accepted · **Replaces:** M00-F02 (`ObjectTypeEnum` RTTI), variant
`object` parameters, `_X : X` dual interfaces, `Parent`/`Application` back-pointers,
`IEnumerable` 1-based collections.

> **Amendment (2026-06, [ADR-0018](ADR-0018-apache-api-contract-module.md)).** The
> "defined twice, deliberately" consequence below is superseded: the public surface
> is now defined **once** in the Apache-2.0 `/api` module — `api/contract` (in-proc
> Go interfaces) and `api/wire` (the JSON contract) live side by side in one place,
> and `/source` implements them. The stable-`TypeID`-for-persistence decision here is
> unchanged.

> **Editor's note (2026-07, #1661 / M40 audit D4).** Paths in this ADR predate the
> repo-root migration: the GPL application module written here as `/source` now
> lives directly at the repository root (`kernel/`, `model/`, `app/`, `head/`, …),
> and the Apache-2.0 `/api` module is the sibling `Oblikovati.API` repository.
> The record below is preserved as written.

## Decision

Replace the COM object-model conventions with idiomatic Go, while **preserving the
two things they actually bought us**: stable identity for persistence, and a way
to handle heterogeneous selections.

| COM mechanism | Purpose it served | Go replacement |
|---|---|---|
| `ObjectTypeEnum Type` on every object | runtime type id for branching + persistence | **In-proc:** Go type switches / interfaces (compile-time). **Persistence/RPC only:** a stable `TypeID uint32` registered per serializable type. |
| `object` / `VARIANT` params & returns | pass "anything" (selection, options) | **Concrete types + generics** in-proc; `any` or protobuf `oneof` *only* at the RPC seam. |
| `_X : X` dual interfaces | additive versioning without breaking ABI | **Interface composition** + new methods on concrete types; protobuf field numbers at the seam. No ABI to preserve in-proc. |
| `Parent` / `Application` on every object | reach the app root / owner | **Runtime mediator** passed explicitly (realtime-3d §1). Model *tree* nodes still keep an explicit `parent` link where the domain is genuinely a tree (feature→def→document); but there is **no global `Application` singleton**. |
| `IEnumerable`, 1-based `Item[i]` | typed collections | `Collection[T any]` generic, **0-based**, range-able. |
| `NameValueMap` options bag | extensible optional args | typed **option structs** with sensible zero-values in-proc; `map<string,Value>` only at the RPC seam. |
| `out HandlingCodeEnum` veto | event veto | handler returns `error`/`Veto(reason)` (core/06). |

## Identity is the part we keep seriously

The COM `Type` enum was *also* doing persistence duty (it's baked into saved files
and reference keys). We must not lose that:

- Every **serializable** type registers a stable `TypeID` in a central registry
  (`registry/types`). These integers are **immutable forever** (same rule as
  `ObjectTypeEnum`'s stable numbers) and are what appear in files, reference keys,
  and the gRPC wire — never Go type names (which can be refactored).
- In-process, we never branch on `TypeID`; we use Go interfaces/type switches.
  `TypeID` exists *only* at the persistence and RPC boundaries.

So: **static types for behavior, stable integer IDs for storage.** This cleanly
separates the two jobs COM's single `Type` enum was overloading.

## Heterogeneous selection without variants

A selection is `[]Entity` where `Entity` is a small interface (`ID()`,
`TypeID()`, `BoundingBox()`). Consumers type-switch:

```go
switch e := sel.(type) {
case *topo.Face:  ...
case *topo.Edge:  ...
case *sketch.Line: ...
}
```

No `VARIANT`, no `kEdgeObject` integer compares — the compiler checks exhaustiveness
pressure and the cases are real types.

## Consequences

- The entire **M00 interop milestone disappears** (no managed↔native marshaling;
  ADR-0002 made the kernel pure Go). What remains of M00 is: the runtime mediator
  (core/00), the type registry (this ADR), generic collections, and the app session.
- The public surface is defined **twice, deliberately**: ergonomic Go interfaces
  in-proc, and the `.proto` services for out-of-proc add-ins (ADR-0003). They stay
  parallel by construction.
