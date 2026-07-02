# ADR-0016 — In-process shared-library add-ins (C ABI) + MCP automation bridge

**Status:** accepted (user decision, 2026-06) · **Amends:** [ADR-0003](ADR-0003-extensibility-hybrid-rpc.md)
(third-party add-in transport) · **Implemented by:** `oblikovati-mcp-bridge` (the
first add-in).

> **Amendment (2026-06, [ADR-0018](ADR-0018-apache-api-contract-module.md)).** The
> in-process JSON method contract described here is now the typed, Apache-2.0
> `api/wire` (method-name constants + DTOs) and `api/client` (a `Transport` + typed
> client). The C-ABI `ObkHostCall` is unchanged — it is the `Transport` an add-in
> supplies — but the bridge and the host no longer re-declare DTOs; they share
> `/api`. A closed-source add-in links only `/api`, never the GPL `/source`.

> **Editor's note (2026-07, #1661 / M40 audit D4).** Paths in this ADR predate the
> repo-root migration: the GPL application module written here as `/source` now
> lives directly at the repository root (`kernel/`, `model/`, `app/`, `head/`, …),
> and the Apache-2.0 `/api` module is the sibling `Oblikovati.API` repository.
> The record below is preserved as written.

## Context

ADR-0003 chose **out-of-process gRPC** for third-party add-ins and explicitly
*rejected* Go `plugin`/`.so` loading ("no Windows support, brittle exact-toolchain
matching, shared crashes"). When we built the first real add-in —
`oblikovati-mcp-bridge`, which exposes the model API over **MCP** so an LLM can
drive the live app and dogfood the API — the gRPC `api/` layer did not yet exist,
and the product decision was to ship the add-in **in-process** so it drives the
running GUI in real time.

## Decision

1. **Add-ins are loaded in-process as C-shared libraries** (`.so`/`.dll`/`.dylib`,
   built `go build -buildmode=c-shared`) discovered in an `addins/` folder beside
   the executable (or `OBK_ADDINS_DIR`). The host scans, loads, and activates each
   at startup via the existing `app.AddIn` lifecycle.

2. **The host↔add-in boundary is a small C ABI**, not Go types. A Go c-shared
   library loaded into the Go host runs its **own Go runtime** (two runtimes, two
   GCs, one process), so no Go pointers may cross. The contract
   (`add-in/include/oblikovati_addin.h`) is therefore data-only:
   - add-in exports `ObkAddInId/Manifest/Activate/Deactivate/Notify/Free`;
   - the host passes one callback, **`ObkHostCall(method, reqJSON) → respJSON`**,
     plus a free for host-owned buffers.

3. **The wire is in-process JSON-RPC.** The JSON `method` strings ARE the public
   automation API: `commands.*`, `documents.*`, `parameters.*`, `model.*`,
   `sketch.*`, `features.*` (see the add-in README and `oblikovati://schema/*`
   resources). The host runs each on the session goroutine and returns JSON.

4. **The first add-in speaks MCP** (Model Context Protocol) over streamable
   HTTP/SSE (official Go SDK), translating MCP tool/resource calls ⇄ the JSON
   method contract, and serving docs + reflected schemas as MCP resources so any
   LLM can learn Oblikovati.

## Why this, and why it diverges from ADR-0003

- **Real-time, in-GUI dogfooding.** The goal is to manipulate the *visible*
  running application and exercise the API end to end; in-process hosting gives
  that directly, without first building the gRPC `api/` layer (M16).
- **Cross-platform `.so` + `.dll`.** `c-shared` (unlike Go `plugin`) emits a
  native shared library on every OS, so the loader is portable in principle (the
  Windows `LoadLibrary` path is a fast-follow; Linux/macOS `dlopen` is done).
- **The transport is swappable.** The JSON method contract is transport-agnostic:
  the same `router` can be re-fronted by gRPC (ADR-0003) or a local socket with no
  change to handlers. So gRPC is **deferred, not abandoned** — the C-ABI function
  pointer is just the first transport, and the router *is* the future `api/`
  surface in spirit.

## Consequences (trade-offs accepted)

- We accept the costs ADR-0003 named for in-process loading: the add-in **shares
  the host address space** (a crashing add-in can take the host down — no crash
  isolation; see *Known gap — add-in panic recovery* below), and the c-shared
  library is coupled to a compatible toolchain. These are acceptable for a
  first-party, trusted automation add-in; untrusted add-ins would argue for the
  out-of-process path.
- **Thread safety** is handled by a dispatch queue: add-in calls arrive on the MCP
  server's goroutines and are marshaled onto the host's single session goroutine
  (drained once per frame). The session is never touched concurrently.
- **Windows `.dll` loading** is stubbed (returns a clear error) pending the
  `LoadLibrary` trampoline; manifest **capability/permission enforcement** is
  recorded but not enforced yet.
- The risky two-runtime assumption was validated by a disposable spike before
  building on it (`experiments/cshared-seam/`, gitignored).

### Known gap — add-in panic recovery (`defer recover`), NOT YET IMPLEMENTED

The **inbound** path (add-in → host) is guarded: `addin/router.Handle` wraps every
handler in `defer recover()`, turning a kernel/handler panic triggered by an add-in
request into a returned error (method + value + stack), and the dispatcher copies the
request bytes and confines model access to the single session goroutine. A buggy
add-in *request* cannot corrupt the model or panic-kill the host.

The **outbound** path (host → add-in) is **not** guarded, and the add-in SDK has no
boundary recovery:

- the host invokes the add-in's `Activate`/`Deactivate`/`Notify` exports through raw
  cgo trampolines (`head/internal/addinhost/dl_unix.go`) on the frame goroutine, with
  no `recover` and no timeout;
- the add-in SDK's exported functions and the goroutines it spawns
  (`oblikovati-mcp-bridge/export.go`) carry **no `defer recover()`**.

Because this is one OS process with two Go runtimes, the host cannot recover a panic
raised in the add-in's runtime: an unrecovered panic *anywhere* in an add-in's own
code crashes the whole process, and a hang in `Activate`/`Notify` freezes the frame
loop.

**Gap to close (cheapest in-process win):** the add-in SDK must `defer recover()` in
**every exported function and every spawned goroutine**, converting an add-in panic
into an error return / `OBK_ERR` instead of a process kill. This does **not** cover
segfaults, stack overflows, or hangs — those require the out-of-process path (a
subprocess speaking the same JSON C-ABI over a pipe/socket), the real fix for
untrusted add-ins. **Deferred** while add-ins remain first-party and trusted; tracked
here as a known gap. (For contrast, the embedded Lua runtime — ADR-0028, proposed —
is panic-recoverable in the host's *own* runtime precisely because it avoids this
foreign-runtime crash surface.)

## Shape / layering

```
add-in/include/oblikovati_addin.h        # the C ABI contract
add-in/oblikovati-mcp-bridge/            # the c-shared add-in: C exports + MCP server
source/addin/{dispatch,router,opregistry,events,modelaccess}
                                         # pure-Go host logic (no cgo, headless-tested)
source/head/internal/addinhost/          # cgo loader (dlopen) + //export ObkHostDispatch shim
```

The host model logic is pure Go and headless-testable; cgo is confined to the
loader/shim (the platform edge, [ADR-0008](ADR-0008-cgo-boundary.md)); the MCP SDK
and HTTP deps live only in the add-in module, out of the cgo-free core.

## Relationship to ADR-0003

ADR-0003's **in-process registries for first-party features** are unchanged. Its
**out-of-process gRPC for third-party add-ins** is amended: the realized mechanism
for the first add-in is in-process c-shared + C ABI, with gRPC kept as a future
transport behind the same method contract. See the amendment note on ADR-0003.
