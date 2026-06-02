# ADR-0003 — Hybrid extensibility: in-proc registries + out-of-proc gRPC add-ins

**Status:** accepted (user decision); third-party transport **amended by
[ADR-0016](ADR-0016-shared-library-addins-mcp-bridge.md)**; the public contract is
now the Apache-2.0 `/api` module **([ADR-0018](ADR-0018-apache-api-contract-module.md))**
· **Replaces:** COM `ApplicationAddInServer` in-process add-ins (M05-F01).

> **Amendment (2026-06, [ADR-0016](ADR-0016-shared-library-addins-mcp-bridge.md)).**
> The first add-in (`oblikovati-mcp-bridge`, an MCP automation bridge) realized the
> third-party mechanism **in-process** as a C-shared library (`.so`/`.dll`) loaded
> over a small **C ABI**, *not* out-of-process gRPC — a deliberate product decision
> to drive the live GUI in real time before the gRPC `api/` layer exists. This
> revisits the "rejected option" below; we accept its costs (shared address space,
> no crash isolation, toolchain coupling) for a trusted first-party add-in. The
> boundary carries an in-process **JSON method contract** (`commands.*`,
> `documents.*`, `parameters.*`, `model.*`, `sketch.*`, `features.*`) that is
> transport-agnostic, so gRPC remains a viable *future* transport behind the same
> contract — deferred, not abandoned. The **in-proc registries for first-party
> features (below) are unchanged.** See ADR-0016 for the full decision.

## Decision

Two distinct extension mechanisms, by audience:

1. **First-party / built-in features** self-register **in-process** via Go `init()`
   into typed **registries** (realtime-3d skill §9). Workspaces (Sketch, Part,
   Assembly, Drawing), commands, feature types, translators, property editors all
   register themselves. Adding one = add a package + one blank import.

2. **Third-party add-ins** run **out-of-process** and talk to the host over a
   stable **gRPC** contract. The `.proto` service definition *is* the modern
   public API — the successor to the COM type library / `Oblikovati.Contracts`.

## Why

COM in-process add-ins are Windows-only and share the host's address space (an
add-in crash kills the app). The hybrid fixes both while staying Go-friendly:

- **In-proc registries** give first-party code native speed and direct typed
  access to the model — no serialization tax for the code we ship ourselves.
- **Out-of-proc gRPC** gives third parties:
  - **language independence** — write add-ins in Python, C#, Rust, TS, anything
    with a gRPC client (a strict superset of COM's reach);
  - **crash isolation** — a misbehaving add-in cannot corrupt the document or
    crash the host; the host detects the dropped connection and degrades;
  - **a versioned, explicit boundary** — protobuf field/version rules replace the
    `_X : X` dual-interface dance;
  - **security/sandboxing** — add-ins get only what the gRPC surface exposes.

Why not Go `plugin`/.so (the rejected option): no Windows support, brittle exact-
toolchain matching, shared crashes — the worst of both worlds for our targets.

## Shape of the gRPC contract

The service mirrors the kept domain model, not the COM surface:

```proto
service Documents   { rpc Open; rpc Save; rpc Create; rpc List; ... }
service Model       { rpc GetFeatures; rpc GetParameters; rpc Tessellate; ... }
service Edit        { rpc Begin(Transaction); rpc AddFeature(Definition); rpc Commit; }
service Events      { rpc Subscribe(Filter) returns (stream Event); }   // before/after, veto via response
service Selection   { rpc Get; rpc Set; rpc Pick; }
service ClientGfx   { rpc Draw(OverlayGraphics); }                      // add-in overlays
```

- **Events stream** to the add-in; for *vetoable* before-events the add-in replies
  on a paired call with `allow|veto{reason}` within a deadline (replaces
  `out HandlingCodeEnum`). After the deadline the host proceeds (no add-in may
  hang the host — a key improvement over COM).
- **Edits are transactional**: `Begin → Add*/Set* → Commit|Abort` maps onto the
  command-history model (ADR-0006 / [core/06](../core/06-transactions-and-events.md)).
- The add-in never holds a model pointer; it holds **stable IDs / reference keys**
  (ADR-0006) it passes back across the boundary.

## Costs / mitigations

- **Per-call latency** for chatty add-ins → batch APIs, streaming, and a
  client-side façade library that feels like local objects.
- **Two mechanisms to maintain** → the in-proc registry interfaces and the gRPC
  services are kept deliberately parallel; first-party features may *also* be
  exercised through the gRPC layer in tests to prove API completeness (the
  dogfood principle, realtime-3d §12).

## Consequences

See [core/07](../core/07-extensibility.md). The `api/` package owns the `.proto`
and the in-proc host adapter; `addins/` owns process lifecycle and supervision.
