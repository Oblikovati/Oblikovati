<!-- SPDX-License-Identifier: GPL-2.0-only -->
# Desktop UI parity plan

The Go desktop UI is the reference implementation. This document records the required Rust
port surfaces and the evidence source for each item.

## Chrome

| Surface | Go source | Rust state | Port requirement |
|---|---|---|---|
| Application menu and quick-access toolbar | `head/ui/chrome.go`, `head/ui/qat.go` | Missing | Add deferred New, Open, Save, Undo, and Redo actions. |
| Contextual ribbon | `head/ui/chrome_ribbon.go` | Basic four-tab buttons | Port panel groups, command enablement, contextual Sketch selection, and compact overflow behavior. |
| Document tabs | `head/ui/chrome_doctabs.go` | Missing | Display active documents and route selection and close requests through the owner. |
| Status bar | `head/ui/statusbar_text.go` | Missing | Display command prompt, selection summary, and model-health text. |
| Dock layout | `head/ui/chrome.go` | Fixed windows | Replace fixed windows with a persisted dock layout. |

## Docked workspaces

| Surface | Go source | Rust state | Port requirement |
|---|---|---|---|
| Model browser | `head/ui/browser_view.go` | Missing | Use the existing `model.tree` API to display document, Origin, Parameters, Sketches, Features, and Bodies. |
| Properties | `head/ui/property_panel.go` | Missing | Show selected entity and feature properties; use existing feature queries and parameter edits. |
| Command window | `head/ui/command_window.go` | Prompt-only | Show owner notices and command feedback as a scrollable history. |
| Parameters | `head/ui/parameters_window.go` | Implemented | Dock it and expose it from View and ribbon commands. |

## Viewport and tools

| Surface | Go source | Rust state | Port requirement |
|---|---|---|---|
| Navigation bar and view controls | `head/ui/navigation_bar.go`, `head/ui/viewcube_*.go` | Keyboard navigation only | Port visible Home, previous-view, projection, and standard-view controls. |
| Sketch environment | `head/ui/sketch_ribbon_*.go`, `head/ui/sketch_hud.go` | Cube-only controls | Port contextual Sketch ribbon and tool feedback. |
| Feature panels | `head/ui/feature_edit_dialog.go`, `head/ui/extrude_dialog.go` | Cube extrude only | Port editable feature panels through the host feature API. |

## Acceptance order

1. Replace fixed shell windows with Go-style chrome and dock layout.
2. Port the live Model browser from `model.tree`, then document tabs and status bar.
3. Port command window and Properties panel with owner-fed snapshots.
4. Port contextual Sketch and 3D Model ribbon panels and feature dialogs.
5. Verify native frames against Go UI layout and command behavior.
