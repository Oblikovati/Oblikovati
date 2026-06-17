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
   [ADR-0018](ADR-0018-apache-api-contract-module.md)): submit a line, get back the produced output, the
   active command's next prompt, and whether more input is awaited — so an add-in or MCP tool runs
   the same REPL headlessly.

5. **The static vocabulary is multi-letter only; single letters belong to the keybinding
   editor** (amended 2026-06-17). Every word in the built-in vocabulary (`app/cmdline`) is a
   multi-letter AutoCAD command name or alias — there are no single-letter words. A bare single
   letter typed at the command line does not resolve to a command; single-letter activation is a
   personalised Shift/Control chord configured in the keybinding editor. The vocabulary doubles as
   the source of a generated command manual: each entry carries a one-line summary and a usage
   example, rendered to [`architecture/mapping/autocad-command-map.md`](../mapping/autocad-command-map.md)
   by `Vocabulary.RenderManual` and held in sync by `TestCommandManualInSync`.

   On the **keyboard** the same rule holds: a bare single letter is never a default and never
   dispatches a command. `bindables` drops bare single-letter alias chords, and `Bindings.SetChord`/
   `SetAlias` reject a bare single letter — the user binds single-letter shortcuts as Shift/Control
   chords. Commands ship sensible **predefined** shortcuts via `Command.WithDefaultChord` (a full
   chord like `Ctrl+N`): the common set is Save `Ctrl+S`, New `Ctrl+N`, Close `Ctrl+W`, Undo
   `Ctrl+Z`, Redo `Ctrl+Y`/`Ctrl+Shift+Z`, plus the F-keys (Help `F1`, Previous View `F5`, Home
   `F6`). The platform "command" modifier maps cross-platform: `Ctrl` on Windows/Linux and `Cmd`
   (Super) on macOS (`obk_ig_key_ctrl` ORs in `KeySuper` under `ConfigMacOSXBehaviors`).

6. **ESC is the universal cancel** (amended 2026-06-17): it drops any pending command-line
   question, closes the active feature/tool creation or editing window, clears the selection when
   nothing is in progress, and always returns keyboard focus to the command-window input
   (`Session.RequestCommandInputFocus` → the head's `TakeCommandInputFocus`).

## Consequences

- One surface to learn, one feedback log, and one command path shared by the UI and the API.
- The vocabulary is many→one and flat, so a word resolves to a single action; context is enforced
  by each command's own enable rule (e.g. 2D `FILLET` vs 3D `FILLETEDGE`). The generated manual at
  [`architecture/mapping/autocad-command-map.md`](../mapping/autocad-command-map.md) lists every
  command with its aliases, description and example; `TestEveryVocabularyWordResolves` pins that
  every word resolves to a real action and `TestCommandManualInSync` keeps the manual current.
- "Full REPL" is honest about scope: coordinate/value steps are fully typed; pick-dependent steps
  (selecting an existing edge to fillet) are still picked in the viewport, co-driven with the line.
- Retiring the modal/toast surfaces means a blocking yes/no is now an inline question; code that
  posted prompts is unchanged (it still calls `ShowPrompt`), but the answer arrives via the
  command line.
- Known follow-ups: the command line is the focused input by default but is not yet a sticky
  keyboard sink (forcing focus every frame broke viewport drag-orbit — it disables ImGui mouse
  hover); Revolve's value path and AutoCAD polyline `Close`/`Undo` chaining are not yet wired.
