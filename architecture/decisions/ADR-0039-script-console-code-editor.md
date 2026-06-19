# ADR-0039 — The Script Console code editor is a custom pure-Go core with a thin cgo view

**Status:** Accepted (2026-06-19) · **Builds on / refines:**
[ADR-0028](ADR-0028-embedded-lua-scripting.md) (the embedded Lua console this editor replaces the
input of) and the headless-core/thin-cgo-renderer split used throughout the head. **Touches:** a
new pure-Go editor stack under `script/console/` (`textbuf`, `history`, `lualex`, `complete`,
`editor`, `diag`, `apidoc`), the head widget (`head/ui/code_editor*.go`), native plumbing
(`head/internal/native/` — input-char queue, clipboard, fill-rect/clip, a vendored monospace
font, an editor-key bitmask), and the parse-only checker in `script/gopherlua/check.go`.

## Context

The Script Console (ADR-0028) is how a user drives the live model from sandboxed Lua, but its
editor was a single Dear ImGui `InputTextMultiline` over a byte buffer: no syntax highlighting,
no autocomplete, no line numbers, no error feedback — a poor surface for writing code against a
561-method host API.

Dear ImGui's stock multiline input cannot colour text inline, so a real code editor needs a
different rendering strategy. The options were: (a) vendor a C++ editor widget
(e.g. ImGuiColorTextEdit) and wrap it; (b) keep the stock widget and overlay coloured tokens on
transparent text; or (c) own the editor as a custom core in Go, drawing it onto the window draw
list.

## Decision

A **custom editor core in pure Go**, rendered by a thin cgo widget — option (c).

1. **All editor mechanics live in headless, unit-tested Go** under `script/console/`: a rune-grid
   buffer + caret/selection (`textbuf`), undo/redo with typing-run coalescing (`history`), a Lua
   5.4 tokenizer (`lualex`), an autocomplete engine over the host API tree + Lua vocab
   (`complete`), the editor command model that ties these together (`editor`), debounced syntax
   diagnostics (`diag`), and signature/hover docs (`apidoc`). Every editing behaviour, token
   class, completion-context and diagnostic is asserted without any UI.

2. **The head widget (`head/ui/code_editor*.go`) is a thin shell**: it forwards raw key/char/
   mouse events into editor commands and draws the model (gutter, highlighting, selection, caret,
   popup, squiggles, tooltips) onto the window draw list. It carries no editing rules.

3. **No public-API change.** Autocomplete sources the router's existing wire-method names
   (`oblikovati.methods()`); signature/hover docs are generated at build time from the API
   contract via `script/luadoc` into an embedded table (`apidoc/data_gen.go`), because that
   source is not present at runtime. Syntax checking is a compile-only gopher-lua parse that never
   executes the script.

## Consequences

- The editor is fully testable headlessly (the core packages sit at 82–100% coverage), matching
  the project's push-logic-down mandate; the cgo view is verified by live screenshot capture
  (`head/cmd/editorshot`).
- We own the text-editing mechanics (caret movement, selection, scrolling, IME-less character
  input) rather than getting them free from a C++ widget — more code, but no third-party C++
  editor state living outside our tests and no new dependency to track.
- Rejected (a) because the editor's state would live in untestable C++ and add a dependency;
  rejected (b) because aligning a transparent overlay with the stock widget's caret, scroll and
  selection is fragile and precludes a gutter.
- Follow-ups: route the editor palette through the theme-token system (ADR-0021); the same
  `lualex`/`complete` core can later back the Command Window's input (ADR-0037).
