# ADR-0028 — Embedded Lua scripting runtime (pure-Go, sandboxed, wire-mirrored)

**Status:** Proposed (2026-06-07)
**Context:** First-party, integral scripting in the GPL-v2 app (not an add-in).
Implementation plan: [lua-scripting-plan.md](../lua-scripting-plan.md).
**Builds on / refines:** [ADR-0013](ADR-0013-ilogic-embedded-scripting.md) (embedded
scripting drives the public API; named "Lua via a pure-Go VM" as a candidate),
[ADR-0016](ADR-0016-shared-library-addins-mcp-bridge.md) (session-goroutine dispatch),
[ADR-0018](ADR-0018-apache-api-contract-module.md) (`api/wire` is the automation
surface), [ADR-0008](ADR-0008-cgo-boundary.md) (cgo confined to the platform edge).

## Problem

Users need to automate the model with scripts: drive parameters, build sketches and
features, batch operations — from the GUI, the CLI, and (later) the MCP bridge. The
hard constraints:

1. **A buggy or malicious script must NEVER crash, hang, or escape the host.**
2. **Everything in `oblikovati/api` must be callable** from a script, and stay callable
   automatically as the contract grows — no per-method scripting glue.
3. The model is **not goroutine-safe**; API calls must run on the **session
   goroutine** (ADR-0016), yet a looping script must not freeze the UI.

## Decision

### 1. Runtime: `github.com/yuin/gopher-lua` (pure-Go Lua 5.1 VM, MIT), behind a seam

Pure Go means **no cgo**, so a script bug is a Go panic we `recover()` — not a
segfault that kills the process. This is the only runtime choice that can *guarantee*
requirement #1, and it keeps the core cgo-free and headless-testable (ADR-0008). A cgo
**LuaJIT** binding is faster but introduces a real crash surface (FFI + C VM) that
directly violates #1; the workload is API-call-bound, not numeric, so we do not need
its speed. gopher-lua's instruction-count hook + `SetContext` give us the timeout and
cancellation primitives we need. MIT is GPL-v2-compatible.

The VM is wrapped behind a project-owned `script.Engine` interface (the only
`gopher-lua` importer is `script/gopherlua`), so the engine is swappable later **only
if** the crash/quota guarantees can be re-proven on the new path.

### 2. Safety / sandbox model

- **Opt-in stdlib.** The VM starts with nothing; we open only `base` (with
  `dofile/loadfile/load/loadstring/require/collectgarbage/print` stripped), `table`,
  `string`, `math` (deterministic seed). **`os`, `io`, `debug`, `package`/`require`,
  any FFI are NEVER opened** — no filesystem, OS, network, process, or code loading.
- **The only host capability is one global `oblikovati` table** (`call`, captured
  `print`, `methods`, plus Phase-2 typed groups). All host power flows through that one
  audited door, which itself only reaches the model via `Router.Handle` on the session
  goroutine — i.e. a script can do exactly what an add-in/MCP client can, no more.
- **Resource limits:** an **instruction-count quota** (opcode hook, checked every N
  ops), a **wall-clock timeout** (`context.WithTimeout`), and a **memory cap**
  (bounded call-stack/registry + allocation guard). Breach → a Lua error that unwinds
  the VM and returns as an error.
- **Cancellable:** the run's `context.Context` drives the quota hook and the dispatch
  submit, so Stop / Ctrl-C / shutdown unwinds promptly.
- **Panic containment:** `Engine.Run` wraps the VM in `defer recover()` → any panic
  becomes a returned error, never a process crash; `Router.Handle` recovers a second
  time on the host-call side. Script errors are surfaced to the console/CLI/slog,
  **never fatal to the app.**

### 3. API exposure: auto-mirror the wire surface via one generic bridge

A single global `oblikovati.call(method, argsTable) -> resultTable` marshals the Lua
table → JSON → `Router.Handle` (over the session-goroutine dispatcher) → JSON → Lua
table. The conversion is purely structural, so **no per-method code exists**: every
`api/wire` method + DTO is callable the instant it is registered in `addin/router`,
with zero scripting-package changes. The method strings ARE the public API.
Phase-2 typed convenience tables (`oblikovati.documents.*`, mirroring `api/client`
groups) are thin sugar **auto-derived** from the `api/wire` constants, not hand-written.
`oblikovati.methods()` exposes the registered method list for discoverability.

### 4. Concurrency model: script on a worker, host calls dispatched to the session

The script runs on its **own worker goroutine** (it may loop/compute freely — that
never touches the model). Each `oblikovati.call` blocks the worker and **submits a job
to the existing `addin/dispatch.Dispatcher`** — the same seam add-ins use — which the
GUI frame loop `Drain`s (bounded batch per frame) on the session goroutine, runs
`Router.Handle`, and replies. The call *feels* synchronous to Lua while the **UI never
freezes**. CLI mode (no frame loop, no UI to protect) uses a direct in-proc caller
behind the same `ScriptCaller` abstraction. One script per session (run lock) for
deterministic ordering.

## Consequences

- **Crash-safe by construction.** No cgo ⇒ recoverable panics; the host survives any
  script. Validated by a containment test suite (infinite loop, error, panic, sandbox
  escape, cancellation, memory bomb).
- **Zero-maintenance API coverage.** New contract methods are scriptable for free.
- **No contract change for Phase 1–2.** All new code is GPL-v2 in `/source` under a
  new `script/` package tree; Lua rides the existing `api/wire` surface. Only the
  optional Phase-3 MCP `script.run` adds a contract-first `api/wire`/`api/client` pair.
- **Reuses existing seams** (`addin/router`, `addin/dispatch`, `app.Session`,
  `api/wire`, `api/client`) — no new concurrency primitive.
- **Accepted trade-offs:** interpreter speed (~10-30× slower than LuaJIT, irrelevant
  for an I/O-bound workload); a small instruction-hook overhead (mitigated by count
  granularity); scripts can mutate the model (intended; serialized, undoable via
  commands/transactions, and a future read-only capability gate can filter method
  names at the single `ScriptCaller` choke point).

## Alternatives considered

- **cgo LuaJIT** — rejected: crash surface violates the no-crash requirement; cgo in
  the core violates ADR-0008.
- **Starlark (pure-Go, deterministic)** — viable and named in ADR-0013, but not Lua
  (the product target), and its non-Turing-complete-by-default model is more awkward
  for general automation scripts. gopher-lua gives us real Lua with the same
  pure-Go/no-cgo safety property.
- **Per-method hand-written bindings** — rejected: violates requirement #2 (coverage
  would lag the contract and duplicate DTOs, against ADR-0018).
