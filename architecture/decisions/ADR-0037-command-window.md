# ADR-0037 — A Command Window is the single command-entry and feedback surface

**Status:** Accepted (2026-06-14) · **Builds on / refines:**
[ADR-0028](ADR-0028-embedded-lua-scripting.md) (the in-app console pattern the panel reuses)
and the binding engine (M05-F17). **Touches:** a new headless engine (`app/cmdline/`,
`app/command_line.go`, `app/command_driven.go`), the binding engine (`app/bindings.go`), the
messaging centers (`app/message_center.go`, `app/balloon_tips.go`, `app/prompt_center.go`), the
public API (`oblikovati.org/api` — `commandLine.submit`), and the head chrome
(`head/ui/command_window.go`, retiring `head/ui/messaging_surfaces.go`).

## Context

Oblikovati scattered user feedback across four head surfaces — the status-bar notice, balloon
toasts, the message center, and prompt modals — and offered only a minimal one-shot alias box for
typed commands. Users coming from AutoCAD expect a single, always-present command line that both
**drives commands** (type a verb, answer its prompts) and **shows all feedback** as a rolling log.

We also want commands to be drivable headlessly — by add-ins, MCP tools, and tests — through the
same path the UI uses, not a parallel one.

The options were: (a) keep the scattered surfaces and bolt a richer command box onto them;
(b) build a command line as a head-only widget; or (c) build a transport-agnostic command-line
*engine* in the application layer and render a thin view of it in the head.

## Decision

A **Command Window**: a docked, always-on, rolling-history panel with a persistent input line,
backed by a command-line engine in the application layer.

1. **Pure primitives** live in `app/cmdline/` (no app/UI deps, fully unit-tested): a bounded
   scrollback + recall history, an input parser (absolute/relative/polar coordinates, distances,
   keyword options), and a many-to-one **AutoCAD command vocabulary** mapping command words and
   aliases onto Oblikovati action ids. The vocabulary is a default layer **beneath** the user's
   own aliases in the binding engine.

2. **The engine** (`app/command_line.go`) is an AutoCAD-style REPL over the existing command and
   tool model. Submitting a line starts a command (resolved through the binding engine) or feeds
   the active tool one parsed token, advancing its prompt. Tools opt into text input with the
   one-method `CommandDriven` capability; the command line and the viewport co-drive the same tool
   step ("type or pick"). It lives in the app layer — not the head — so the UI and the API drive
   one engine.

3. **The Command Window is the single feedback surface.** Every messaging path — the status
   notice, message-center entries, balloon tips, and prompt questions — funnels into the engine's
   scrollback; prompts become inline command-line questions answered by the next submitted line.
   The standalone toast / message-center / prompt-modal windows are retired. The underlying
   centers remain as the data sources behind the wire API; only their separate windows are gone.

4. **The command line is on the public API** (`commandLine.submit`, contract-first per
   [ADR-0018](ADR-0018-public-api-split.md)): submit a line, get back the produced output, the
   active command's next prompt, and whether more input is awaited — so an add-in or MCP tool runs
   the same REPL headlessly.

## Consequences

- One surface to learn, one feedback log, and one command path shared by the UI and the API.
- The vocabulary is many→one and flat, so a word resolves to a single action; context is enforced
  by each command's own enable rule (e.g. 2D `FILLET` vs 3D `FILLETEDGE`). The map is documented
  in [`architecture/mapping/autocad-command-map.md`](../mapping/autocad-command-map.md) and pinned
  by a test that every word resolves to a real action.
- "Full REPL" is honest about scope: coordinate/value steps are fully typed; pick-dependent steps
  (selecting an existing edge to fillet) are still picked in the viewport, co-driven with the line.
- Retiring the modal/toast surfaces means a blocking yes/no is now an inline question; code that
  posted prompts is unchanged (it still calls `ShowPrompt`), but the answer arrives via the
  command line.
- Known follow-ups: the command line is the focused input by default but is not yet a sticky
  keyboard sink (forcing focus every frame broke viewport drag-orbit — it disables ImGui mouse
  hover); Revolve's value path and AutoCAD polyline `Close`/`Undo` chaining are not yet wired.
