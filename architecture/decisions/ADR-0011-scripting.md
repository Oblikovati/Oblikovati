# ADR-0011: Lua scripting and rule execution

Date: 2026-10-10. Status: Accepted for the Rust target design.
Implementation evidence: pending R7; no Rust Lua dependency is selected.

Boundary: user-authored automation, execution limits, and model calls.
Owner: script module; session/automation owns edits; shell owns the editor view.

## Context

Existing scripts target Lua 5.1 and the public method contract. Go-specific
claims that a pure-language VM can never crash its process are not proof for
Rust, native Lua, allocation failure, or stack exhaustion.

## Decision

Preserve the baseline Lua 5.1 behavior behind an application-owned execution
interface. Select a runtime only after language compatibility, limits,
cancellation, containment, license, and toolchain checks. Do not choose native
Lua/LuaJIT merely because a Rust wrapper exists. If an in-process engine cannot
meet the containment requirements, prove an isolated worker design before
integration. The observable script contract remains the same.

Scripts drive the public automation surface through a single structural bridge.
Use API method declarations and wire shapes for `oblikovati.call`, supported
method discovery, typed convenience groups, completion, and generated help.
Keep JSON conversion at this interpreter interface; do not hand-maintain a
second DTO schema or expose model pointers. Host method calls use
[ADR-0002](ADR-0002-api-and-addin-host.md) and session transactions.

Execute scripts away from the UI owner. Host calls enqueue work for the session
owner; the CLI must service the same contract without a GUI loop. Start with
one script execution per session. Stop, close, timeout, and shutdown must stop
new host calls and terminate bounded interpreter work. Preserve the existing
effect of commands already committed; cancellation does not imply rollback
unless the script used an explicit transaction that is aborted.

Open only the supported standard-library subset. Do not expose filesystem,
network, OS process, dynamic loading, debug hooks, or FFI through VM libraries.
Host API calls are the deliberate capability surface, including any file or
external operations that surface already permits. This is a constrained VM,
not a claim that a script with the full host API has no external effects.

## Interfaces and invariants

- Enforce instruction, time, memory, stack, output, and host-call bounds with
  explicit diagnostics. A Rust `catch_unwind` alone cannot contain aborts,
  memory exhaustion, or faults in a native runtime.
- Keep deterministic random seeding where the baseline requires it. State
  required numeric/string/table conversion, null, array, and error behavior;
  Lua numbers must not silently corrupt API integer identifiers.
- No script waits on a callback that requires its own blocked session owner.
  Preserve accepted-request timeout behavior; do not reissue a timed-out edit.
- Store document rules as source plus supported trigger metadata, not VM
  memory. When rule triggers are implemented, order them through explicit
  dependencies, reject cycles, and bound repeated event-driven execution.
  Legacy planned triggers are not assumed to have shipped.
- Parse-only editor checks never execute a script. Runtime grammar, editor
  diagnostics, and generated API documentation must use the same pinned
  contract. UI/editor mechanics follow [ADR-0009](ADR-0009-ui-and-selection.md).

## Alternatives and consequences

Changing the script language would break user source. Per-method bindings
would drift as the API grows. An unrestricted native VM would not meet the
existing containment goal. Reject these options. A proved isolated worker may
add IPC and lifecycle cost; an in-process runtime needs evidence rather than
a stronger safety claim than its failure modes support.

## Legacy disposition

Consolidate Oblikovati ADR-0013 and 0028 plus runtime/documentation parts of
0039 at the [fixed revision](legacy-map.md#source-baseline). Lua replaces
the early Starlark candidate. Omit gopher-lua, goroutine, and Go panic-recovery
mechanics. Keep the shared API, execution limits, cancellation, and editor
features while rejecting absolute no-crash claims without proof.

## Validation required

Use Go `script/engine.go`, VM/bridge tests, console fixtures, and CLI scripts.
Run normal scripts plus infinite loops, recursion, memory/output exhaustion,
forbidden library access, malformed values, Stop, and shutdown during a host
call. Test committed and aborted transaction outcomes and a responsive live
UI. Completion/help inventories must match the implemented host and API pin.
