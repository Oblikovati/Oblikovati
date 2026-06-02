# ADR-0013 — iLogic-style rules via embedded scripting over the public API

**Status:** accepted · **Context:** design automation (plan M15-F03, iLogic, PBI-149,
flagged XL) lets users author *rules* that automate parameters/features/components.
This is distinct from third-party add-ins (ADR-0003) — rules are authored *inside* a
document, triggered by model changes, low-latency. How do we host them?

> **Amendment (2026-06, [ADR-0018](ADR-0018-apache-api-contract-module.md)).** "The
> public API" the rules drive is now the Apache-2.0 `/api` module (`api/contract`
> in-proc, `api/wire`/`api/client` for the wire); references below to "the gRPC
> contract / api" mean that surface. gRPC remains a deferred transport.

## Decision

Host user rules in an **embedded scripting runtime** (a sandboxed interpreter such
as **Starlark** — deterministic, Go-native, no cgo — or Lua via a pure-Go VM) that
drives the **same public API** the `/api` contract exposes (ADR-0018). Rules are
**in-process automation clients**, triggered by the event bus (core/06), with forms
rendered in ImGui (core/09).

```go
package automation
type Rule struct {
    name     string
    source   string             // user-authored script
    triggers []event.TypeID     // parameter-changed, feature-added, before-save, …
}
// the script's host bindings are the SAME surface as /api: params, features,
// documents, selection — exposed as script globals
func (r *Rule) Run(ctx, doc *doc.Document) error   // edits go through commands (core/06)
```

## Why embedded scripting (not the other extensibility paths)

- **Rules are document data, not installed software.** They live in the part/assembly
  (`model/`, persisted), travel with it, and must run automatically on every relevant
  edit — an out-of-process gRPC add-in per rule (ADR-0003) would be absurd ceremony
  and latency for "if width > 100 then suppress rib."
- **Low-latency, synchronous-feeling triggers.** A parameter edit firing a rule that
  sets three other parameters must feel instant; an in-proc interpreter call is
  microseconds.
- **Safety + determinism.** Starlark is sandboxed (no filesystem/network by default),
  deterministic, and pure Go (no cgo, honors ADR-0002/0008). Users get automation
  without the power to crash or compromise the host — unlike native plugins.
- **It dogfoods the public API.** Rules call the *same* `/api` surface as add-ins
  (ADR-0018). If rules can fully automate a model, the public API is complete — the
  realtime-3d skill's dogfood principle (§12), enforced.

### Note on the realtime-3d skill's "don't embed a scripting runtime" (§8)

That guidance is about **UI markup** — keep *view templates* logic-free, in compiled
code. It does **not** forbid an automation/scripting layer; CAD design automation is
exactly the case where user-authored logic is the feature. We honor the spirit:
**UI** stays compiled (ImGui forms, core/09); **scripting** is confined to the
automation domain and cannot reach the renderer or UI internals — only the public
model API.

## Costs / mitigations

- **Rule ordering & loops** (rule A edits a param that triggers rule B that edits
  A's input) → rules participate in the **dependency DAG** (core/04): a rule declares
  its inputs/outputs; the engine orders rule execution topologically and detects
  cycles (reject → rule health sick), exactly like parameters.
- **Long-running rules** → run on the worker pool with a deadline (like vetoes,
  core/06); a runaway rule is cancelled, not allowed to hang a recompute.
- **Debuggability** → structured logging per rule run; a rule editor with the model
  API surfaced (ImGui, core/09).

## Consequences

- `automation/` depends only on the public API + event bus + command history — it is
  a *client*, adding no privileged backdoor.
- **iPart/iAssembly factories** (M15-F01) are the *declarative* sibling: a member
  table is data-driven generation (no scripting); rules are the *imperative* sibling.
  Both drive the same parameter/feature API. See [apps/01](../apps/01-design-automation.md).
