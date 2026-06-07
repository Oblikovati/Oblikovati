# Lua Scripting — Implementation Plan

**Status:** Phase 1 shipped (PR #75); Phase 2 shipped (GUI Script Console PR #82 +
typed convenience tables); Phase 3 in progress — MCP `scripts.run` shipped
(Oblikovati.API #27 + Oblikovati #85 + AddIns #20), events + persistence outstanding.
All merged to `develop` 2026-06-07. See [§11 Phased rollout](#11-phased-rollout) for the
per-phase status.
**Scope:** Integral, first-party Lua scripting in the GPL-v2 application (`/source`,
i.e. this `oblikovati` module). **Not** an add-in.
**Companion ADR:** [ADR-0028 — Embedded Lua scripting runtime](decisions/ADR-0028-embedded-lua-scripting.md)
**Builds on:** ADR-0013 (embedded scripting drives the public API), ADR-0016
(in-process dispatch / session goroutine), ADR-0018 (Apache-2.0 `api/wire` is the
automation surface), ADR-0008 (cgo confined to the platform edge).

---

## 1. Goal & non-goals

**Goal.** Let a user run a Lua script that drives the live model through the exact
same method surface (`oblikovati/api/wire`) that add-ins and the MCP bridge use, from
the GUI (a script console), from the CLI (`oblikovati-cli script run file.lua`), and
(later) from the MCP bridge — with a sandbox so a buggy or malicious script can never
crash, hang, or exfiltrate from the host.

**Non-goals (this plan).**
- Document-embedded *rules* (iLogic-style, event-triggered, persisted in the part) —
  that is ADR-0013's separate feature; this plan delivers the **runtime + bindings**
  it will later sit on. Persistence/events are Phase 3.
- A new GUI scripting *language editor* with IntelliSense — Phase 2 ships a minimal
  console; a richer editor is a follow-on.
- Any change to the public contract. Phase 1 rides the existing wire surface; no
  `api/*` additions are required (see §9).

---

## 2. Runtime choice — `yuin/gopher-lua`

**Decision: `github.com/yuin/gopher-lua` (pure-Go Lua 5.1 VM, MIT).** Wrapped behind a
project-owned interface so the VM is swappable.

| Criterion | gopher-lua (pure Go) | LuaJIT via cgo |
|---|---|---|
| Crash isolation | **No cgo → a script bug is a Go panic we `recover()`**; the host process is never at risk. | A bug in the C VM or the binding is a real segfault that takes the whole process down. Directly violates requirement #1. |
| Headless-testable core | **Pure Go, `CGO_ENABLED=0`** — fits the cgo-free, headless-tested core (ADR-0008); tests run anywhere. | Needs a C toolchain + LuaJIT on every build/test machine; cgo in the core is exactly what ADR-0008 keeps out. |
| Timeouts / quotas | **Native instruction-count hook** (`SetMyOpcountHook` / `context`-aware `SetContext`) → enforce instruction budget + wall-clock + cancellation. | Possible via `lua_sethook`, but the binding + recovery story is far harder and a runaway C loop is unkillable from Go. |
| Memory cap | `RegistrySize` / `CallStackSize` limits; we add an allocation-tracking guard (§7). | Manual `lua_Alloc` interception. |
| Sandbox | Library registration is **opt-in** — we open *only* the libs we choose (§6). | Same idea, more surface (FFI must be hard-disabled). |
| License | MIT — compatible with GPL-v2. | MIT/Lua — compatible, but moot given crash risk. |
| Performance | Interpreted, ~10-30× slower than LuaJIT. Acceptable: scripts are I/O-bound on `oblikovati.call` (each call hops to the session goroutine), not on tight Lua numerics. | Much faster, irrelevant for an API-driven workload. |

**Conclusion.** The whole point of requirement #1 is "a buggy/malicious script must
NEVER crash the host." A pure-Go VM makes that a `recover()` instead of a segfault,
and matches the cgo-free core. We pay an interpreter-speed cost that does not matter
for an API-call-bound workload. If a future numeric-heavy use case ever needs LuaJIT,
the wrapper interface (§4) lets us swap the engine without touching call sites; the
sandbox/quota guarantees would have to be re-proven for the cgo path.

---

## 3. Package layout (in `/source`, GPL-v2)

A new top-level package tree `script/` (sibling to `addin/`, `app/`, `command/`):

```
script/
  doc.go                      # package doc + SPDX
  engine.go                   # Engine interface (the project-owned VM seam) + Result/Limits types
  gopherlua/
    vm.go                     # gopher-lua–backed Engine impl (the ONLY file that imports gopher-lua)
    sandbox.go                # stdlib allow/deny: open only safe libs, strip the rest
    quota.go                  # instruction-count hook + wall-clock + ctx cancel + mem guard
    convert.go                # Lua table <-> JSON ([]byte) <-> Lua table marshalling
  bridge/
    call.go                   # the `oblikovati.call(method, argsTable)->resultTable` global
    caller.go                 # ScriptCaller: adapts a client.Caller onto the session goroutine
    api_tables.go             # (Phase 2) generated typed convenience tables (oblikovati.documents.*, …)
  runner/
    runner.go                 # ScriptRunner: own goroutine, wires Engine+bridge+limits+ctx, returns Result
  console/                    # (Phase 2) headless console state (history, output buffer) — UI-agnostic
    console.go
```

Rules honoured: one responsibility per file, <500 lines each, the third-party import
(`gopher-lua`) lives behind `script.Engine` and appears in exactly one subpackage
(`script/gopherlua`). Everything except a future head panel is pure Go and
headless-testable.

### 3.1 The engine seam (project-owned interface)

```go
// script/engine.go
package script

type Limits struct {
    Instructions uint64        // hard opcode budget (0 = unlimited, tests only)
    Wall         time.Duration // wall-clock deadline
    MemBytes     int64         // approximate allocation cap
}

type Result struct {
    Stdout   string            // script print() output
    Err      error             // script error (syntax/runtime/quota/panic), nil on success
    Op       uint64            // opcodes executed (for metrics/tests)
    Duration time.Duration
}

// Engine runs one script source against a set of host globals under Limits. It must
// NEVER panic to its caller: a script panic, syntax error, quota breach, or context
// cancellation all come back as Result.Err. Implementations wrap a concrete VM.
type Engine interface {
    Run(ctx context.Context, source string, globals Globals, lim Limits) Result
}

// Globals is what the host injects into the sandbox: the call bridge plus print.
type Globals struct {
    Call  CallFunc   // backs oblikovati.call(method, args)
    Print func(string)
}

type CallFunc func(method string, argsJSON []byte) (resultJSON []byte, err error)
```

This is the only surface the rest of the app sees. Swapping VM = new `Engine` impl.

---

## 4. API exposure — automatic mirroring of the wire surface

**Mechanism: a single generic bridge, `oblikovati.call`, that round-trips through the
existing `Router.Handle`.** The method strings ARE the public API (`documents.activate`,
`sketch.rectangle`, `features.list`, …). The bridge never enumerates them:

```lua
local r = oblikovati.call("sketch.rectangle", { sketch = id, x1=0, y1=0, x2=10, y2=5 })
-- r is the decoded JSON response as a Lua table
```

Data flow for one call:

```
Lua table  ──convert.go──▶  JSON []byte  ──ScriptCaller.Call──▶ dispatch.Submit
   ▲                                                                 │ (session goroutine)
   │                                                          Router.Handle(s, method, req)
   └───────── Lua table ◀──convert.go── JSON []byte ◀── result ◀────┘
```

- `convert.go` does Lua-table ⇄ JSON purely structurally (tables→objects/arrays,
  numbers/strings/bools/nil), so **no per-method code exists**. Any DTO that
  marshals as JSON works automatically.
- `ScriptCaller` is a `client.Caller` (`Call(method, req) ([]byte, error)`) — the
  same interface `api/client` already consumes. It marshals the call onto the
  session goroutine via `dispatch.Dispatcher` (§5).
- The *host wiring* hands the runner a `client.Caller` that closes over
  `func(m, b) { return router.Handle(session, m, b) }` (GUI/CLI in-proc) — identical
  to how `head/cmd/oblikovati-head/addins.go` already wires the router into the
  add-in host.

**Why coverage stays automatic.** When a new wire method + DTO is added (contract-first
per ADR-0018) and registered in `addin/router`, it is *immediately* callable from Lua
as `oblikovati.call("new.method", {...})` with **zero scripting-package changes**. The
bridge is method-name- and schema-agnostic by construction.

**Discoverability helper (cheap, optional):** `oblikovati.methods()` returns the sorted
list of registered method names (the router already exposes its handler map; surface a
read-only `Router.Methods() []string`). This makes the surface introspectable from a
script/console without coupling to any specific method.

### 4.1 Phase-2 typed convenience tables (ergonomic sugar, still auto-derived)

On top of the generic bridge, expose grouped tables mirroring the `api/client` groups:

```lua
oblikovati.documents.activate{ id = "..." }
oblikovati.sketch.rectangle{ ... }
oblikovati.features.list{ ... }
```

These are **thin sugar over `oblikovati.call`**, generated from the `api/wire` method
constants by namespace prefix (`documents.*`, `sketch.*`, …) at startup — *not*
hand-written per method. A small code-gen step (or a reflective walk over the wire
constants) keeps them in lockstep with the contract. If the generic `call` is enough
for Phase 1, these are deferred to Phase 2.

---

## 5. Concurrency / dispatch model

The model is **not** goroutine-safe; `Router.Handle` must run on the **session
goroutine** (ADR-0016, the same constraint add-ins live under).

- The **script runs on its own worker goroutine** (the `ScriptRunner`). It may loop,
  compute, recurse — none of that touches the model.
- Each `oblikovati.call` from the script blocks the worker and **submits a `Job` to
  the existing `dispatch.Dispatcher`**; the GUI frame loop `Drain`s it on the session
  goroutine, runs `Router.Handle`, and replies. The script gets the result
  synchronously (it *feels* synchronous to Lua), but the **UI never freezes** —
  rendering keeps draining one bounded batch per frame.
- This is **the same seam add-ins already use** (`dispatch.Submit` ⇄ `Drain`), so
  scripting introduces no new concurrency primitive — it reuses `addin/dispatch`.
- **Headless / CLI mode** has no frame loop. The runner there runs a tiny pump
  goroutine that `Drain`s the dispatcher in a loop until the script finishes (or the
  session is single-goroutine and we call `Router.Handle` directly behind the
  `ScriptCaller` — for CLI there is no UI to protect, so a direct in-proc caller is
  simplest and correct). The `ScriptCaller` abstraction lets GUI (dispatched) and CLI
  (direct) share the same bridge.
- **Cancellation:** the `context.Context` passed to `Engine.Run` is the same ctx used
  by the quota hook (§7) and by `dispatch.Submit`. Cancelling it (console "Stop"
  button, CLI Ctrl-C, app shutdown) unwinds a blocked call and trips the instruction
  hook so the script stops promptly.
- **One script at a time per session** (a run lock). Concurrent scripts mutating the
  same model is out of scope; the lock makes behaviour deterministic and the dispatch
  ordering obvious.

---

## 6. Sandbox policy (exact stdlib allow/deny)

`gopher-lua` lets us choose which standard libraries to open. We start from **nothing**
and open only safe libs. We do **not** call `lua.NewState()` with default libs;
instead we register a curated set.

**ALLOW (opened):**
- `base` — but with the dangerous globals **removed after open**:
  strip `dofile`, `loadfile`, `load`, `loadstring`, `require`, `collectgarbage`
  (we control GC), `print` (replaced by our captured `print` → output buffer).
  Keep: `assert`, `error`, `pcall`, `xpcall`, `select`, `type`, `tostring`,
  `tonumber`, `pairs`, `ipairs`, `next`, `rawget/rawset/rawequal/rawlen`, `unpack`,
  `setmetatable`, `getmetatable`.
- `table`, `string`, `math` — pure, deterministic, no I/O. (`math.random` seeded
  deterministically per run, or removed if determinism is required for rules.)
- `coroutine` — pure control flow, no I/O. (Optional; include only if needed.)

**DENY (never opened):**
- `os` (clock, time, env, exit, execute, remove, rename, tmpname) — no OS access.
- `io` (files, stdin/stdout, popen) — no filesystem/process.
- `debug` (getinfo, sethook from script, setmetatable bypass) — no introspection/escape.
- `package` / `require` — no loading external code.
- Any FFI (N/A for gopher-lua; explicitly never added).

**The ONLY host capability injected** is the `oblikovati` global table (`call`,
`print`/console output, `methods`, and Phase-2 typed tables). All host power flows
through that one audited door, which itself only reaches the model via `Router.Handle`
on the session goroutine — i.e. scripts can do exactly what an add-in/MCP client can
do, no more.

---

## 7. Resource limits & error handling

**Instruction quota (the primary runaway guard).** Install a count hook
(`L.SetContext(ctx)` + an opcode-count hook) that, every N opcodes, (a) checks
`ctx.Err()` and (b) decrements the instruction budget; on breach it raises a Lua error
that unwinds the VM. This kills `while true do end` deterministically. N is tuned so
the hook overhead is small (§10 risk).

**Wall-clock timeout.** A `context.WithTimeout` (config default, e.g. 5 s GUI / 30 s
CLI, overridable) cancels the ctx; the hook observes it. Belt-and-suspenders with the
instruction budget (a script blocked in a long host call is cut by the dispatch ctx,
not the opcode hook).

**Memory cap.** Bound `CallStackSize` and `RegistrySize` at state creation, and track
approximate allocation via a periodic check in the count hook (string/table growth);
on breach, raise a Lua error. (gopher-lua has no hard allocator cap, so this is
best-effort + the instruction/time guards backstop true runaways.)

**Panic containment (the absolute guarantee).** `Engine.Run` wraps the VM call in
`defer recover()`. Any panic from the VM, a binding, or `convert.go` becomes
`Result.Err`, never a process crash. `Router.Handle` *also* already recovers panics
(see `addin/router/logs_test.go: TestHandleRecoversPanicIntoError`), so a panic inside
a host call is contained at two layers.

**Error surfacing.** Syntax errors, runtime errors, quota/timeout breaches, and
recovered panics all return as `Result.Err` with the offending line/method in the
message (per CLAUDE.md: messages include the offending value). They are written to the
console output / CLI stderr / a slog record — **never fatal to the app**.

---

## 8. Integration points

**Head (GUI) — Phase 2.**
- A **Script Console panel** (ImGui): a source editor pane, a "Run" and a "Stop"
  button (Stop cancels the ctx), and an output/error log pane fed by the console
  state in `script/console`. The panel calls the `ScriptRunner` with the
  dispatched `ScriptCaller`.
- A **"Run Script…"** command (`command/` + `app/commands_*`) that opens a file and
  runs it. Ribbon home: a **Tools tab → Manage panel** (consistent with the
  Inventor-derived ribbon map in `architecture/mapping/`; place alongside add-in/macro
  tooling, not on a modeling panel).

**CLI — Phase 1.**
- `oblikovati-cli script run <file.lua>` (a new `cmd_script.go` + a `script` case in
  `cmd/oblikovati-cli/main.go`'s `run` switch). It opens/creates a `*app.Session`
  (reusing the existing CLI session setup), builds an in-proc `ScriptCaller` over
  `router.Handle`, runs the file under default CLI limits, prints script output to
  stdout and errors to stderr, exits non-zero on `Result.Err`. CGO-free.
- Optional `--doc <file.obk>` to run against an existing document and `--save <out>`.

**MCP bridge — Phase 3 (note only).** The bridge can expose a `script.run` wire
method (contract-first: a `MethodScriptRun` + request/response DTO in `api/wire`, a
router handler that runs source via the `ScriptRunner`, a `client.Scripts()` group)
so an LLM can submit a whole Lua program in one call instead of N tool calls. This is
the only contract-first addition the whole effort needs, and only in Phase 3.

**Reuse, don't duplicate:** `addin/router` (the method surface), `addin/dispatch`
(the session-goroutine seam), `app.Session` (the model owner), `api/wire` (method
names), `api/client` (`Caller`, and the typed groups for Phase-2 sugar).

---

## 9. Contract-first impact (ADR-0018)

- **Phase 1 & 2: none.** Lua rides the existing `api/wire` surface via the generic
  bridge. No new method-name constants, DTOs, or `api/contract` interfaces. All new
  code is GPL-v2 in `/source` under `script/`.
- **Phase 3 (only if MCP `script.run` is wanted):** add `MethodScriptRun` + DTOs in
  `api/wire` and a `client.Scripts()` group in `api/client` (Apache-2.0) *first*, then
  the router handler in `/source`. This is the standard two-part flow.

Every new exported `.go` file in `/source` carries `SPDX-License-Identifier:
GPL-2.0-only` (run `scripts/add-spdx-headers.py`).

---

## 10. Testing strategy (headless, against `*app.Session`)

All Phase-1/2 logic is pure Go → fully unit-testable with no GPU and no cgo.

**Model-effect tests (`script/runner` + `script/bridge`).**
- Run a script that calls `documents.*` / `sketch.*` / `parameters.*` against a real
  `*app.Session` (in-proc `ScriptCaller` over `router.Handle`) and assert the model
  changed (e.g. a rectangle exists, a parameter equals the set value). This proves the
  table↔JSON↔router round-trip end to end.

**Containment tests (the safety spec — the heart of requirement #1).**
- *Infinite loop* (`while true do end`) → returns `Result.Err` (quota/timeout), host
  survives, asserts within a bounded wall-clock.
- *Script error* (`error("boom")`, nil index) → `Result.Err` carries line + message,
  no panic escapes.
- *Forced panic path* (a fake `CallFunc`/Engine that panics) → recovered into
  `Result.Err`; a follow-up run on the same session still works (process intact).
- *Sandbox escape attempts* → `os`, `io`, `require`, `debug`, `loadstring` are all
  `nil` in the script env (table-driven test asserting each global is absent).
- *Cancellation* → cancel the ctx mid-run; `Run` returns promptly with a
  cancellation error.
- *Memory* → an allocation-bomb script trips the mem/instruction guard, not the host.

**Conversion tests (`script/gopherlua/convert`).** Round-trip nested
table↔JSON↔table: objects, arrays, numbers, bools, nil, deep nesting; reject cyclic
tables with a clear error.

**CLI test (`cmd/oblikovati-cli`).** `script run` on a fixture `.lua` creates the
expected model / exits 0; a failing script exits non-zero with the error on stderr.

Mocks are named fakes (e.g. `fakeCaller`, `panicEngine`) per CLAUDE.md, not inline
stubs.

---

## 11. Phased rollout

**Phase 1 — runtime + generic bridge + sandbox + CLI (no UI, no contract change).
✅ SHIPPED (PR #75, merged to `develop` 2026-06-07).**
1. ✅ `script.Engine` interface (`script/engine.go`) + `script/gopherlua` impl
   (`vm.go`/`sandbox.go`/`quota.go`/`convert.go`/`host.go`): the only importer of
   `yuin/gopher-lua v1.1.1`.
2. ✅ `script/bridge` — the generic `oblikovati.call`/`oblikovati.methods` door
   (`call.go`/`host.go`) + a `DirectCaller` (CLI, synchronous) and a
   `DispatchedCaller` (head, over `addin/dispatch`), both `client.Caller`.
3. ✅ `script/runner` (worker goroutine, one-script-per-session run lock, default
   CLI/GUI `Limits`, ctx).
4. ✅ `oblikovati-cli script run <file.lua> [--doc] [--save]`.
5. ✅ Full headless test suite (§10): model-effect round-trip, conversion, and the
   complete containment spec (loop/error/panic/denied-globals/cancel).
6. ✅ ADR-0028 → Accepted.

**Implementation note — instruction quota.** `gopher-lua` exposes **no public opcode
hook**, and its `SetMx` memory guard calls `os.Exit` on breach (which would crash the
host, violating requirement #1), so neither is used. The enforced runaway guard is the
**wall-clock `context`** (checked per opcode by the VM's context-aware main loop),
backstopped by a **bounded registry / call-stack** for the memory cap.
`Limits.Instructions` is kept in the contract (recorded in `Result.Op`) as the seam a
future hooked VM — or a swapped `Engine` — would wire a real opcode counter onto;
`script/gopherlua/quota.go: instructionBudget` is its single anchor point.

**Phase 2 — typed sugar + GUI console. ✅ SHIPPED.** The GUI Script Console shipped in
PR #82 and the typed convenience tables completed it (both merged to `develop`
2026-06-07):
1. ✅ `oblikovati.documents.*` / typed tables auto-derived from `api/wire` method names
   (`oblikovati.methods()`): each `group.method` becomes `oblikovati.group.method{…}`,
   thin sugar over `oblikovati.call`, in lockstep with the contract by construction.
2. ✅ `script/console` state (Console + async Controller) + the head Script Console
   panel (source editor, Run/Stop/Clear, output pane, status) on the **Manage ▸
   Scripts** ribbon panel; Stop/cancel. The panel drives the `DispatchedCaller` over
   the same router + dispatcher add-ins use, so a looping script never freezes the UI.
3. ✅ output/error pane wiring (streamed `print()` lines + last-run status; the
   `oblikovati.methods()` discoverability primitive shipped in Phase 1).
4. ✅ UI e2e tests — headless `script/console` unit tests + in-window cgo render/run
   tests that open the real Vulkan window and assert streamed output reaches the panel.

**Phase 3 — events, library, persistence, MCP. 🔄 IN PROGRESS (MCP `scripts.run`
shipped; events + persistence outstanding).**
1. ⏳ Event/callback hooks so scripts can subscribe to the event bus (foundation for
   ADR-0013 rules: `on("parameterChanged", fn)`). The hard part is a *persistent* VM
   lifecycle (the one-shot runner closes the VM after the main chunk), so this is a
   dedicated subsystem, not a small slice.
2. ⏳ A bundled script library / examples; saved scripts (console Open/Save + a
   "Run Script…" command); (later) document-embedded rules persisted in the model
   (ADR-0013 territory; overlaps the .obk persistence effort).
3. ✅ Contract-first `scripts.run` wire method + `client.Scripts()` so the MCP bridge
   can run whole programs in one call (Oblikovati.API #27 + Oblikovati #85 + the
   `run_script` bridge tool, AddIns #20; merged 2026-06-07). The handler runs the source
   **inline** through the sandboxed runtime with a host door that calls `router.Handle`
   **re-entrantly** (no lock ⇒ deadlock-free, inner mutations still emit `edit.committed`);
   a per-run wall-clock (default 10s, cap 60s) bounds a runaway. Live-verified via the
   bridge driving the model. This was the only contract-first addition the effort needed.

---

## 12. Risks & mitigations

- **Instruction-hook overhead.** A per-opcode hook is costly. *Mitigation:* run the
  hook every N opcodes (count granularity), not every op; benchmark to pick N that
  keeps overhead <~5% while still cutting a runaway within a few ms. Workload is
  API-bound, so Lua-side opcode rate is modest anyway.
- **A long script blocking the session goroutine.** Only the *host call* runs on the
  session goroutine, and the frame loop drains a **bounded batch per frame**
  (`addInDrainPerFrame`, already 32), so even a chatty script can't starve rendering.
  A single *slow handler* could stall one frame — same risk add-ins already have; the
  dispatch timeout backstops it.
- **Exposing mutating methods.** Scripts can do anything an add-in/MCP client can
  (create/delete/modify). That is the intended power. *Mitigation:* it all goes
  through `commands`/transactions (undoable), the run lock serialises it, and a future
  capability gate (a read-only mode for untrusted scripts) can filter method names at
  the `ScriptCaller` before dispatch — the single choke point makes that trivial later.
- **Determinism.** `math.random`/any clock would make rules non-deterministic.
  *Mitigation:* seed deterministically per run (or drop `math.random` for the rules
  use case); `os`/`io` already denied. Document the guarantee.
- **gopher-lua performance.** ~10-30× slower than LuaJIT for pure numerics.
  *Mitigation:* accept it (workload is I/O-bound on host calls); the `Engine` seam
  lets us swap to a faster VM later **only if** we can re-prove the crash/quota
  guarantees on that path.
- **Two layers of recover masking real bugs.** *Mitigation:* recovered panics are
  logged (slog, with stack) before being converted to `Result.Err`, so genuine host
  bugs are still visible in observability.

---

## 13. Key files / packages (summary)

- New: `script/` (engine seam), `script/gopherlua/` (the only gopher-lua importer +
  sandbox/quota/convert), `script/bridge/` (the auto-mirroring `oblikovati.call` +
  `ScriptCaller`), `script/runner/`, `script/console/` (Phase 2),
  `cmd/oblikovati-cli/cmd_script.go`, head Script Console panel (Phase 2).
- Reused unchanged: `addin/router` (`Router.Handle`), `addin/dispatch` (`Dispatcher`),
  `app.Session`, `api/wire` (method constants), `api/client` (`Caller` + typed groups).
- Contract-first additions: **none** until Phase 3's optional `script.run` wire method.

---

*Plan author note:* this plan extends ADR-0013's already-recorded intent ("Lua via a
pure-Go VM" driving "the same public API") into a concrete, sandboxed, headless-tested
runtime. The companion decision is recorded in ADR-0028.
