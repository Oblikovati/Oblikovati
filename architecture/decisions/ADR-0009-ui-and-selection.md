# ADR-0009: Desktop UI, tools, and selection

Date: 2026-10-10. Status: Accepted for the Rust target design.
Implementation evidence: pending R1V/R4/R7.

Boundary: desktop input and presentation of application state.
Owner: shell/platform modules; session owns tools, selection, and commands.

## Context

The Go shell uses Dear ImGui, shared commands, semantic themes, and a common
selection engine. Replacing those workflows during a language port would add
product changes and make behavior comparisons harder.

## Decision

Retain Dear ImGui with docking for application panels and dialogs. Keep a
narrow native binding where required. Port UI behavior and editor state to
Rust; select the binding only after its lifetime, input, DPI, platform, and
toolchain checks pass. Rendering and shared presentation ownership follow
[ADR-0008](ADR-0008-vulkan-renderer.md).

The shell presents session state and issues commands. It does not mutate
feature recipes, resolve kernel topology, or own undo. Keep ribbon, browser,
property editors, tables, tool previews, and document tabs separate from
document state. Use explicit typed field metadata where Go used reflection.

One sketch-host capability supports create/edit/finish in parts and assemblies.
Tools declare the accepted selection kinds for their current step and expose
their current picks through one interface. The session applies the tool filter,
updates it after each step, and restores the ambient filter on completion or
cancel. The declared tool filter takes precedence while the tool runs.

One selection engine handles candidate priority, preselection, selection, and
tool picks. The viewport renders one defined highlight per kind, including
visible face fill/outline. New selectable kinds need visible highlight checks;
an empty rendering implementation is not support. Presentation IDs resolve
through the renderer's generation-aware mapping to model references.

## Interfaces and invariants

- The Command Window is the shared command-entry and user-feedback surface.
  Its parser, bounded history, alias resolution, prompts, and tool progression
  are headless session behavior also used by the API. Text entry and picking
  advance the same tool. Generate the command manual from that registry.
- Keep multi-letter built-in command words, full shortcut chords, and the
  baseline modifier behavior on macOS. Escape cancels the current question or
  tool, otherwise clears selection, and requests command-input focus. Focus
  must not consume viewport drag or text input accidentally.
- Preserve the custom editor's headless buffer, selection, undo, completion,
  and diagnostic behavior. Its view forwards input and draws state. Grammar
  and parse-only diagnostics match the runtime selected under
  [ADR-0011](ADR-0011-scripting.md), not the legacy Lua 5.4 tokenizer label.
- Themes use API semantic tokens and Blender XML, with a full custom token
  snapshot and preservation of unmapped XML content. Missing required mappings
  are named errors. Built-in themes are embedded assets; selected-theme
  preferences remain separate from document recipes.
- Preserve primary, secondary, tertiary, and background SVG icon roles. Cache
  role masks and recolor on theme revision. GPU texture retirement follows
  ADR-0008 rather than a fixed number of frames. Disabled style is applied once.
- Keep fonts, clipboard, dialogs, and OS input behind platform interfaces.
  Use bundled fonts in deterministic tests and preserve resource provenance.
- User feedback and script output are distinct from structured JSON diagnostic
  logs. Public API selection sets use shared contract types; interactive tool
  internals are not new public methods without a contract-first requirement.

## Alternatives and consequences

A new UI toolkit or a native editor that owns all editing state would expand
the port and weaken headless comparison. Per-tool filters and custom highlight
accessors would reintroduce known selection defects. Reject these designs.
Owning a Rust editor and input model costs implementation work, but keeps its
behavior testable through the same commands used by the desktop.

## Legacy disposition

Consolidate Oblikovati ADR-0004, 0021, 0032, 0033, 0036, 0037, 0039, and
0041 at the [fixed revision](legacy-map.md#source-baseline). Keep semantic
theme state; Blender XML replaces the old YAML theme-file choice. Replace Go
editor/cgo mechanics and reflection with Rust interfaces. Omit no-op highlights,
native enum-index assumptions, and the claim that screenshots cannot be inspected.

## Validation required

Use Go `head/ui`, sketch tools, selection tests, `app/cmdline`, `theme/`, icon
assets, and `script/console`. R4/R7 must test type-or-pick flows, cancel/undo,
part and assembly sketch hosts, focus, Unicode input, clipboard, DPI, and
theme round trips. Drive representative tools through MCPBridge in the
foreground and inspect captures on light/dark themes and target platforms.
